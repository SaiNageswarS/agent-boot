package agent

import (
	"agent-boot/proto/schema"
	"context"
	"fmt"
	"strconv"
	"time"
)

// MCPTool represents a Model Context Protocol tool
type MCPTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
	// SummarizeContext enables automatic summarization of tool results using the mini model.
	// When enabled, each ToolResult's Sentences will be summarized with respect to the user's query.
	// Irrelevant content will be filtered out, making this ideal for RAG search and web search tools.
	SummarizeContext bool `json:"summarize_context"`
	Handler          func(ctx context.Context, params map[string]string) <-chan *schema.ToolExecutionResultChunk
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
