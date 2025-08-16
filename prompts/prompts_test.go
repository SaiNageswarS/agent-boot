package prompts

import (
	"strings"
	"testing"
)

func TestRenderSummarizationPrompt(t *testing.T) {
	// Test basic summarization prompt rendering
	systemPrompt, userPrompt, err := RenderSummarizationPrompt("What is machine learning?", "Machine learning is a subset of artificial intelligence that enables computers to learn from data. It uses algorithms to find patterns and make predictions.", "")
	if err != nil {
		t.Fatalf("Failed to render summarization prompt: %v", err)
	}

	// Verify system prompt contains expected content
	expectedSystemContent := []string{
		"text summarization expert",
		"relevant to the user's question",
		"IRRELEVANT",
		"important facts, numbers, and key details",
	}

	for _, expected := range expectedSystemContent {
		if !strings.Contains(systemPrompt, expected) {
			t.Errorf("System prompt should contain '%s'", expected)
		}
	}

	// Verify user prompt contains expected content
	expectedUserContent := []string{
		"What is machine learning?",
		"Machine learning is a subset of artificial intelligence",
		"summarize the above content",
	}

	for _, expected := range expectedUserContent {
		if !strings.Contains(userPrompt, expected) {
			t.Errorf("User prompt should contain '%s'", expected)
		}
	}

	// Verify both prompts are non-empty
	if systemPrompt == "" {
		t.Error("System prompt should not be empty")
	}

	if userPrompt == "" {
		t.Error("User prompt should not be empty")
	}
}

func TestRenderSummarizationPromptEmptyContent(t *testing.T) {
	// Test with empty content
	systemPrompt, userPrompt, err := RenderSummarizationPrompt("Test query", "", "")
	if err != nil {
		t.Fatalf("Failed to render prompt with empty content: %v", err)
	}

	// Should still work with empty content
	if systemPrompt == "" {
		t.Error("System prompt should not be empty even with empty content")
	}

	if userPrompt == "" {
		t.Error("User prompt should not be empty even with empty content")
	}

	// Verify query is included
	if !strings.Contains(userPrompt, "Test query") {
		t.Error("User prompt should contain the query")
	}
}

func TestRenderSummarizationPromptSpecialCharacters(t *testing.T) {
	// Test with special characters
	systemPrompt, userPrompt, err := RenderSummarizationPrompt("Calculate 2+2 & search for \"golang\"", "Content with special chars: <>&\"'", "")
	if err != nil {
		t.Fatalf("Failed to render prompt with special characters: %v", err)
	}

	// Verify special characters are preserved in both prompts
	if !strings.Contains(systemPrompt, "text summarization expert") {
		t.Error("System prompt should contain expected content")
	}

	if !strings.Contains(userPrompt, "Calculate 2+2 & search for \"golang\"") {
		t.Error("User prompt should preserve special characters in query")
	}

	if !strings.Contains(userPrompt, "Content with special chars: <>&\"'") {
		t.Error("User prompt should preserve special characters in content")
	}
}

func TestRenderSummarizationPromptConsistency(t *testing.T) {
	// Test that multiple calls with same data produce same output
	sys1, user1, err1 := RenderSummarizationPrompt("test", "test content", "")
	if err1 != nil {
		t.Fatalf("First render failed: %v", err1)
	}

	sys2, user2, err2 := RenderSummarizationPrompt("test", "test content", "")
	if err2 != nil {
		t.Fatalf("Second render failed: %v", err2)
	}

	if sys1 != sys2 {
		t.Error("System prompts should be consistent between calls")
	}

	if user1 != user2 {
		t.Error("User prompts should be consistent between calls")
	}
}

func TestRenderSummarizationPromptWithToolInputs(t *testing.T) {
	// Test with tool inputs
	toolInputs := "Tool: `calculator`\n\nParameters:\n- **expression**: 2+2\n- **format**: decimal"

	systemPrompt, userPrompt, err := RenderSummarizationPrompt("Calculate 2+2", "The calculation result is 4", toolInputs)
	if err != nil {
		t.Fatalf("Failed to render prompt with tool inputs: %v", err)
	}

	// Verify system prompt mentions tool inputs
	if !strings.Contains(systemPrompt, "tool inputs") {
		t.Error("System prompt should mention tool inputs")
	}

	// Verify user prompt contains tool inputs section
	if !strings.Contains(userPrompt, "Tool Inputs:") {
		t.Error("User prompt should contain Tool Inputs section")
	}

	if !strings.Contains(userPrompt, "calculator") {
		t.Error("User prompt should contain tool inputs content")
	}

	// Verify the instructions are updated to mention tool inputs
	if !strings.Contains(userPrompt, "user's question and tool inputs") {
		t.Error("User prompt should mention both question and tool inputs in instructions")
	}
}
