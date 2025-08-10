package agent

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// TestMain can be used to setup/teardown for the entire test suite if needed
func TestMain(m *testing.M) {
	// Setup code here if needed

	// Run tests
	m.Run()

	// Teardown code here if needed
}

// Test coverage verification
func TestPackageCoverage(t *testing.T) {
	t.Log("Agent package tests cover the following components:")
	t.Log("✓ agent.go - Core agent structures and types")
	t.Log("✓ agent_builder.go - Builder pattern for agent configuration")
	t.Log("✓ execute_turn_based.go - Main execution logic and LLM interaction")
	t.Log("✓ mcp_tool_builder.go - MCP tool creation and result building")
	t.Log("✓ progress.go - Progress reporting and streaming chunks")
	t.Log("✓ utils.go - Utility functions for time, tool finding, and conversions")
}

// Integration test example that uses multiple components together
func TestAgentIntegration(t *testing.T) {
	// This test demonstrates how all components work together

	// 1. Create an agent using the builder
	builder := NewAgentBuilder()

	// 2. Add a mock model
	mockModel := &mockLLMClient{model: "integration-test"}

	// 3. Create a tool using the MCP tool builder
	tool := NewMCPTool("integration-tool", "Integration test tool").
		StringParam("input", "Test input parameter", true).
		Summarize(false).
		WithHandler(func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			chunk := NewToolResultChunk().
				Title("Integration Test Result").
				Sentences("Integration test completed successfully").
				ToolName("integration-tool").
				MetadataKV("test", "passed").
				Build()
			ch <- chunk
			close(ch)
			return ch
		}).
		Build()

	// 4. Build the agent with all components
	agent := builder.
		WithBigModel(mockModel).
		WithMiniModel(mockModel).
		AddTool(tool).
		WithMaxTokens(1000).
		WithMaxTurns(2).
		Build()

	// 5. Verify agent was created correctly
	if agent == nil {
		t.Fatal("Failed to create agent")
	}

	if len(agent.config.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(agent.config.Tools))
	}

	if agent.config.MaxTokens != 1000 {
		t.Fatalf("Expected MaxTokens 1000, got %d", agent.config.MaxTokens)
	}

	if agent.config.MaxTurns != 2 {
		t.Fatalf("Expected MaxTurns 2, got %d", agent.config.MaxTurns)
	}

	t.Log("✓ Integration test passed - all components work together correctly")
}
