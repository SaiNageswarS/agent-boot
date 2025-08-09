package agent

import (
	"fmt"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
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

func (a *Agent) GetToolByName(name string) *MCPTool {
	for _, tool := range a.config.Tools {
		if tool.Function.Name == name {
			return &tool
		}
	}
	return nil
}

func (a *Agent) shouldUseBigModel(query string, toolResults []string) bool {
	// Simple heuristics to decide when to use the big model

	// Use big model for complex queries (longer than 100 chars)
	if len(query) > 100 {
		return true
	}

	// Use big model when we have tool results to synthesize
	if len(toolResults) > 1 {
		return true
	}

	// Use big model for certain keywords indicating complexity
	complexKeywords := []string{
		"analyze", "compare", "summarize", "explain", "detailed",
		"comprehensive", "research", "investigate", "complex",
	}

	queryLower := strings.ToLower(query)
	for _, keyword := range complexKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}

	return false
}

// GetAvailableTools returns a list of all available tools
func (a *Agent) GetAvailableTools() []MCPTool {
	return a.config.Tools
}

// formatToolResult formats a ToolResult into a human-readable string for prompts
func (a *Agent) formatToolResult(toolName string, result *schema.ToolExecutionResultChunk) string {
	var formatted strings.Builder

	formatted.WriteString(fmt.Sprintf("Tool %s result:", toolName))

	if result.Title != "" {
		formatted.WriteString(fmt.Sprintf("\nTitle: %s", result.Title))
	}

	if len(result.Sentences) > 0 {
		formatted.WriteString("\nContent:")
		for _, sentence := range result.Sentences {
			formatted.WriteString(fmt.Sprintf("\n- %s", sentence))
		}
	}

	if result.Attribution != "" {
		formatted.WriteString(fmt.Sprintf("\nSource: %s", result.Attribution))
	}

	if len(result.Metadata) > 0 {
		formatted.WriteString("\nAdditional Info:")
		for key, value := range result.Metadata {
			formatted.WriteString(fmt.Sprintf("\n- %s: %v", key, value))
		}
	}

	return formatted.String()
}

func (a *Agent) getMaxTokens() int {
	if a.config.MaxTokens > 0 {
		return a.config.MaxTokens
	}
	return 2000 // Default max tokens
}

func getMaxTools(requested int) int {
	if requested > 0 && requested <= 10 {
		return requested
	}
	return 3 // Default max tools
}
