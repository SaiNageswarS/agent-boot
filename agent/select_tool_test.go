package agent

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

func TestSelectToolsWithNoTools(t *testing.T) {
	agent := NewAgentBuilder().
		WithMiniModel(&mockLLMClient{responses: []string{"No tools available"}, model: "test-model"}).
		Build()

	req := ToolSelectionRequest{
		Query:    "test query",
		Context:  "test context",
		MaxTools: 3,
	}

	selections, err := agent.SelectTools(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectTools should not error with no tools: %v", err)
	}

	if len(selections) != 0 {
		t.Errorf("Expected 0 selections with no tools, got %d", len(selections))
	}
}

func TestSelectToolsWithValidResponse(t *testing.T) {
	// Mock LLM response in structured text format
	mockResponse := `
TOOL_SELECTION_START
TOOL: calculator
CONFIDENCE: 0.9
REASONING: User asked for mathematical calculation
PARAMETERS:
expression: 2+2
TOOL_SELECTION_END
`

	calcTool := MCPTool{
		Tool: api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "calculator",
				Description: "Performs calculations",
				Parameters: struct {
					Type       string                      `json:"type"`
					Defs       any                         `json:"$defs,omitempty"`
					Items      any                         `json:"items,omitempty"`
					Required   []string                    `json:"required"`
					Properties map[string]api.ToolProperty `json:"properties"`
				}{
					Type:     "object",
					Required: []string{"expression"},
					Properties: map[string]api.ToolProperty{
						"expression": {
							Type:        []string{"string"},
							Description: "Mathematical expression to evaluate",
						},
					},
				},
			},
		},
		Handler: func(ctx context.Context, params map[string]string) <-chan *schema.ToolExecutionResultChunk {
			result := NewMathToolResult("2+2", "4", []string{"Step 1: 2 + 2 = 4"})
			ch := make(chan *schema.ToolExecutionResultChunk, 1)
			ch <- result
			close(ch)
			return ch
		},
	}

	agent := NewAgentBuilder().
		WithMiniModel(&mockLLMClient{responses: []string{mockResponse}, model: "test-model"}).
		AddTool(calcTool).
		Build()

	req := ToolSelectionRequest{
		Query:    "What is 2+2?",
		Context:  "Math question",
		MaxTools: 1,
	}

	selections, err := agent.SelectTools(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectTools failed: %v", err)
	}

	if len(selections) != 1 {
		t.Fatalf("Expected 1 selection, got %d", len(selections))
	}

	if selections[0].Name != "calculator" {
		t.Errorf("Expected tool 'calculator', got '%s'", selections[0].Name)
	}
}
