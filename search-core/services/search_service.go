package services

import (
	pb "agent-boot/proto/generated"
	"context"
	"slices"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/ds"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"github.com/ollama/ollama/api"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// search parameters.
const (
	vectorSearchWeight = 0.7
	textSearchWeight   = 0.3

	vecK  = 20 // Number of vector search results to return
	textK = 20 // Number of text search results to return

	textSearchIndexName   = "chunkIndex"
	vectorSearchIndexName = "chunkEmbeddingIndex"

	vectorSearchFieldName = "embedding" //chunkAnnModel can have multiple vector fields, but we use only one for search

	maxChunks = 20
)

type SearchService struct {
	pb.UnimplementedSearchServer
	mongo        *mongo.Client
	ollamaClient *api.Client
	ccfgg        *appconfig.AppConfig
}

func ProvideSearchService(mongo *mongo.Client, ollamaClient *api.Client, ccfgg *appconfig.AppConfig) *SearchService {
	return &SearchService{
		mongo:        mongo,
		ollamaClient: ollamaClient,
		ccfgg:        ccfgg,
	}
}

func (s *SearchService) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	_, tenant := auth.GetUserIdAndTenant(ctx)

	// 1. Validate
	if len(req.Queries) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Queries cannot be empty")
	}

	if len(req.Queries) > 5 {
		req.Queries = req.Queries[:5] // Limit to 5 queries
	}

	// 2. Prepare search tasks
	hybridSearchTasks := make([]<-chan async.Result[[]*db.ChunkModel], 0, len(req.Queries))

	// 3. Perform search
	for _, q := range req.Queries {
		if q == "" {
			continue
		}

		// 3a. text search task
		hybridSearchTask := s.hybridSearch(ctx, tenant, q)
		hybridSearchTasks = append(hybridSearchTasks, hybridSearchTask)
	}

	// 4. Collect results
	hybridSearchResults, err := async.AwaitAll(hybridSearchTasks...)
	if err != nil {
		logger.Error("Failed to perform hybrid search", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "hybrid search: %v", err)
	}

	rankedChunks := linq.Flatten(hybridSearchResults)
	rankedChunks = linq.Distinct(rankedChunks, func(c *db.ChunkModel) string {
		return c.ChunkID
	})

	return buildSearchResponse(rankedChunks), nil
}

func (s *SearchService) hybridSearch(ctx context.Context, tenant, query string) <-chan async.Result[[]*db.ChunkModel] {
	return async.Go(func() ([]*db.ChunkModel, error) {
		textSerchTask := odm.CollectionOf[db.ChunkModel](s.mongo, tenant).
			TermSearch(ctx, query, odm.TermSearchParams{
				IndexName: textSearchIndexName,
				Path:      []string{"sentences", "sectionPath", "tags", "title"},
				Limit:     textK,
			})

		emb, err := prompts.EmbedOnce(ctx, s.ollamaClient, query)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "embed: %v", err)
		}

		vecSearchTask := odm.CollectionOf[db.ChunkAnnModel](s.mongo, tenant).
			VectorSearch(ctx, emb, odm.VectorSearchParams{
				IndexName:     vectorSearchIndexName,
				Path:          vectorSearchFieldName,
				K:             vecK,
				NumCandidates: 100,
			})

		textSearchChunkScoreMap, chunkCache, err := collectTextSearchHitTasks(textSerchTask)
		if err != nil {
			logger.Error("Failed to perform text search", zap.Error(err))
		}

		logger.Info("Text Search Results", zap.String("query", query), zap.Any("textSearchChunkScoreMap", textSearchChunkScoreMap))

		vecSearchChunkScoreMap, err := collectVectorSearchHitTasks(vecSearchTask)
		if err != nil {
			logger.Error("Failed to perform vector search", zap.Error(err))
		}

		logger.Info("Vector Search Results", zap.String("query", query), zap.Any("vecSearchChunkScoreMap", vecSearchChunkScoreMap))

		// Combine scores
		combinedScores := make(map[string]float64)
		for id, score := range textSearchChunkScoreMap {
			combinedScores[id] = score * textSearchWeight
		}

		for id, score := range vecSearchChunkScoreMap {
			combinedScores[id] += score * vectorSearchWeight
		}

		// Rank the chunks by combined score
		type chunkScore struct {
			id    string
			score float64
		}

		docScoresHeap := ds.NewMinHeap(func(a, b chunkScore) bool {
			return a.score < b.score // Min heap. Lowest score at the top
		})

		for id, score := range combinedScores {
			if score <= 0 {
				continue // Skip low-scoring results
			}

			docScoresHeap.Push(chunkScore{id: id, score: score})
			if docScoresHeap.Len() > maxChunks {
				docScoresHeap.Pop() // Remove the lowest scoring chunk if we exceed maxChunks
			}
		}

		selectedChunkIds := linq.Map(docScoresHeap.ToSortedSlice(), func(cs chunkScore) string {
			return cs.id
		})
		slices.Reverse(selectedChunkIds) // Reverse to have highest scores first

		// Fetch the top N chunks from the cache or database
		selectedChunks := s.fetchChunksByIds(ctx, tenant, chunkCache, selectedChunkIds)
		return selectedChunks, nil
	})
}

func (s *SearchService) fetchChunksByIds(ctx context.Context, tenant string, cache map[string]*db.ChunkModel, rankedIds []string) []*db.ChunkModel {

	if len(rankedIds) == 0 {
		return nil
	}

	/* 1. build map[id]Chunk from cache ------------------------ */
	chunkByID := make(map[string]*db.ChunkModel, len(rankedIds))
	var missing []string

	for _, id := range rankedIds {
		if c, ok := cache[id]; ok {
			chunkByID[id] = c
		} else {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		/* 2. fetch all missing in **one** DB round-trip -------- */
		dbChunks, err := async.Await(
			odm.CollectionOf[db.ChunkModel](s.mongo, tenant).Find(ctx, bson.M{"_id": bson.M{"$in": missing}}, nil, 0, 0),
		)
		if err != nil {
			logger.Error("Failed to fetch chunks from database", zap.Error(err))
			// we still return whatever we already have
		}
		for _, ch := range dbChunks {
			chunkByID[ch.ChunkID] = &ch
		}
	}

	/* 3. assemble slice in ranking order ---------------------- */
	ordered := make([]*db.ChunkModel, 0, len(rankedIds))
	for _, id := range rankedIds {
		if ch, ok := chunkByID[id]; ok {
			ordered = append(ordered, ch)
		} else {
			logger.Info("chunk id missing after lookup", zap.String("id", id))
		}
	}

	return ordered
}

func buildSearchResponse(docs []*db.ChunkModel) *pb.SearchResponse {
	if len(docs) == 0 {
		logger.Info("No search results found")
		return &pb.SearchResponse{
			Results: []*pb.SearchResult{},
			Status:  "no_results",
		}
	}

	var out []*pb.SearchResult

	for idx, doc := range docs {
		out = append(out, &pb.SearchResult{
			Sentences:   doc.Sentences,
			Title:       doc.Title,
			SectionPath: doc.SectionPath,
			Source:      doc.SourceURI,
			URL:         doc.SourceURI,
			DocIndex:    int32(idx + 1),
		})
	}

	return &pb.SearchResponse{
		Results: out,
		Status:  "success",
	}
}

func collectVectorSearchHitTasks(task <-chan async.Result[[]odm.SearchHit[db.ChunkAnnModel]]) (map[string]float64, error) {
	out := make(map[string]float64)

	// 1. Await all tasks
	searchHits, err := async.Await(task)
	if err != nil {
		logger.Error("Failed to collect search hits", zap.Error(err))
		return out, status.Errorf(codes.Internal, "collect search hits: %v", err)
	}

	max := 0.0
	for _, h := range searchHits {
		if h.Score > max {
			max = h.Score
		}
	}
	if max == 0 {
		max = 1
	}

	// 3. Remove duplicates by averaging the scores
	for _, hit := range searchHits {
		out[hit.Doc.Id()] = hit.Score / max // Normalise score to [0,1]
	}

	return out, nil
}

// collectTextSearchHitTasks merges duplicate hits *and* normalises the
// averaged text-search scores onto [0,1]  (0 = worst, 1 = best).
func collectTextSearchHitTasks(task <-chan async.Result[[]odm.SearchHit[db.ChunkModel]]) (map[string]float64, map[string]*db.ChunkModel, error) {
	cache := make(map[string]*db.ChunkModel)
	chunkToScoreMap := make(map[string]float64) // id → raw text score

	/* 1. await ------------------------------------------------------------ */
	textSearchResult, err := async.Await(task)
	if err != nil {
		logger.Error("Failed to collect search hits", zap.Error(err))
		return chunkToScoreMap, cache, status.Errorf(codes.Internal, "collect search hits: %v", err)
	}

	/* 2. Get chunk to score map and chunk cache ------------------------------------------ */
	for _, searchHit := range textSearchResult {
		id := searchHit.Doc.Id()
		chunkToScoreMap[id] = searchHit.Score

		if _, ok := cache[id]; !ok {
			cache[id] = &searchHit.Doc
		}
	}

	/* 3. min–max normalisation  ------------------------------------------ */
	//   Atlas text scores can be >> 1; we squeeze them to 0-1 so that
	//   textWeight + vectorWeight combine on comparable scales.
	var maxRaw float64
	for _, v := range chunkToScoreMap {
		if v > maxRaw {
			maxRaw = v
		}
	}
	if maxRaw == 0 {
		maxRaw = 1 // avoid div-zero if all scores are identical/zero
	}

	norm := make(map[string]float64, len(chunkToScoreMap))
	for id, v := range chunkToScoreMap {
		norm[id] = v / maxRaw // ∈[0,1]
	}

	return norm, cache, nil
}
