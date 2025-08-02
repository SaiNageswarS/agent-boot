package agent

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/core/llm"
)

func TestSelectToolsWithNoTools(t *testing.T) {
	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: "No tools available"},
			Model:  "test-model",
		},
	})

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

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: mockResponse},
			Model:  "test-model",
		},
		Tools: []MCPTool{
			{
				Name:        "calculator",
				Description: "Performs calculations",
				Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResultChunk, error) {
					return NewMathToolResult("2+2", "4", nil), nil
				},
			},
		},
	})

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

	if selections[0].Tool.Name != "calculator" {
		t.Errorf("Expected tool 'calculator', got '%s'", selections[0].Tool.Name)
	}

	if selections[0].Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", selections[0].Confidence)
	}
}
