package services

import (
	pb "agent-boot/proto/generated"
	"context"
	"fmt"
	"time"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type AgentService struct {
	pb.UnimplementedAgentServer
	mongo         odm.MongoClient
	llmClient     *llm.AnthropicClient
	searchService *SearchService
}

func ProvideAgentService(mongo odm.MongoClient, llmClient *llm.AnthropicClient, embedder llm.Embedder) *AgentService {
	return &AgentService{
		mongo:         mongo,
		llmClient:     llmClient,
		searchService: ProvideSearchService(mongo, embedder),
	}
}

func (s *AgentService) CallAgent(req *pb.AgentInput, stream grpc.ServerStreamingServer[pb.AgentStreamChunk]) error {
	ctx := stream.Context()
	userId, tenant := auth.GetUserIdAndTenant(ctx)

	session := db.NewSessionModel(userId)
	if len(req.SessionId) > 0 {
		existingSession, err := async.Await(odm.CollectionOf[db.SessionModel](s.mongo, tenant).FindOneByID(ctx, req.SessionId))
		if err != nil {
			logger.Error("Error finding existing session", zap.String("session_id", req.SessionId), zap.Error(err))
		} else if existingSession != nil {
			logger.Info("Using existing session", zap.String("session_id", req.SessionId))
			session = existingSession
		}
	}

	turn := db.TurnModel{UserInput: req.Text}

	sh := newStreamHelper(stream)
	defer sh.Recover() // never crash caller
	if err := sh.SendMetadata("starting", 0, 0); err != nil {
		return err
	}

	// 1. Get Agent Details from DB.
	defaultAgentName := "default-agent"
	agentDetail, err := async.Await(odm.CollectionOf[db.AgentModel](s.mongo, tenant).FindOneByID(ctx, defaultAgentName))
	if err != nil {
		logger.Error("Error finding agent", zap.String("agent", defaultAgentName), zap.Error(err))
		return sh.SendFinalError("Failed to find agent configuration", "AGENT_NOT_FOUND")
	}

	if agentDetail == nil {
		logger.Error("Agent not found", zap.String("agent", defaultAgentName))
		return sh.SendFinalError(fmt.Sprintf("Agent %s not found", defaultAgentName), "AGENT_CONFIG_MISSING")
	}

	// Update metadata - agent found
	if err := sh.SendMetadata("processing_input", 0, 0); err != nil {
		return err
	}

	// 2. Extract search queries using LLM.
	searchQueries, err := async.Await(prompts.ExtractSearchQueries(ctx, s.llmClient, req.Text, agentDetail.Capability))
	if err != nil {
		logger.Error("Error extracting agent input", zap.Error(err))
		return sh.SendFinalError("Failed to extract search queries from your input", "INPUT_EXTRACTION_FAILED")
	}

	if !searchQueries.Relevant {
		logger.Info("Agent input not relevant", zap.String("reasoning", searchQueries.Reasoning))

		// Send answer explaining why it's not relevant
		return sh.SendNotRelevant(searchQueries.Reasoning, len(searchQueries.SearchQueries))
	}

	logger.Info("Agent input is relevant", zap.String("reasoning", searchQueries.Reasoning))

	// Send search queries
	if err := sh.SendQueries(searchQueries.SearchQueries); err != nil {
		return err
	}
	turn.SearchQueries = searchQueries.SearchQueries

	// 3. Perform Search using Search Service based on extracted queries of the agent input.
	// Update metadata - starting search
	if err := sh.SendMetadata("searching", int32(len(searchQueries.SearchQueries)), 0); err != nil {
		return err
	}

	searchResults, err := s.searchService.Search(ctx, &pb.SearchRequest{Queries: searchQueries.SearchQueries})
	if err != nil {
		logger.Error("Error performing search", zap.Error(err))
		return sh.SendFinalError("Search service failed to process your query", "SEARCH_FAILED")
	}

	totalResults := len(searchResults.Results)
	turn.SearchResultChunkIds = linq.Map(searchResults.Results, func(r *pb.SearchResult) string {
		return r.ChunkId
	})

	// Update metadata with actual results count
	if err := sh.SendMetadata("processing_results", int32(len(searchQueries.SearchQueries)), int32(totalResults)); err != nil {
		return err
	}

	// Send search results asynchronously with proper error handling
	sendSearchResultsTask := async.Go(func() (struct{}, error) {
		if err = sh.SendSearchResults(ctx, searchResults.Results); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	})

	// 4. Generate Answer using LLM based on search results and agent input.
	// Update metadata - generating answer
	if err := sh.SendMetadata("generating_answer", int32(len(searchQueries.SearchQueries)), int32(totalResults)); err != nil {
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
		return sh.SendFinalError("Failed to process search results for answer generation", "MARSHAL_FAILED")
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
		return sh.SendFinalError("Failed to generate answer, but search results are available above", "ANSWER_GENERATION_FAILED")
	}

	turn.AgentAnswer = answer
	turn.Model = "claude-3-5-sonnet-20241022"
	session.Turns = append(session.Turns, turn)

	async.Await(odm.CollectionOf[db.SessionModel](s.mongo, tenant).Save(ctx, *session))

	// Send answer with extra safety
	logger.Info("Sending answer", zap.Int("answer_length", len(answer)))
	return sh.SendAnswer(answer, totalResults, len(searchQueries.SearchQueries))
}

type streamHelper struct {
	stream grpc.ServerStreamingServer[pb.AgentStreamChunk]
	flush  func() error
}

func newStreamHelper(s grpc.ServerStreamingServer[pb.AgentStreamChunk]) *streamHelper {
	var flusher func() error
	if f, ok := s.(interface{ Flush() error }); ok {
		flusher = f.Flush
	}
	return &streamHelper{stream: s, flush: flusher}
}

func (h *streamHelper) send(c *pb.AgentStreamChunk) error {
	if err := h.stream.Send(c); err != nil {
		return err
	}
	if h.flush != nil {
		_ = h.flush()
	}
	return nil
}

func (h *streamHelper) SendMetadata(status string, q, r int32) error {
	return h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Metadata{Metadata: &pb.StreamMetadata{
			Status: status, EstimatedQueries: q, EstimatedResults: r,
		}},
	})
}

func (h *streamHelper) SendFinalError(msg, code string) error {
	return h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Error{Error: &pb.StreamError{
			ErrorMessage: msg, ErrorCode: code,
		}},
	})
}

func (h *streamHelper) SendNotRelevant(reason string, q int) error {
	if err := h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Answer{Answer: &pb.AnswerChunk{
			Content: fmt.Sprintf("I'm not able to help with this request. %s", reason),
			IsFinal: true,
		}},
	}); err != nil {
		return err
	}

	return h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Complete{Complete: &pb.StreamComplete{
			FinalStatus: "not_relevant", TotalResultsSent: 0, TotalQueriesSent: int32(q),
		}},
	})
}

func (h *streamHelper) SendQueries(queries []string) error {
	return h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_SearchRequest{SearchRequest: &pb.SearchRequestChunk{
			Queries: queries, ChunkIndex: 0, IsFinalChunk: true,
		}},
	})
}

func (h *streamHelper) SendSearchResults(ctx context.Context, res []*pb.SearchResult) error {
	const size = 10
	if len(res) == 0 {
		return h.send(&pb.AgentStreamChunk{
			ChunkType: &pb.AgentStreamChunk_SearchResults{SearchResults: &pb.SearchResultsChunk{
				Results: []*pb.SearchResult{}, ChunkIndex: 0, TotalChunks: 1, IsFinalChunk: true,
			}},
		})
	}

	total := len(res)
	chunks := (total + size - 1) / size
	for i := 0; i < total; i += size {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		end := min(i+size, total)
		if err := h.send(&pb.AgentStreamChunk{
			ChunkType: &pb.AgentStreamChunk_SearchResults{SearchResults: &pb.SearchResultsChunk{
				Results: res[i:end], ChunkIndex: int32(i / size), TotalChunks: int32(chunks),
				IsFinalChunk: end == total,
			}},
		}); err != nil {
			return err
		}
		if end != total {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}

func (h *streamHelper) SendAnswer(answer string, totalRes, totalQ int) error {
	if err := h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Answer{Answer: &pb.AnswerChunk{Content: answer, IsFinal: true}},
	}); err != nil {
		return err
	}

	return h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Complete{Complete: &pb.StreamComplete{
			FinalStatus: "success", TotalResultsSent: int32(totalRes), TotalQueriesSent: int32(totalQ),
		}},
	})
}

func (h *streamHelper) Recover() {
	if r := recover(); r != nil {
		logger.Error("Stream panic recovered", zap.Any("panic", r))
	}
}
