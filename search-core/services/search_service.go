package services

import (
	pb "agent-boot/proto/generated"
	"context"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SearchService struct {
	pb.UnimplementedSearchServer
	mongo     *mongo.Client
	llm       *llm.AnthropicClient
	embeddder *llm.JinaAIEmbeddingClient
	ccfgg     *appconfig.AppConfig
}

func ProvideSearchService(mongo *mongo.Client, llmClient *llm.AnthropicClient, embedder *llm.JinaAIEmbeddingClient, ccfgg *appconfig.AppConfig) *SearchService {
	return &SearchService{
		mongo:     mongo,
		llm:       llmClient,
		embeddder: embedder,
		ccfgg:     ccfgg,
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

	if !s.ccfgg.VectorSearchEnabled && !s.ccfgg.TextSearchEnabled {
		return nil, status.Error(codes.Unimplemented, "Both Vector and text search are disabled in the configuration")
	}

	var searchResultTasks []<-chan async.Result[[]odm.SearchHit[db.ChunkModel]]

	for _, query := range req.Queries {
		if query == "" {
			continue
		}

		if s.ccfgg.VectorSearchEnabled {
			queryEmbedding, err := async.Await(s.embeddder.GetEmbedding(ctx, llm.JinaAIEmbeddingRequest{Task: "retrieval.query", Input: []string{query}}))
			if err != nil {
				return nil, status.Error(codes.Internal, "Failed to get query embedding: "+err.Error())
			}

			if len(queryEmbedding) == 0 {
				return nil, status.Error(codes.Internal, "Query embedding is empty")
			}

			logger.Info("Query embedding", zap.String("query", query), zap.Any("embedding", queryEmbedding))
			searchResultTask := odm.CollectionOf[db.ChunkModel](s.mongo, tenant).VectorSearch(ctx, queryEmbedding,
				odm.VectorSearchParams{IndexName: "chunkEmbeddingIndex", Path: "body", NumCandidates: 10, K: 5})
			searchResultTasks = append(searchResultTasks, searchResultTask)
		}

		if s.ccfgg.TextSearchEnabled {
			textSearchResultTask := odm.CollectionOf[db.ChunkModel](s.mongo, tenant).TermSearch(ctx, query,
				odm.TermSearchParams{IndexName: "chunkIndex", Path: []string{"body", "sectionPath"}, Limit: 5})
			searchResultTasks = append(searchResultTasks, textSearchResultTask)
		}
	}

	searchResults, err := async.AwaitAll(searchResultTasks...)
	if err != nil {
		logger.Error("Failed to perform search", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to perform search: "+err.Error())
	}

	docs := flattenSearchResults(searchResults)

	// TODO: Implement ranking logic. De-Duplicate chunks by similarity between them.
	return buildSearchResponse(docs), nil
}

func buildSearchResponse(docs []db.ChunkModel) *pb.SearchResponse {
	var out []*pb.Chunk
	for _, doc := range docs {
		out = append(out, &pb.Chunk{
			Citation: doc.SourceURI,
			Body:     doc.SectionPath + "\n\n" + doc.Body,
		})
	}

	return &pb.SearchResponse{
		Chunks: out,
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
