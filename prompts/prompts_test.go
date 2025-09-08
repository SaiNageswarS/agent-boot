package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderSummarizationPrompt(t *testing.T) {
	// Test basic summarization prompt rendering
	systemPrompt, userPrompt, err := RenderSummarizationPrompt("What is machine learning?", "Machine learning is a subset of artificial intelligence that enables computers to learn from data. It uses algorithms to find patterns and make predictions.", "")

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, systemPrompt)
	assert.NotEmpty(t, userPrompt)

	// Verify system prompt contains expected content
	expectedSystemContent := []string{
		"text summarization expert",
		"relevant to the user's question",
		"IRRELEVANT",
		"important facts, numbers, and key details",
	}

	for _, expected := range expectedSystemContent {
		assert.Contains(t, systemPrompt, expected)
	}

	// Verify user prompt contains expected content
	expectedUserContent := []string{
		"What is machine learning?",
		"Machine learning is a subset of artificial intelligence",
		"summarize the above content",
	}

	for _, expected := range expectedUserContent {
		assert.Contains(t, userPrompt, expected)
	}
}

func TestRenderSummarizationPromptEmptyContent(t *testing.T) {
	// Test with empty content
	systemPrompt, userPrompt, err := RenderSummarizationPrompt("Test query", "", "")

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, systemPrompt, "System prompt should not be empty even with empty content")
	assert.NotEmpty(t, userPrompt, "User prompt should not be empty even with empty content")
	assert.Contains(t, userPrompt, "Test query", "User prompt should contain the query")
}

func TestRenderSummarizationPromptSpecialCharacters(t *testing.T) {
	// Test with special characters
	systemPrompt, userPrompt, err := RenderSummarizationPrompt("Calculate 2+2 & search for \"golang\"", "Content with special chars: <>&\"'", "")

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, systemPrompt, "text summarization expert", "System prompt should contain expected content")
	assert.Contains(t, userPrompt, "Calculate 2+2 & search for \"golang\"", "User prompt should preserve special characters in query")
	assert.Contains(t, userPrompt, "Content with special chars: <>&\"'", "User prompt should preserve special characters in content")
}

func TestRenderSummarizationPromptConsistency(t *testing.T) {
	// Test that multiple calls with same data produce same output
	sys1, user1, err1 := RenderSummarizationPrompt("test", "test content", "")
	sys2, user2, err2 := RenderSummarizationPrompt("test", "test content", "")

	// Assertions
	assert.NoError(t, err1, "First render should not fail")
	assert.NoError(t, err2, "Second render should not fail")
	assert.Equal(t, sys1, sys2, "System prompts should be consistent between calls")
	assert.Equal(t, user1, user2, "User prompts should be consistent between calls")
}

func TestRenderSummarizationPromptWithToolInputs(t *testing.T) {
	// Test with tool inputs
	toolInputs := "Tool: `calculator`\n\nParameters:\n- **expression**: 2+2\n- **format**: decimal"

	systemPrompt, userPrompt, err := RenderSummarizationPrompt("Calculate 2+2", "The calculation result is 4", toolInputs)

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, systemPrompt, "tool inputs", "System prompt should mention tool inputs")
	assert.Contains(t, userPrompt, "Tool Inputs:", "User prompt should contain Tool Inputs section")
	assert.Contains(t, userPrompt, "calculator", "User prompt should contain tool inputs content")
	assert.Contains(t, userPrompt, "user's question and tool inputs", "User prompt should mention both question and tool inputs in instructions")
}

func TestRenderToolSelectionPrompt(t *testing.T) {
	// Test data
	turn := 1

	// Test the function
	systemPrompt, err := RenderToolSelectionPrompt(turn)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, systemPrompt)

	// Check that system prompt contains expected content
	assert.Contains(t, systemPrompt, "intelligent tool selection assistant")
	assert.Contains(t, systemPrompt, "Multi-Step Reasoning")
	assert.Contains(t, systemPrompt, "Information Dependencies")
	assert.Contains(t, systemPrompt, "This is turn 1 of the conversation")
}

func TestRenderToolSelectionPromptFirstTurn(t *testing.T) {
	// Test data for first turn
	turn := 0

	// Test the function
	systemPrompt, err := RenderToolSelectionPrompt(turn)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, systemPrompt)

	// Check that system prompt contains first turn guidance
	assert.Contains(t, systemPrompt, "This is turn 0 of the conversation")
	assert.Contains(t, systemPrompt, "Focus on gathering basic foundational information first")
}

func TestRenderToolSelectionPromptEmptyTools(t *testing.T) {
	// Test with turn 0
	turn := 0

	// Test the function
	systemPrompt, err := RenderToolSelectionPrompt(turn)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, systemPrompt)
	assert.Contains(t, systemPrompt, "intelligent tool selection assistant")
}
