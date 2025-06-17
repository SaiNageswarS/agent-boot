package services

import (
	pb "agent-boot/proto/generated"
	"context"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/ollama/ollama/api"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	if len(req.Queries) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Queries cannot be empty")
	}

	if len(req.Queries) > 5 {
		req.Queries = req.Queries[:5] // Limit to 5 queries
	}

	var searchResultTasks []<-chan async.Result[[]odm.SearchHit[db.ChunkModel]]

	for _, query := range req.Queries {
		if query == "" {
			continue
		}

		// vector search
		queryEmbedding, err := prompts.EmbedOnce(ctx, s.ollamaClient, query)
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to get query embedding: "+err.Error())
		}

		if len(queryEmbedding) == 0 {
			return nil, status.Error(codes.Internal, "Query embedding is empty")
		}

		searchResultTask := odm.CollectionOf[db.ChunkModel](s.mongo, tenant).VectorSearch(ctx, queryEmbedding,
			odm.VectorSearchParams{IndexName: "chunkEmbeddingIndex", Path: "embedding", NumCandidates: 10, K: 5})
		searchResultTasks = append(searchResultTasks, searchResultTask)

		// text search
		textSearchResultTask := odm.CollectionOf[db.ChunkModel](s.mongo, tenant).TermSearch(ctx, query,
			odm.TermSearchParams{IndexName: "chunkIndex", Path: []string{"body", "sectionPath"}, Limit: 5})
		searchResultTasks = append(searchResultTasks, textSearchResultTask)
	}

	searchResults, err := async.AwaitAll(searchResultTasks...)
	if err != nil {
		logger.Error("Failed to perform search", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to perform search: "+err.Error())
	}

	docs := flattenSearchResults(searchResults)

	// TODO: Implement ranking logic. De-Duplicate chunks by similarity between them.
	searchResponse := buildSearchResponse(docs)
	searchResponse.SearchQueries = req.Queries
	logger.Info("Search completed", zap.Int("results_count", len(searchResponse.Results)))

	return searchResponse, nil
}

func buildSearchResponse(docs []db.ChunkModel) *pb.SearchResponse {
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

func flattenSearchResults(searchResults [][]odm.SearchHit[db.ChunkModel]) []db.ChunkModel {
	var docs []db.ChunkModel
	for _, hits := range searchResults {
		for _, hit := range hits {
			docs = append(docs, hit.Doc)
		}
	}

	// Remove duplicates based on ChunkID
	uniqueDocs := make(map[string]db.ChunkModel)
	for _, doc := range docs {
		uniqueDocs[doc.ChunkID] = doc
	}

	docs = make([]db.ChunkModel, 0, len(uniqueDocs))
	for _, doc := range uniqueDocs {
		docs = append(docs, doc)
	}

	return docs
}
