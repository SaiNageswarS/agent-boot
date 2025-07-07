package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
)

type Reporter interface {
	Metadata(ctx context.Context, status string, estQueries, estResults int32)
	Queries(ctx context.Context, q []string)
	SearchResults(ctx context.Context, res []*db.ChunkModel)
	Answer(ctx context.Context, ans string)
	NotRelevant(ctx context.Context, reason string, q []string)
	Error(ctx context.Context, code, msg string)
}

type AgentFlow struct {
	chunkRepo  odm.OdmCollectionInterface[db.ChunkModel]
	vectorRepo odm.OdmCollectionInterface[db.ChunkAnnModel]

	embedder  llm.Embedder
	llmClient llm.LLMClient
	model     string
	rep       Reporter

	// agent flow outputs
	Err           error
	SearchQueries []string
	SearchResults []*db.ChunkModel
	Answer        string
}

func New(llmClient llm.LLMClient, model string, rep Reporter, embedder llm.Embedder, chunkRepo odm.OdmCollectionInterface[db.ChunkModel], vectorRepo odm.OdmCollectionInterface[db.ChunkAnnModel]) *AgentFlow {
	return &AgentFlow{
		llmClient:  llmClient,
		model:      model,
		rep:        rep,
		chunkRepo:  chunkRepo,
		vectorRepo: vectorRepo,
		embedder:   embedder,
	}
}

func (af *AgentFlow) ExtractQueries(ctx context.Context, userInput, agentCapability string) *AgentFlow {
	if af.Err != nil {
		return af
	}

	af.rep.Metadata(ctx, "extract_search_queries", 0, 0)
	resp, err := async.Await(
		prompts.ExtractSearchQueries(ctx, af.llmClient, af.model, userInput, agentCapability),
	)
	if err != nil {
		af.Err = fmt.Errorf("input extraction failed: %w", err)
		af.rep.Error(ctx, "INPUT_EXTRACTION_FAILED", err.Error())
		return af
	}

	if !resp.Relevant {
		af.Err = errors.New("input_not_relevant")
		af.rep.NotRelevant(ctx, resp.Reasoning, resp.SearchQueries)
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

	af.SearchResults = searchResults
	af.rep.SearchResults(ctx, searchResults)

	return af
}

func (af *AgentFlow) GenerateAnswer(ctx context.Context, userInput, agentCapability string) *AgentFlow {
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
		prompts.GenerateAnswer(ctx, af.llmClient, af.model, agentCapability, userInput, string(searchResultsJson)),
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
