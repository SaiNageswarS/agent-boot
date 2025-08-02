package agent

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/core/llm"
	"github.com/SaiNageswarS/agent-boot/example/db"
	"github.com/SaiNageswarS/go-api-boot/dotenv"
	"github.com/SaiNageswarS/go-api-boot/embed"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/stretchr/testify/assert"
)

func TestAgentFlow(t *testing.T) {
	dotenv.LoadEnv("../.env")

	testQuery := "What is remedy for excessive fear of ghosts?"
	agentCapability := "health information analysis"

	mongoClient := odm.ProvideMongoClient()
	embedder := embed.ProvideJinaAIEmbeddingClient()
	llmClient := llm.ProvideAnthropicClient()

	testTenant := "devinderhealthcare"
	chunkRepository := odm.CollectionOf[db.ChunkModel](mongoClient, testTenant)
	vectorRepository := odm.CollectionOf[db.ChunkAnnModel](mongoClient, testTenant)

	model := "claude-3-5-sonnet-20241022"
	miniModel := "claude-3-5-haiku-20241022"

	agentFlow := New(
		llmClient, &ReporterMock{}, embedder, chunkRepository, vectorRepository)

	ctx := t.Context()
	result := agentFlow.
		ExtractQueries(ctx, miniModel, testQuery, agentCapability).
		Search(ctx).
		SummarizeContext(ctx, miniModel, testQuery).
		GenerateAnswer(ctx, model, testQuery, agentCapability)

	assert.True(t, result.Err == nil, "Expected no error in agent flow execution")
}

type ReporterMock struct{}

func (r *ReporterMock) Metadata(ctx context.Context, status string, estQueries, estResults int32) {
	// Mock implementation
}

func (r *ReporterMock) Queries(ctx context.Context, q []string) {
	// Mock implementation
}

func (r *ReporterMock) SearchResults(ctx context.Context, res *db.ChunkModel, citationIdx, totalChunks int, isFinal bool) {
	// Mock implementation
}

func (r *ReporterMock) Answer(ctx context.Context, ans string) {
	// Mock implementation
}

func (r *ReporterMock) NotRelevant(ctx context.Context, reason string, q []string) {
	// Mock implementation
}

func (r *ReporterMock) Error(ctx context.Context, code, msg string) {
	// Mock implementation
}
