package agent

import (
	"context"
	"slices"
	"sort"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
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

type SearchStep struct {
	embedder         llm.Embedder
	chunkRepository  odm.OdmCollectionInterface[db.ChunkModel]
	vectorRepository odm.OdmCollectionInterface[db.ChunkAnnModel]
}

func NewSearchStep(chunkRepository odm.OdmCollectionInterface[db.ChunkModel], vectorRepository odm.OdmCollectionInterface[db.ChunkAnnModel], embedder llm.Embedder) *SearchStep {
	return &SearchStep{
		chunkRepository:  chunkRepository,
		vectorRepository: vectorRepository,
		embedder:         embedder,
	}
}

func (s *SearchStep) Run(ctx context.Context, queries []string) ([]*db.ChunkModel, error) {
	hybridSearchTasks := make([]<-chan async.Result[[]*db.ChunkModel], 0, len(queries))

	// 1. Perform hybrid search for each query
	//    (text search + vector search)
	for _, q := range queries {
		if q == "" {
			continue
		}

		hybridSearchTask := s.hybridSearch(ctx, q)
		hybridSearchTasks = append(hybridSearchTasks, hybridSearchTask)
	}

	// 2. Collect results ranked by RRF score
	hybridSearchResults, err := async.AwaitAll(hybridSearchTasks...)
	if err != nil {
		logger.Error("Failed to perform hybrid search", zap.Error(err))
		return nil, err
	}

	// 3. Flatten and deduplicate results
	rankedChunks := linq.Flatten(hybridSearchResults)
	rankedChunks = linq.Distinct(rankedChunks, func(c *db.ChunkModel) string {
		return c.ChunkID
	})

	// 4. Add neighbors and reorder chunks
	rankedChunks = s.addNeighborsAndReorder(ctx, rankedChunks)
	return rankedChunks, nil
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
func (s *SearchStep) hybridSearch(ctx context.Context, query string) <-chan async.Result[[]*db.ChunkModel] {

	return async.Go(func() ([]*db.ChunkModel, error) {
		//----------------------------------------------------------------------
		// 1. Fire the two independent searches in parallel
		//----------------------------------------------------------------------
		textTask := s.chunkRepository.
			TermSearch(ctx, query, odm.TermSearchParams{
				IndexName: db.TextSearchIndexName,
				Path:      db.TextSearchPaths,
				Limit:     textK,
			})

		emb, err := async.Await(s.embedder.GetEmbedding(ctx, query, llm.WithTask("retrieval.query")))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "embed: %v", err)
		}

		vecTask := s.vectorRepository.
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
		return s.fetchChunksByIds(ctx, cache, ids), nil
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

func (s *SearchStep) fetchChunksByIds(ctx context.Context, cache map[string]*db.ChunkModel, rankedIds []string) []*db.ChunkModel {

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
			s.chunkRepository.Find(ctx, bson.M{"_id": bson.M{"$in": missing}}, nil, 0, 0),
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

func (s *SearchStep) addNeighborsAndReorder(ctx context.Context, rankedChunks []*db.ChunkModel) []*db.ChunkModel {
	if len(rankedChunks) == 0 {
		return rankedChunks
	}

	// 1. Order chunks by WindowIndex in a Section preserving the RRF order.
	sectionRank := make(map[string]int, len(rankedChunks))
	for idx, ch := range rankedChunks {
		if _, ok := sectionRank[ch.SectionID]; !ok {
			sectionRank[ch.SectionID] = idx
		}
	}

	sort.SliceStable(rankedChunks, func(i, j int) bool {
		si, sj := rankedChunks[i], rankedChunks[j]

		if si.SectionID != sj.SectionID {
			return sectionRank[si.SectionID] < sectionRank[sj.SectionID]
		}
		return si.WindowIndex < sj.WindowIndex
	})

	// 2. Populate PrevChunkID and NextChunkID for each chunk.
	seenIds := ds.NewSet[string]()
	for _, c := range rankedChunks {
		seenIds.Add(c.ChunkID)
	}

	needIds := ds.NewSet[string]()
	for _, c := range rankedChunks {
		if len(c.PrevChunkID) > 0 && !seenIds.Contains(c.PrevChunkID) && !needIds.Contains(c.PrevChunkID) {
			needIds.Add(c.PrevChunkID)
		}

		if len(c.NextChunkID) > 0 && !seenIds.Contains(c.NextChunkID) && !needIds.Contains(c.NextChunkID) {
			needIds.Add(c.NextChunkID)
		}
	}

	var neighbors []db.ChunkModel
	var err error
	if needIds.Len() > 0 {
		neighbors, err = async.Await(
			s.chunkRepository.Find(ctx, bson.M{"_id": bson.M{"$in": needIds.ToSlice()}}, nil, 0, 0),
		)

		if err != nil {
			logger.Error("Failed to fetch neighbors from database", zap.Error(err))
			return rankedChunks // return what we have so far
		}
	}

	// 3. Build a map for quick neighbor lookup.
	neighborsById := make(map[string]*db.ChunkModel, len(neighbors))
	for i := range neighbors {
		n := neighbors[i]
		neighborsById[n.ChunkID] = &n
	}

	// 4. Update PrevChunkID and NextChunkID for each chunk.
	out := make([]*db.ChunkModel, 0, len(rankedChunks)*3)
	for _, c := range rankedChunks {
		if prevChunk, ok := neighborsById[c.PrevChunkID]; ok {
			out = append(out, prevChunk)
			delete(neighborsById, c.PrevChunkID) // avoid duplicates
		}

		out = append(out, c)

		if nextChunk, ok := neighborsById[c.NextChunkID]; ok {
			out = append(out, nextChunk)
			delete(neighborsById, c.NextChunkID) // avoid duplicates
		}
	}

	return out
}
