package agent

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	Name      string            `json:"name"`
	Template  string            `json:"template"`
	Variables []string          `json:"variables"`
	Metadata  map[string]string `json:"metadata"`
}

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	MiniModel llm.LLMClient
	BigModel  llm.LLMClient
	Tools     []MCPTool
	Prompt    PromptTemplate
	MaxTokens int
	MaxTurns  int
}

// Agent represents the main agent system
type Agent struct {
	config AgentConfig
}

// NewAgent creates a new agent instance
func NewAgent(config AgentConfig) *Agent {
	return &Agent{
		config: config,
	}
}

// MCPTool wraps an api.Tool and provides a handler for execution
type MCPTool struct {
	api.Tool
	// SummarizeContext enables automatic summarization of tool results using the mini model.
	// When enabled, each ToolResult's Sentences will be summarized with respect to the user's query.
	// Irrelevant content will be filtered out, making this ideal for RAG search and web search tools.
	SummarizeContext bool `json:"summarize_context"`
	Handler          func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk
}
