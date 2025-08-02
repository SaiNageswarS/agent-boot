package agent

import (
	"context"
	"fmt"
	"time"
)

// MCPTool represents a Model Context Protocol tool
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	// SummarizeContext enables automatic summarization of tool results using the mini model.
	// When enabled, each ToolResult's Sentences will be summarized with respect to the user's query.
	// Irrelevant content will be filtered out, making this ideal for RAG search and web search tools.
	SummarizeContext bool `json:"summarize_context"`
	Handler          func(ctx context.Context, params map[string]interface{}) ([]*ToolResultChunk, error)
}

// ToolResultChunk represents a standardized format for tool execution results
type ToolResultChunk struct {
	// Primary content - can be multiple sentences or a single result
	Sentences []string `json:"sentences"`

	// Source attribution - where the information came from
	Attribution string `json:"attributions,omitempty"`

	// Title or summary of the result
	Title string `json:"title,omitempty"`

	// Metadata for additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMathToolResult creates a ToolResult specifically for mathematical calculations
func NewToolResult(title string, sentences []string) *ToolResultChunk {
	return &ToolResultChunk{
		Title:     title,
		Sentences: sentences,
		Metadata:  make(map[string]interface{}),
	}
}

func NewMathToolResult(expression string, result string, steps []string) []*ToolResultChunk {
	sentences := []string{fmt.Sprintf("%s = %s", expression, result)}
	if len(steps) > 0 {
		sentences = append(sentences, "Calculation steps:")
		sentences = append(sentences, steps...)
	}

	toolResult := NewToolResult("Mathematical Calculation", sentences)
	toolResult.Metadata["expression"] = expression
	toolResult.Metadata["result"] = result
	toolResult.Metadata["calculation_type"] = "arithmetic"

	return []*ToolResultChunk{toolResult}
}

// NewDateTimeToolResult creates a ToolResult for date/time operations
func NewDateTimeToolResult(operation string, result string, timezone string) []*ToolResultChunk {
	sentences := []string{fmt.Sprintf("%s: %s", operation, result)}

	toolResult := NewToolResult("Date/Time Operation", sentences)
	toolResult.Metadata["operation"] = operation
	toolResult.Metadata["timezone"] = timezone
	toolResult.Metadata["timestamp"] = time.Now().Unix()

	return []*ToolResultChunk{toolResult}
}
