package services

import (
	pb "agent-boot/proto/generated"
	"context"
	"slices"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/ds"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// search parameters.
const (
	rrfK               = 60  // “dampening” constant from the RRF paper
	textSearchWeight   = 1.0 // optional per-engine weights
	vectorSearchWeight = 1.0
	vecK               = 20 // # of hits to keep from each engine
	textK              = 20
	maxChunks          = 20
)

type SearchService struct {
	pb.UnimplementedSearchServer
	mongo    odm.MongoClient
	embedder llm.Embedder
}

func ProvideSearchService(mongo odm.MongoClient, embedder llm.Embedder) *SearchService {
	return &SearchService{
		mongo:    mongo,
		embedder: embedder,
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

// ──────────────────────────────────────────────────────────────────────────────
//
//	Reciprocal-Rank Fusion (RRF)
//
//	Goal
//	────
//	▸ Convert *recall hits* (relevant docs that show up anywhere) into
//	  *precision hits* (relevant docs that land in the first N spots the user
//	  actually sees).
//
//	How it works
//	────────────
//	    RRF_score(d) = Σ_e  w_e / (k + rank_e(d))
//
//	    • One top-rank appearance (rank = 1) gets a big boost 1/(k+1), often
//	      enough to push the doc into the visible window.
//	    • A tail hit (rank = 20) earns < 1 % of that weight, so background
//	      noise barely moves the needle.
//
//	Why *rank* beats raw *score*
//	────────────────────────────
//	    – Scores live on different scales (BM25 ≈ 0-1000, cosine ≈ −1-1,
//	      PageRank ≪ 1).  Cross-normalising them is brittle.
//	    – Even a single engine’s scores drift when we rebuild the index or
//	      retrain embeddings; relative rank is far more stable.
//	    – Rank directly expresses “how good versus its peers,” the signal we
//	      need when merging heterogeneous lists.
//
//	Why we don’t hard-threshold BM25 or similarity scores
//	─────────────────────────────────────────────────────
//	    – The 1/(k+rank) formula *already* down-weights tail hits; a rank-20
//	      doc contributes < 1 % of a rank-1 doc, so low-quality noise is
//	      effectively ignored without hurting recall.
//	    – Fixed score cut-offs tie us to one model/index version and risk
//	      dropping docs that are mediocre in one engine but stellar in another
//	      (the classic hybrid-search win).
//
//	Bottom line
//	───────────
//	Let every engine vote by rank, fuse with 1/(k+rank), and keep explicit
//	score thresholds only for domain-specific guard-rails.
//
// ──────────────────────────────────────────────────────────────────────────────
func (s *SearchService) hybridSearch(ctx context.Context,
	tenant, query string) <-chan async.Result[[]*db.ChunkModel] {

	return async.Go(func() ([]*db.ChunkModel, error) {
		//----------------------------------------------------------------------
		// 1. Fire the two independent searches in parallel
		//----------------------------------------------------------------------
		textTask := odm.CollectionOf[db.ChunkModel](s.mongo, tenant).
			TermSearch(ctx, query, odm.TermSearchParams{
				IndexName: db.TextSearchIndexName,
				Path:      db.TextSearchPaths,
				Limit:     textK,
			})

		emb, err := async.Await(s.embedder.GetEmbedding(ctx, query, llm.WithTask("retrieval.query")))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "embed: %v", err)
		}

		vecTask := odm.CollectionOf[db.ChunkAnnModel](s.mongo, tenant).
			VectorSearch(ctx, emb, odm.VectorSearchParams{
				IndexName:     db.VectorIndexName,
				Path:          db.VectorPath,
				K:             vecK,
				NumCandidates: 100,
			})

		//----------------------------------------------------------------------
		// 2. Convert each result list → id→rank    (rank ∈ {1,2,…})
		//----------------------------------------------------------------------
		textRanks, cache, err := collectTextSearchRanks(textTask)
		if err != nil {
			logger.Error("text search failed", zap.Error(err))
		}

		vecRanks, err := collectVectorSearchRanks(vecTask)
		if err != nil {
			logger.Error("vector search failed", zap.Error(err))
		}

		//----------------------------------------------------------------------
		// 3. Reciprocal-Rank Fusion
		//     score(id) = Σ  weight_e / (rrfK + rank_e(id))
		//----------------------------------------------------------------------
		combined := make(map[string]float64)
		for id, r := range textRanks {
			combined[id] = textSearchWeight / float64(rrfK+r)
		}
		for id, r := range vecRanks {
			combined[id] += vectorSearchWeight / float64(rrfK+r)
		}

		//----------------------------------------------------------------------
		// 4. Keep the top-N with a min-heap (higher RRF score = better)
		//----------------------------------------------------------------------
		type pair struct {
			id    string
			score float64
		}

		h := ds.NewMinHeap(func(a, b pair) bool { return a.score < b.score })
		for id, sc := range combined {
			h.Push(pair{id, sc})
			if h.Len() > maxChunks {
				h.Pop()
			}
		}

		ids := linq.Map(h.ToSortedSlice(), func(p pair) string { return p.id })
		slices.Reverse(ids) // highest score first

		//----------------------------------------------------------------------
		// 5. Materialise the chunks
		//----------------------------------------------------------------------
		return s.fetchChunksByIds(ctx, tenant, cache, ids), nil
	})
}

// Returns id→rank (1-based) **and** a cache of the full ChunkModel docs.
func collectTextSearchRanks(
	task <-chan async.Result[[]odm.SearchHit[db.ChunkModel]],
) (map[string]int, map[string]*db.ChunkModel, error) {

	ranks := make(map[string]int) // id → rank
	cache := make(map[string]*db.ChunkModel)

	hits, err := async.Await(task)
	if err != nil {
		return ranks, cache, status.Errorf(codes.Internal, "await text hits: %v", err)
	}

	for i, h := range hits {
		id := h.Doc.Id()
		if _, seen := ranks[id]; !seen { // keep first (best-ranked) hit
			ranks[id] = i + 1  // 1-based rank
			cache[id] = &h.Doc // stash full doc for later
		}
	}
	return ranks, cache, nil
}

// Returns id→rank (1-based) for vector search hits.
func collectVectorSearchRanks(
	task <-chan async.Result[[]odm.SearchHit[db.ChunkAnnModel]],
) (map[string]int, error) {

	ranks := make(map[string]int)

	hits, err := async.Await(task)
	if err != nil {
		return ranks, status.Errorf(codes.Internal, "await vector hits: %v", err)
	}

	for i, h := range hits {
		id := h.Doc.Id()
		if _, seen := ranks[id]; !seen {
			ranks[id] = i + 1
		}
	}
	return ranks, nil
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
