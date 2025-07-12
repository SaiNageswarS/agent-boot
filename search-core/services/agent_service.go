package services

import (
	pb "agent-boot/proto/generated"
	"context"
	"fmt"
	"sync"

	"github.com/SaiNageswarS/agent-boot/search-core/agent"
	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type AgentService struct {
	pb.UnimplementedAgentServer
	mongo           odm.MongoClient
	anthropicClient *llm.AnthropicClient
	ollamaClient    *llm.OllamaLLMClient
	ccfgg           *appconfig.AppConfig
	embdder         llm.Embedder
}

func ProvideAgentService(mongo odm.MongoClient, anthropicClient *llm.AnthropicClient, ollamaClient *llm.OllamaLLMClient, embedder llm.Embedder, ccfgg *appconfig.AppConfig) *AgentService {
	return &AgentService{
		mongo:           mongo,
		anthropicClient: anthropicClient,
		ollamaClient:    ollamaClient,
		ccfgg:           ccfgg,
		embdder:         embedder,
	}
}

func (s *AgentService) CallAgent(req *pb.AgentInput, stream grpc.ServerStreamingServer[pb.AgentStreamChunk]) error {
	ctx := stream.Context()
	userId, tenant := auth.GetUserIdAndTenant(ctx)

	session := s.getSession(ctx, tenant, userId, req.SessionId)

	sh := newStreamHelper(stream)
	defer sh.recover() // never crash caller
	defer sh.Close()   // ensure we close the stream helper

	sh.Metadata(ctx, "starting", 0, 0)

	// 1. Get Agent Details from DB.
	agentDetail, err := s.getAgentConfig(ctx, tenant, sh)
	if err != nil {
		return err
	}

	// Update metadata - agent found
	sh.Metadata(ctx, "processing_input", 0, 0)

	// 2. Initialize AgentFlow with stream helper as reporter.
	chunkRepository := odm.CollectionOf[db.ChunkModel](s.mongo, tenant)
	vectorRepository := odm.CollectionOf[db.ChunkAnnModel](s.mongo, tenant)

	llmClient, model, miniModel := s.getLLMClientAndModel(req.Model)

	agentFlow := agent.New(llmClient, sh, s.embdder, chunkRepository, vectorRepository)

	// 3. Execute AgentFlow pipeline
	result := agentFlow.
		ExtractQueries(ctx, miniModel, req.Text, agentDetail.Capability).
		Search(ctx).
		SummarizeContext(ctx, miniModel, req.Text).
		GenerateAnswer(ctx, model, req.Text, agentDetail.Capability)

	// 4. Handle final result or error
	if result.Err != nil {
		// Error already reported through the reporter interface
		logger.Error("AgentFlow execution failed", zap.Error(result.Err))
		return result.Err
	}

	s.saveSession(ctx, result, session, tenant, req.Text)

	// Send final completion
	totalResults := len(result.SearchResults)
	totalQueries := len(result.SearchQueries)

	logger.Info("AgentFlow execution completed successfully",
		zap.Int("total_queries", totalQueries),
		zap.Int("total_results", totalResults))

	sh.sendComplete("completed", totalResults, totalQueries)
	return nil
}

func (s *AgentService) getSession(ctx context.Context, tenant, userId, sessionId string) *db.SessionModel {
	session := db.NewSessionModel(userId, sessionId)
	if len(sessionId) > 0 {
		existingSession, err := async.Await(odm.CollectionOf[db.SessionModel](s.mongo, tenant).FindOneByID(ctx, sessionId))
		if err != nil {
			logger.Error("Error finding existing session", zap.String("session_id", sessionId), zap.Error(err))
		} else if existingSession != nil {
			logger.Info("Using existing session", zap.String("session_id", sessionId))
			session = existingSession
		}
	}

	return session
}

func (s *AgentService) getAgentConfig(ctx context.Context, tenant string, sh *streamHelper) (*db.AgentModel, error) {
	defaultAgentName := "default-agent"
	agentDetail, err := async.Await(odm.CollectionOf[db.AgentModel](s.mongo, tenant).FindOneByID(ctx, defaultAgentName))
	if err != nil {
		logger.Error("Error finding agent", zap.String("agent", defaultAgentName), zap.Error(err))
		sh.Error(ctx, "Failed to find agent configuration", "AGENT_NOT_FOUND")
		return nil, err
	}

	if agentDetail == nil {
		logger.Error("Agent not found", zap.String("agent", defaultAgentName))
		sh.Error(ctx, fmt.Sprintf("Agent %s not found", defaultAgentName), "AGENT_CONFIG_MISSING")
		return nil, err
	}

	return agentDetail, nil
}

func (s *AgentService) saveSession(ctx context.Context, result *agent.AgentFlow, session *db.SessionModel, tenant, input string) {
	turn := db.TurnModel{UserInput: input}
	turn.SearchQueries = result.SearchQueries
	turn.AgentAnswer = result.Answer
	turn.SearchResultChunkIds = linq.Map(result.SearchResults, func(r *db.ChunkModel) string {
		return r.ChunkID
	})
	session.Turns = append(session.Turns, turn)
	async.Await(odm.CollectionOf[db.SessionModel](s.mongo, tenant).Save(ctx, *session))
}

func (s *AgentService) getLLMClientAndModel(m string) (llm.LLMClient, string, string) {
	if m == "claude" {
		return s.anthropicClient, s.ccfgg.ClaudeVersion, s.ccfgg.ClaudeMini
	} else {
		return s.ollamaClient, s.ccfgg.OllamaModel, s.ccfgg.OllamaMiniModel
	}
}

type streamHelper struct {
	stream grpc.ServerStreamingServer[pb.AgentStreamChunk]
	flush  func() error
	wg     sync.WaitGroup
	queue  chan *pb.AgentStreamChunk
	once   sync.Once
}

func newStreamHelper(s grpc.ServerStreamingServer[pb.AgentStreamChunk]) *streamHelper {
	var flusher func() error
	if f, ok := s.(interface{ Flush() error }); ok {
		flusher = f.Flush
	}
	return &streamHelper{
		stream: s,
		flush:  flusher,
		queue:  make(chan *pb.AgentStreamChunk, 32), // buffered channel to queue chunks
	}
}

// Reporter interface implementation
func (h *streamHelper) Metadata(ctx context.Context, status string, estQueries, estResults int32) {
	h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Metadata{Metadata: &pb.StreamMetadata{
			Status:           status,
			EstimatedQueries: estQueries,
			EstimatedResults: estResults,
		}},
	})
}

func (h *streamHelper) Queries(ctx context.Context, q []string) {
	h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_SearchRequest{SearchRequest: &pb.SearchRequestChunk{
			Queries:      q,
			ChunkIndex:   0,
			IsFinalChunk: true,
		}},
	})
}

func (h *streamHelper) SearchResults(ctx context.Context, doc *db.ChunkModel, citationIdx, totalChunks int, isFinal bool) {
	// Convert to proto format
	searchResultProto := &pb.SearchResult{
		Sentences:     doc.Sentences,
		Title:         doc.Title,
		SectionPath:   doc.SectionPath,
		Source:        doc.SourceURI,
		URL:           doc.SourceURI,
		CitationIndex: int32(citationIdx),
		ChunkId:       doc.ChunkID,
	}

	h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_SearchResults{SearchResults: &pb.SearchResultsChunk{
			Results: []*pb.SearchResult{searchResultProto}, ChunkIndex: int32(citationIdx), TotalChunks: int32(totalChunks),
			IsFinalChunk: isFinal,
		}},
	})
}

func (h *streamHelper) Answer(ctx context.Context, ans string) {
	h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Answer{Answer: &pb.AnswerChunk{
			Content: ans,
			IsFinal: true,
		}},
	})
}

func (h *streamHelper) Error(ctx context.Context, code, msg string) {
	h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Error{Error: &pb.StreamError{
			ErrorMessage: msg,
			ErrorCode:    code,
		}},
	})
}

// Non-Reporter methods
func (h *streamHelper) sendComplete(status string, totalResults, totalQueries int) {
	h.send(&pb.AgentStreamChunk{
		ChunkType: &pb.AgentStreamChunk_Complete{Complete: &pb.StreamComplete{
			FinalStatus:      status,
			TotalResultsSent: int32(totalResults),
			TotalQueriesSent: int32(totalQueries),
		}},
	})
}

func (h *streamHelper) recover() {
	if r := recover(); r != nil {
		logger.Error("Stream panic recovered", zap.Any("panic", r))
	}
}

// Queues the chunks to be sent in queue channel
// and sends them in a separate goroutine.
func (h *streamHelper) send(c *pb.AgentStreamChunk) {
	h.once.Do(func() {
		h.wg.Add(1) // only one goroutine will be created

		go func() {
			defer h.wg.Done()

			for chunk := range h.queue {
				if err := h.stream.Send(chunk); err != nil {
					logger.Error("Failed to send chunk", zap.Error(err))
				}
				if h.flush != nil {
					_ = h.flush()
				}
			}
		}()
	})

	h.queue <- c
}

func (h *streamHelper) Close() {
	close(h.queue) // stop worker
	h.wg.Wait()    // wait for all go routines to finish
}
