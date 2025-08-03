package agent

import (
	"agent-boot/proto/schema"
	"context"
	"testing"
)

func TestExecuteTool(t *testing.T) {
	handlerCalled := false
	tool := MCPTool{
		Name:        "test-tool",
		Description: "Test tool",
		Handler: func(ctx context.Context, params map[string]string) <-chan *schema.ToolExecutionResultChunk {
			handlerCalled = true
			result := make(chan *schema.ToolExecutionResultChunk, 1)
			defer close(result)
			result <- NewToolResult("Test Result", []string{"Success"})

			return result
		},
		SummarizeContext: false,
	}

	agent := NewAgent(AgentConfig{Tools: []MCPTool{tool}})
	selection := &schema.SelectedTool{
		Name:       "test-tool",
		Parameters: map[string]string{"test": "value"},
		Query:      "What is the test?",
	}

	resultsChan := agent.ExecuteTool(context.Background(), selection)
	var result *schema.ToolExecutionResultChunk
	for res := range resultsChan {
		result = res
	}

	if !handlerCalled {
		t.Error("Tool handler was not called")
	}

	if result == nil {
		t.Error("Expected a result from the tool execution, got nil")
		return
	}

	if result.Title != "Test Result" {
		t.Errorf("Expected result title 'Test Result', got '%s'", result.Title)
	}
}
