package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// MCPTool wraps an api.Tool and provides a handler for execution
type MCPTool struct {
	api.Tool
	// SummarizeContext enables automatic summarization of tool results using the mini model.
	// When enabled, each ToolResult's Sentences will be summarized with respect to the user's query.
	// Irrelevant content will be filtered out, making this ideal for RAG search and web search tools.
	SummarizeContext bool `json:"summarize_context"`
	Handler          func(ctx context.Context, params map[string]string) <-chan *schema.ToolExecutionResultChunk
}

// Turn represents a single turn in the agent execution
type Turn struct {
	TurnNumber int            `json:"turn_number"`
	ToolCalls  []api.ToolCall `json:"tool_calls,omitempty"`
	Answer     string         `json:"answer,omitempty"`
	IsComplete bool           `json:"is_complete"`
	Error      string         `json:"error,omitempty"`
}

// NewMathToolResult creates a ToolResult specifically for mathematical calculations
func NewToolResult(title string, sentences []string) *schema.ToolExecutionResultChunk {
	return &schema.ToolExecutionResultChunk{
		Title:     title,
		Sentences: sentences,
		Metadata:  make(map[string]string),
	}
}

func NewMathToolResult(expression string, result string, steps []string) *schema.ToolExecutionResultChunk {
	sentences := []string{fmt.Sprintf("%s = %s", expression, result)}
	if len(steps) > 0 {
		sentences = append(sentences, "Calculation steps:")
		sentences = append(sentences, steps...)
	}

	toolResult := NewToolResult("Mathematical Calculation", sentences)
	toolResult.Metadata["expression"] = expression
	toolResult.Metadata["result"] = result
	toolResult.Metadata["calculation_type"] = "arithmetic"

	return toolResult
}

// NewDateTimeToolResult creates a ToolResult for date/time operations
func NewDateTimeToolResult(operation string, result string, timezone string) *schema.ToolExecutionResultChunk {
	sentences := []string{fmt.Sprintf("%s: %s", operation, result)}

	toolResult := NewToolResult("Date/Time Operation", sentences)
	toolResult.Metadata["operation"] = operation
	toolResult.Metadata["timezone"] = timezone
	toolResult.Metadata["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)

	return toolResult
}

// ToAPITools converts MCPTools to api.Tools for native tool calling
func ToAPITools(tools []MCPTool) []api.Tool {
	apiTools := make([]api.Tool, len(tools))
	for i, tool := range tools {
		apiTools[i] = tool.Tool
	}
	return apiTools
}

// FindMCPToolByName finds an MCPTool by its function name
func FindMCPToolByName(tools []MCPTool, name string) *MCPTool {
	for _, tool := range tools {
		if tool.Function.Name == name {
			return &tool
		}
	}
	return nil
}
