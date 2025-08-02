package agent

import (
	"context"
	"testing"
)

func TestExecuteTool(t *testing.T) {
	handlerCalled := false
	tool := MCPTool{
		Name:        "test-tool",
		Description: "Test tool",
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResultChunk, error) {
			handlerCalled = true
			return []*ToolResultChunk{NewToolResult("Test Result", []string{"Success"})}, nil
		},
	}

	agent := NewAgent(AgentConfig{})
	selection := ToolSelection{
		Tool:       tool,
		Parameters: map[string]interface{}{"test": "value"},
	}

	results, err := agent.ExecuteTool(context.Background(), selection)
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}

	if !handlerCalled {
		t.Error("Tool handler should have been called")
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Title != "Test Result" {
		t.Errorf("Expected title 'Test Result', got '%s'", results[0].Title)
	}
}
