package agent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
)

type Reporter interface {
	Metadata(status string, estQueries, estResults int32)
	Queries(q []string)
	SearchResults(ctx context.Context, res []*db.ChunkModel)
	Answer(ans string)
	NotRelevant(reason string, q []string)
	Error(code, msg string)
	Final(status string)
}

type AgentFlow struct {
	chunkRepo  odm.OdmCollectionInterface[db.ChunkModel]
	vectorRepo odm.OdmCollectionInterface[db.ChunkAnnModel]

	embedder  llm.Embedder
	llmClient llm.LLMClient
	model     string
	rep       Reporter

	// agent flow outputs
	err           error
	SearchQueries []string
	SearchResults []*db.ChunkModel
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
	if af.err != nil {
		return af
	}

	af.rep.Metadata("extract_search_queries", 0, 0)
	resp, err := async.Await(
		prompts.ExtractSearchQueries(ctx, af.llmClient, af.model, userInput, agentCapability),
	)
	if err != nil {
		af.err = err
		af.rep.Error("INPUT_EXTRACTION_FAILED", err.Error())
		return af
	}

	if !resp.Relevant {
		af.err = errors.New("input_not_relevant")
		af.rep.NotRelevant(resp.Reasoning, resp.SearchQueries)
		af.rep.Final("not_relevant")
		return af
	}

	af.SearchQueries = resp.SearchQueries
	af.rep.Queries(resp.SearchQueries)
	return af
}

func (af *AgentFlow) Search(ctx context.Context) *AgentFlow {
	if af.err != nil {
		return af
	}

	if len(af.SearchQueries) == 0 {
		af.err = errors.New("no search queries provided")
		af.rep.Error("NO_SEARCH_QUERIES", "No search queries extracted")
		return af
	}

	af.rep.Metadata("searching", int32(len(af.SearchQueries)), 0)
	searchStep := NewSearchStep(af.chunkRepo, af.vectorRepo, af.embedder)

	searchResults, err := searchStep.Run(ctx, af.SearchQueries)

	if err != nil {
		af.err = err
		af.rep.Error("SEARCH_FAILED", err.Error())
		return af
	}

	af.SearchResults = searchResults
	af.rep.SearchResults(ctx, searchResults)
	return af
}

func (af *AgentFlow) GenerateAnswer(ctx context.Context, userInput, agentCapability string) *AgentFlow {
	if af.err != nil {
		return af
	}

	if len(af.SearchResults) == 0 {
		af.err = errors.New("no search results found")
		af.rep.Error("NO_SEARCH_RESULTS", "No search results available for generating answer")
		return af
	}

	af.rep.Metadata("generating_answer", int32(len(af.SearchQueries)), int32(len(af.SearchResults)))

	searchResultsJson, err := json.Marshal(af.SearchResults)
	if err != nil {
		af.err = err
		af.rep.Error("SEARCH_RESULTS_MARSHAL_FAILED", err.Error())
		return af
	}

	answer, err := async.Await(
		prompts.GenerateAnswer(ctx, af.llmClient, af.model, agentCapability, userInput, string(searchResultsJson)),
	)

	if err != nil {
		af.err = err
		af.rep.Error("ANSWER_GENERATION_FAILED", err.Error())
		return af
	}

	af.rep.Answer(answer)
	return af
}
