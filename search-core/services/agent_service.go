package services

import (
	pb "agent-boot/proto/generated"
	"context"
	"fmt"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/ollama/ollama/api"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type AgentService struct {
	pb.UnimplementedAgentServer
	mongo         *mongo.Client
	llmClient     *llm.AnthropicClient
	searchService *SearchService
}

func ProvideAgentService(mongo *mongo.Client, llmClient *llm.AnthropicClient, ollamaClient *api.Client, ccfgg *appconfig.AppConfig) *AgentService {
	return &AgentService{
		mongo:         mongo,
		llmClient:     llmClient,
		searchService: ProvideSearchService(mongo, ollamaClient, ccfgg),
	}
}

func (s *AgentService) CallAgent(ctx context.Context, req *pb.AgentInput) (*pb.AgentResponse, error) {
	_, tenant := auth.GetUserIdAndTenant(ctx)

	// 1. Get Agent Details from DB.
	defaultAgentName := "default-agent"
	agentDetail, err := async.Await(odm.CollectionOf[db.AgentModel](s.mongo, tenant).FindOneByID(ctx, defaultAgentName))
	if err != nil {
		logger.Error("Error finding agent", zap.String("agent", defaultAgentName), zap.Error(err))
		return nil, err
	}

	if agentDetail == nil {
		logger.Error("Agent not found", zap.String("agent", defaultAgentName))
		return nil, status.Error(codes.NotFound, fmt.Sprintf("agent %s not found", defaultAgentName))
	}

	// 2. Extract Agent Input using LLM.
	agentInput, err := async.Await(prompts.ExtractAgentInput(ctx, s.llmClient, req.Text, agentDetail.Capability))
	if err != nil {
		logger.Error("Error extracting agent input", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to extract agent input")
	}

	if !agentInput.Relevant {
		logger.Info("Agent input not relevant", zap.String("reasoning", agentInput.Reasoning))
		return &pb.AgentResponse{Status: "not_relevant"}, nil
	}

	logger.Info("Agent input is relevant", zap.String("reasoning", agentInput.Reasoning))

	// 3. Perform Search using Search Service based on extracted queries of the agent input.
	searchResults, err := s.searchService.Search(ctx, &pb.SearchRequest{Queries: agentInput.SearchQueries})

	if err != nil {
		logger.Error("Error performing search", zap.Error(err))
		return nil, err
	}

	// 4. Generate Answer using LLM based on search results and agent input.
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,  // Use field names from the .proto instead of lowerCamelCase
		EmitUnpopulated: false, // Don't include fields with zero values
	}

	searchResultsJson, err := marshaler.Marshal(searchResults)
	if err != nil {
		logger.Error("Error marshaling search results", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to marshal search results")
	}

	answer, err := async.Await(prompts.GenerateAnswer(ctx, s.llmClient, agentDetail.Capability, req.Text, string(searchResultsJson)))
	if err != nil {
		logger.Error("Error generating answer", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate answer")
	}

	return &pb.AgentResponse{
		Status:        "success",
		Answer:        answer,
		SearchResults: searchResults.Results,
		SearchRequest: &pb.SearchRequest{
			Queries: agentInput.SearchQueries,
		},
	}, nil
}
