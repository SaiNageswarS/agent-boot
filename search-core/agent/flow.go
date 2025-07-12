package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.uber.org/zap"
)

type Reporter interface {
	Metadata(ctx context.Context, status string, estQueries, estResults int32)
	Queries(ctx context.Context, q []string)
	SearchResults(ctx context.Context, res *db.ChunkModel, citationIdx, totalChunks int, isFinal bool)
	Answer(ctx context.Context, ans string)
	Error(ctx context.Context, code, msg string)
}

type AgentFlow struct {
	chunkRepo  odm.OdmCollectionInterface[db.ChunkModel]
	vectorRepo odm.OdmCollectionInterface[db.ChunkAnnModel]

	embedder  llm.Embedder
	llmClient llm.LLMClient
	rep       Reporter

	// agent flow outputs
	Err           error
	SearchQueries []string
	SearchResults []*db.ChunkModel
	Answer        string
}

func New(llmClient llm.LLMClient, rep Reporter, embedder llm.Embedder, chunkRepo odm.OdmCollectionInterface[db.ChunkModel], vectorRepo odm.OdmCollectionInterface[db.ChunkAnnModel]) *AgentFlow {
	return &AgentFlow{
		llmClient:  llmClient,
		rep:        rep,
		chunkRepo:  chunkRepo,
		vectorRepo: vectorRepo,
		embedder:   embedder,
	}
}

func (af *AgentFlow) ExtractQueries(ctx context.Context, model, userInput, agentCapability string) *AgentFlow {
	if af.Err != nil {
		return af
	}

	af.rep.Metadata(ctx, "extract_search_queries", 0, 0)
	resp, err := async.Await(
		prompts.ExtractSearchQueries(ctx, af.llmClient, model, userInput, agentCapability),
	)
	if err != nil {
		af.Err = fmt.Errorf("input extraction failed: %w", err)
		af.rep.Error(ctx, "INPUT_EXTRACTION_FAILED", err.Error())
		return af
	}

	af.SearchQueries = resp.SearchQueries
	af.rep.Queries(ctx, resp.SearchQueries)
	return af
}

func (af *AgentFlow) Search(ctx context.Context) *AgentFlow {
	if af.Err != nil {
		return af
	}

	if len(af.SearchQueries) == 0 {
		af.Err = errors.New("no search queries provided")
		af.rep.Error(ctx, "NO_SEARCH_QUERIES", "No search queries extracted")
		return af
	}

	af.rep.Metadata(ctx, "searching", int32(len(af.SearchQueries)), 0)
	searchStep := NewSearchStep(af.chunkRepo, af.vectorRepo, af.embedder)

	searchResults, err := searchStep.Run(ctx, af.SearchQueries)
	if err != nil {
		af.Err = fmt.Errorf("search failed: %w", err)
		af.rep.Error(ctx, "SEARCH_FAILED", err.Error())
		return af
	}

	logger.Info("Search completed", zap.Int("total_results", len(searchResults)))
	af.SearchResults = searchResults
	// af.rep.SearchResults(ctx, searchResults)

	return af
}

func (af *AgentFlow) SummarizeContext(ctx context.Context, model, userInput string) *AgentFlow {
	if af.Err != nil {
		return af
	}
	if len(af.SearchResults) == 0 {
		af.Err = errors.New("no search results found")
		af.rep.Error(ctx, "NO_SEARCH_RESULTS", "No search results available for summarization")
		return af
	}

	af.rep.Metadata(ctx, "summarizing_context", int32(len(af.SearchResults)), 0)

	// ── 1. group consecutive chunks by section ──────────────────────────────
	type sectionJob struct {
		head      *db.ChunkModel
		sentences []string
	}
	jobs := make([]sectionJob, 0)

	for i := 0; i < len(af.SearchResults); {
		head := af.SearchResults[i]
		section, _ := getSectionAndIndex(head)

		buf := append([]string(nil), head.Sentences...)
		j := i + 1
		for j < len(af.SearchResults) {
			if s, _ := getSectionAndIndex(af.SearchResults[j]); s != section {
				break
			}
			buf = append(buf, af.SearchResults[j].Sentences...)
			j++
		}
		jobs = append(jobs, sectionJob{head: head, sentences: buf})
		i = j
	}

	logger.Info("[Context Summarization] Grouped sections", zap.Int("total_sections", len(jobs)))

	// ── 2. parallel summarisation & immediate streaming ─────────────────────
	var wg sync.WaitGroup
	var streamMu sync.Mutex

	var citation int32            // 1-based, only for streamed items
	remaining := int32(len(jobs)) // decremented for every finished job

	for _, job := range jobs {
		wg.Add(1)
		go func(j sectionJob) {
			defer wg.Done()

			summary, err := async.Await(
				prompts.SummarizeContext(ctx, af.llmClient, model, userInput, j.sentences),
			)

			// decide whether to keep this section
			if err != nil || len(summary) == 0 {
				atomic.AddInt32(&remaining, -1) // skipped
				return                          // nothing streamed
			}

			j.head.Sentences = summary

			// ── critical section: write to stream ──────────────────────────
			streamMu.Lock()
			idx := atomic.AddInt32(&citation, 1) // assign citation #
			last := atomic.AddInt32(&remaining, -1) == 0
			af.rep.SearchResults(ctx, j.head, int(idx), len(af.SearchResults), last)
			streamMu.Unlock()
		}(job)
	}

	wg.Wait()

	// collect only the sections that were summarised & streamed
	out := make([]*db.ChunkModel, 0, citation)
	for _, j := range jobs {
		if len(j.head.Sentences) > 0 && j.head != nil {
			out = append(out, j.head)
		}
	}

	logger.Info("Context summarization completed",
		zap.Int("total_sections_kept", len(out)),
		zap.Int("total_sections_skipped", len(jobs)-len(out)))

	af.SearchResults = out
	return af
}

func (af *AgentFlow) GenerateAnswer(ctx context.Context, model, userInput, agentCapability string) *AgentFlow {
	if af.Err != nil {
		return af
	}

	if len(af.SearchResults) == 0 {
		af.Err = errors.New("no search results found")
		af.rep.Error(ctx, "NO_SEARCH_RESULTS", "No search results available for generating answer")
		return af
	}

	af.rep.Metadata(ctx, "generating_answer", int32(len(af.SearchQueries)), int32(len(af.SearchResults)))

	searchResultsJson, err := json.Marshal(af.SearchResults)
	if err != nil {
		af.Err = fmt.Errorf("search results marshal failed: %w", err)
		af.rep.Error(ctx, "SEARCH_RESULTS_MARSHAL_FAILED", err.Error())
		return af
	}

	answer, err := async.Await(
		prompts.GenerateAnswer(ctx, af.llmClient, model, agentCapability, userInput, string(searchResultsJson)),
	)

	if err != nil {
		af.Err = fmt.Errorf("answer generation failed: %w", err)
		af.rep.Error(ctx, "ANSWER_GENERATION_FAILED", err.Error())
		return af
	}

	af.Answer = answer
	af.rep.Answer(ctx, answer)
	return af
}

func (af *AgentFlow) IsSuccess() bool {
	return af.Err == nil
}

func (af *AgentFlow) GetError() error {
	return af.Err
}
