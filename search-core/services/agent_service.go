package services

import (
	pb "agent-boot/proto/generated"
	"fmt"
	"time"

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
	"google.golang.org/grpc"
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

func (s *AgentService) CallAgent(req *pb.AgentInput, stream grpc.ServerStreamingServer[pb.AgentStreamChunk]) error {
	ctx := stream.Context()
	_, tenant := auth.GetUserIdAndTenant(ctx)

	// Helper function to send with error checking and flushing
	sendChunk := func(chunk *pb.AgentStreamChunk) error {
		if err := stream.Send(chunk); err != nil {
			logger.Error("Failed to send chunk", zap.Error(err))
			return err
		}
		// Force flush after each send for better streaming
		if flusher, ok := stream.(interface{ Flush() error }); ok {
			flusher.Flush()
		}
		return nil
	}

	// Helper function to send error and return (since error chunks are final)
	sendFinalError := func(message, code string) error {
		return sendChunk(&pb.AgentStreamChunk{
			ChunkType: &pb.AgentStreamChunk_Error{
				Error: &pb.StreamError{
					ErrorMessage: message,
					ErrorCode:    code,
				},
			},
		})
	}

	// Helper function to send metadata
	sendMetadata := func(status string, estimatedQueries, estimatedResults int32) error {
		return sendChunk(&pb.AgentStreamChunk{
			ChunkType: &pb.AgentStreamChunk_Metadata{
				Metadata: &pb.StreamMetadata{
					Status:           status,
					EstimatedQueries: estimatedQueries,
					EstimatedResults: estimatedResults,
				},
			},
		})
	}

	// Add defer to ensure stream is properly closed
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Stream panic recovered", zap.Any("panic", r))
		}
	}()

	// Send initial metadata
	if err := sendMetadata("starting", 0, 0); err != nil {
		return err
	}

	// 1. Get Agent Details from DB.
	defaultAgentName := "default-agent"
	agentDetail, err := async.Await(odm.CollectionOf[db.AgentModel](s.mongo, tenant).FindOneByID(ctx, defaultAgentName))
	if err != nil {
		logger.Error("Error finding agent", zap.String("agent", defaultAgentName), zap.Error(err))
		return sendFinalError("Failed to find agent configuration", "AGENT_NOT_FOUND")
	}

	if agentDetail == nil {
		logger.Error("Agent not found", zap.String("agent", defaultAgentName))
		return sendFinalError(fmt.Sprintf("Agent %s not found", defaultAgentName), "AGENT_CONFIG_MISSING")
	}

	// Update metadata - agent found
	if err := sendMetadata("processing_input", 0, 0); err != nil {
		return err
	}

	// 2. Extract Agent Input using LLM.
	agentInput, err := async.Await(prompts.ExtractAgentInput(ctx, s.llmClient, req.Text, agentDetail.Capability))
	if err != nil {
		logger.Error("Error extracting agent input", zap.Error(err))
		return sendFinalError("Failed to extract search queries from your input", "INPUT_EXTRACTION_FAILED")
	}

	if !agentInput.Relevant {
		logger.Info("Agent input not relevant", zap.String("reasoning", agentInput.Reasoning))

		// Send answer explaining why it's not relevant
		err = sendChunk(&pb.AgentStreamChunk{
			ChunkType: &pb.AgentStreamChunk_Answer{
				Answer: &pb.AnswerChunk{
					Content: fmt.Sprintf("I'm not able to help with this request. %s", agentInput.Reasoning),
					IsFinal: true,
				},
			},
		})
		if err != nil {
			return err
		}

		// Send completion for not relevant
		return sendChunk(&pb.AgentStreamChunk{
			ChunkType: &pb.AgentStreamChunk_Complete{
				Complete: &pb.StreamComplete{
					FinalStatus:      "not_relevant",
					TotalResultsSent: 0,
					TotalQueriesSent: int32(len(agentInput.SearchQueries)),
				},
			},
		})
	}

	logger.Info("Agent input is relevant", zap.String("reasoning", agentInput.Reasoning))

	// Send search queries
	err = sendChunk(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_SearchRequest{
			SearchRequest: &pb.SearchRequestChunk{
				Queries:      agentInput.SearchQueries,
				ChunkIndex:   0,
				IsFinalChunk: true,
			},
		},
	})
	if err != nil {
		return err
	}

	// 3. Perform Search using Search Service based on extracted queries of the agent input.
	// Update metadata - starting search
	if err := sendMetadata("searching", int32(len(agentInput.SearchQueries)), 0); err != nil {
		return err
	}

	searchResults, err := s.searchService.Search(ctx, &pb.SearchRequest{Queries: agentInput.SearchQueries})
	if err != nil {
		logger.Error("Error performing search", zap.Error(err))
		return sendFinalError("Search service failed to process your query", "SEARCH_FAILED")
	}

	totalResults := len(searchResults.Results)

	// Update metadata with actual results count
	if err := sendMetadata("processing_results", int32(len(agentInput.SearchQueries)), int32(totalResults)); err != nil {
		return err
	}

	const chunkSize = 10
	// Send search results asynchronously with proper error handling
	sendSearchResultsTask := async.Go(func() (struct{}, error) {
		// Send empty chunk if no results
		if totalResults == 0 {
			return struct{}{}, sendChunk(&pb.AgentStreamChunk{
				ChunkType: &pb.AgentStreamChunk_SearchResults{
					SearchResults: &pb.SearchResultsChunk{
						Results:      []*pb.SearchResult{},
						ChunkIndex:   0,
						TotalChunks:  1,
						IsFinalChunk: true,
					},
				},
			})
		}

		totalChunks := (totalResults + chunkSize - 1) / chunkSize

		for i := 0; i < totalResults; i += chunkSize {
			// Check if context is cancelled
			if ctx.Err() != nil {
				return struct{}{}, ctx.Err()
			}

			end := min(i+chunkSize, totalResults)
			chunk := searchResults.Results[i:end]
			chunkIndex := i / chunkSize
			isLast := end >= totalResults

			err := sendChunk(&pb.AgentStreamChunk{
				ChunkType: &pb.AgentStreamChunk_SearchResults{
					SearchResults: &pb.SearchResultsChunk{
						Results:      chunk,
						ChunkIndex:   int32(chunkIndex),
						TotalChunks:  int32(totalChunks),
						IsFinalChunk: isLast,
					},
				},
			})
			if err != nil {
				logger.Error("Failed to send search results chunk",
					zap.Int("chunk_index", chunkIndex),
					zap.Error(err))
				return struct{}{}, err
			}

			// Small delay only if not last chunk
			if !isLast {
				time.Sleep(50 * time.Millisecond) // Increased delay for stability
			}
		}

		logger.Info("Search results sending completed", zap.Int("total_chunks", totalChunks))
		return struct{}{}, nil
	})

	// 4. Generate Answer using LLM based on search results and agent input.
	// Update metadata - generating answer
	if err := sendMetadata("generating_answer", int32(len(agentInput.SearchQueries)), int32(totalResults)); err != nil {
		return err
	}

	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}

	searchResultsJson, err := marshaler.Marshal(searchResults)
	if err != nil {
		logger.Error("Error marshaling search results", zap.Error(err))
		// Wait for search results to finish sending before sending error
		async.Await(sendSearchResultsTask)
		return sendFinalError("Failed to process search results for answer generation", "MARSHAL_FAILED")
	}

	answerTask := async.Go(func() (string, error) {
		return async.Await(prompts.GenerateAnswer(ctx, s.llmClient, agentDetail.Capability, req.Text, string(searchResultsJson)))
	})

	// Wait for search results to finish sending
	_, searchErr := async.Await(sendSearchResultsTask)
	if searchErr != nil {
		logger.Error("Error sending search results", zap.Error(searchErr))
		return searchErr
	}

	// Wait for answer generation
	answer, answerErr := async.Await(answerTask)
	if answerErr != nil {
		logger.Error("Error generating answer", zap.Error(answerErr))
		return sendFinalError("Failed to generate answer, but search results are available above", "ANSWER_GENERATION_FAILED")
	}

	// Send answer with extra safety
	logger.Info("Sending answer", zap.Int("answer_length", len(answer)))
	err = sendChunk(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Answer{
			Answer: &pb.AnswerChunk{
				Content: answer,
				IsFinal: true,
			},
		},
	})
	if err != nil {
		logger.Error("Failed to send answer", zap.Error(err))
		return err
	}

	// Send final completion with extra safety
	logger.Info("Sending completion")
	err = sendChunk(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Complete{
			Complete: &pb.StreamComplete{
				FinalStatus:      "success",
				TotalResultsSent: int32(totalResults),
				TotalQueriesSent: int32(len(agentInput.SearchQueries)),
			},
		},
	})
	if err != nil {
		logger.Error("Failed to send completion", zap.Error(err))
		return err
	}

	logger.Info("Agent streaming completed successfully",
		zap.Int("total_results", totalResults),
		zap.Int("total_queries", len(agentInput.SearchQueries)))

	return nil
}
