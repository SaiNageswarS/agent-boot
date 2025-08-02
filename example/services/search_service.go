package services

import (
	pb "agent-boot/proto/generated"
	"context"

	"github.com/SaiNageswarS/agent-boot/example/agent"
	"github.com/SaiNageswarS/agent-boot/example/db"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/embed"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SearchService struct {
	pb.UnimplementedSearchServer
	mongo    odm.MongoClient
	embedder embed.Embedder
}

func ProvideSearchService(mongo odm.MongoClient, embedder embed.Embedder) *SearchService {
	return &SearchService{
		mongo:    mongo,
		embedder: embedder,
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

	chunkRepository := odm.CollectionOf[db.ChunkModel](s.mongo, tenant)
	vectorRepository := odm.CollectionOf[db.ChunkAnnModel](s.mongo, tenant)

	searchStep := agent.NewSearchStep(chunkRepository, vectorRepository, s.embedder)
	rankedChunks, err := searchStep.Run(ctx, req.Queries)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Search failed: %v", err)
	}

	return buildSearchResponse(rankedChunks), nil
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
			Sentences:     doc.Sentences,
			Title:         doc.Title,
			SectionPath:   doc.SectionPath,
			Source:        doc.SourceURI,
			URL:           doc.SourceURI,
			CitationIndex: int32(idx + 1),
			ChunkId:       doc.ChunkID,
		})
	}

	return &pb.SearchResponse{
		Results: out,
		Status:  "success",
	}
}
