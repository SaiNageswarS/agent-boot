package agentboot

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/memory"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	MiniModel    llm.LLMClient
	BigModel     llm.LLMClient
	ToolSelector llm.LLMClient
	SystemPrompt string
	Tools        []MCPTool
	MaxTokens    int
	MaxTurns     int

	// Conversation management
	ConversationManager *memory.ConversationManager
}

// Agent represents the main agent system
type Agent struct {
	config AgentConfig
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
