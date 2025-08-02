package prompts

import (
	"strings"
	"testing"
)

func TestRenderToolSelectionPrompt(t *testing.T) {
	// Test basic prompt rendering
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{
			"- calculator: Performs basic arithmetic calculations",
			"- weather: Gets current weather information",
			"- search: Searches the web for information",
		},
		MaxTools: 2,
		Query:    "Calculate 5+5 and search for weather",
		Context:  "User needs help with math and weather",
	}

	systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
	if err != nil {
		t.Fatalf("Failed to render prompt: %v", err)
	}

	// Verify system prompt contains expected content
	expectedSystemContent := []string{
		"tool selection expert",
		"calculator: Performs basic arithmetic calculations",
		"weather: Gets current weather information",
		"search: Searches the web for information",
		"Maximum 2 tools",
	}

	for _, expected := range expectedSystemContent {
		if !strings.Contains(systemPrompt, expected) {
			t.Errorf("System prompt should contain '%s'", expected)
		}
	}

	// Verify user prompt contains expected content
	expectedUserContent := []string{
		"Calculate 5+5 and search for weather",
		"User needs help with math and weather",
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

func TestRenderToolSelectionPromptNoTools(t *testing.T) {
	// Test with no tools
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{},
		MaxTools:         0,
		Query:            "Help me with something",
		Context:          "No tools available",
	}

	systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
	if err != nil {
		t.Fatalf("Failed to render prompt with no tools: %v", err)
	}

	// Should still work with no tools
	if systemPrompt == "" {
		t.Error("System prompt should not be empty even with no tools")
	}

	if userPrompt == "" {
		t.Error("User prompt should not be empty even with no tools")
	}

	// Verify query is included
	if !strings.Contains(userPrompt, "Help me with something") {
		t.Error("User prompt should contain the query")
	}

	if !strings.Contains(userPrompt, "No tools available") {
		t.Error("User prompt should contain the context")
	}
}

func TestRenderToolSelectionPromptEmptyContext(t *testing.T) {
	// Test with empty context
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{
			"- calculator: Math tool",
		},
		MaxTools: 1,
		Query:    "Calculate something",
		Context:  "", // Empty context
	}

	systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
	if err != nil {
		t.Fatalf("Failed to render prompt with empty context: %v", err)
	}

	// Should work with empty context
	if !strings.Contains(systemPrompt, "calculator: Math tool") {
		t.Error("System prompt should contain tool description")
	}

	if !strings.Contains(userPrompt, "Calculate something") {
		t.Error("User prompt should contain query")
	}

	// Context section should handle empty context gracefully
	if !strings.Contains(systemPrompt, "Maximum 1 tools") {
		t.Error("System prompt should contain max tools setting")
	}
}

func TestRenderToolSelectionPromptManyTools(t *testing.T) {
	// Test with many tools
	tools := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		tools = append(tools, "- tool"+string(rune('0'+i))+": Description for tool "+string(rune('0'+i)))
	}

	data := ToolSelectionPromptData{
		ToolDescriptions: tools,
		MaxTools:         5,
		Query:            "Use multiple tools",
		Context:          "Complex task requiring multiple tools",
	}

	systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
	if err != nil {
		t.Fatalf("Failed to render prompt with many tools: %v", err)
	}

	// Verify all tools are included
	for i := 0; i < 10; i++ {
		expectedTool := "tool" + string(rune('0'+i)) + ": Description for tool " + string(rune('0'+i))
		if !strings.Contains(systemPrompt, expectedTool) {
			t.Errorf("System prompt should contain '%s'", expectedTool)
		}
	}

	// Verify max tools setting
	if !strings.Contains(systemPrompt, "Maximum 5 tools") {
		t.Error("System prompt should contain max tools setting")
	}

	// Verify user content
	if !strings.Contains(userPrompt, "Use multiple tools") {
		t.Error("User prompt should contain query")
	}

	if !strings.Contains(userPrompt, "Complex task requiring multiple tools") {
		t.Error("User prompt should contain context")
	}
}

func TestRenderToolSelectionPromptSpecialCharacters(t *testing.T) {
	// Test with special characters in descriptions and query
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{
			"- calc: Performs math with symbols like +, -, *, /",
			"- search: Searches with quotes \"like this\" and & symbols",
		},
		MaxTools: 2,
		Query:    "Calculate 2+2 & search for \"golang\"",
		Context:  "Testing special chars: <>&\"'",
	}

	systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
	if err != nil {
		t.Fatalf("Failed to render prompt with special characters: %v", err)
	}

	// Verify special characters are preserved
	if !strings.Contains(systemPrompt, "+, -, *, /") {
		t.Error("System prompt should preserve math symbols")
	}

	if !strings.Contains(systemPrompt, "\"like this\"") {
		t.Error("System prompt should preserve quotes")
	}

	if !strings.Contains(userPrompt, "Calculate 2+2 & search for \"golang\"") {
		t.Error("User prompt should preserve special characters in query")
	}

	if !strings.Contains(userPrompt, "Testing special chars: <>&\"'") {
		t.Error("User prompt should preserve special characters in context")
	}
}

func TestToolSelectionPromptDataStructure(t *testing.T) {
	// Test the data structure itself
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{"tool1", "tool2"},
		MaxTools:         3,
		Query:            "test query",
		Context:          "test context",
	}

	if len(data.ToolDescriptions) != 2 {
		t.Errorf("Expected 2 tool descriptions, got %d", len(data.ToolDescriptions))
	}

	if data.MaxTools != 3 {
		t.Errorf("Expected MaxTools to be 3, got %d", data.MaxTools)
	}

	if data.Query != "test query" {
		t.Errorf("Expected query 'test query', got '%s'", data.Query)
	}

	if data.Context != "test context" {
		t.Errorf("Expected context 'test context', got '%s'", data.Context)
	}
}

func TestRenderToolSelectionPromptConsistency(t *testing.T) {
	// Test that multiple calls with same data produce same output
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{"- test: tool"},
		MaxTools:         1,
		Query:            "test",
		Context:          "test",
	}

	sys1, user1, err1 := RenderToolSelectionPrompt(data)
	if err1 != nil {
		t.Fatalf("First render failed: %v", err1)
	}

	sys2, user2, err2 := RenderToolSelectionPrompt(data)
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

func BenchmarkRenderToolSelectionPrompt(b *testing.B) {
	// Benchmark the prompt rendering
	data := ToolSelectionPromptData{
		ToolDescriptions: []string{
			"- calculator: Performs basic arithmetic calculations",
			"- weather: Gets current weather information",
			"- search: Searches the web for information",
		},
		MaxTools: 2,
		Query:    "Sample query for benchmarking",
		Context:  "Sample context for benchmarking",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
		if err != nil {
			b.Fatal(err)
		}
		// Use the results to prevent optimization
		_ = systemPrompt
		_ = userPrompt
	}
}

func BenchmarkRenderToolSelectionPromptManyTools(b *testing.B) {
	// Benchmark with many tools
	tools := make([]string, 50)
	for i := 0; i < 50; i++ {
		tools[i] = "- tool" + string(rune('0'+(i%10))) + ": Description for tool " + string(rune('0'+(i%10)))
	}

	data := ToolSelectionPromptData{
		ToolDescriptions: tools,
		MaxTools:         10,
		Query:            "Complex query with many tools available",
		Context:          "Benchmark context with many tools",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
		if err != nil {
			b.Fatal(err)
		}
		_ = systemPrompt
		_ = userPrompt
	}
}

// Tests for RenderSummarizationPrompt

func TestRenderSummarizationPrompt(t *testing.T) {
	// Test basic summarization prompt rendering
	data := SummarizationPromptData{
		Query:   "What is machine learning?",
		Content: "Machine learning is a subset of artificial intelligence that enables computers to learn from data. It uses algorithms to find patterns and make predictions.",
	}

	systemPrompt, userPrompt, err := RenderSummarizationPrompt(data)
	if err != nil {
		t.Fatalf("Failed to render summarization prompt: %v", err)
	}

	// Verify system prompt contains expected content
	expectedSystemContent := []string{
		"text summarization expert",
		"relevant to the user's question",
		"IRRELEVANT",
		"1-3 clear, concise sentences",
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
	data := SummarizationPromptData{
		Query:   "Test query",
		Content: "",
	}

	systemPrompt, userPrompt, err := RenderSummarizationPrompt(data)
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
	data := SummarizationPromptData{
		Query:   "Calculate 2+2 & search for \"golang\"",
		Content: "Content with special chars: <>&\"'",
	}

	systemPrompt, userPrompt, err := RenderSummarizationPrompt(data)
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

func TestSummarizationPromptDataStructure(t *testing.T) {
	// Test the data structure itself
	data := SummarizationPromptData{
		Query:   "test query",
		Content: "test content",
	}

	if data.Query != "test query" {
		t.Errorf("Expected query 'test query', got '%s'", data.Query)
	}

	if data.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", data.Content)
	}
}

func TestRenderSummarizationPromptConsistency(t *testing.T) {
	// Test that multiple calls with same data produce same output
	data := SummarizationPromptData{
		Query:   "test",
		Content: "test content",
	}

	sys1, user1, err1 := RenderSummarizationPrompt(data)
	if err1 != nil {
		t.Fatalf("First render failed: %v", err1)
	}

	sys2, user2, err2 := RenderSummarizationPrompt(data)
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

func BenchmarkRenderSummarizationPrompt(b *testing.B) {
	// Benchmark the summarization prompt rendering
	data := SummarizationPromptData{
		Query:   "What is artificial intelligence?",
		Content: "Artificial intelligence (AI) is intelligence demonstrated by machines, in contrast to the natural intelligence displayed by humans and animals. Leading AI textbooks define the field as the study of intelligent agents.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		systemPrompt, userPrompt, err := RenderSummarizationPrompt(data)
		if err != nil {
			b.Fatal(err)
		}
		// Use the results to prevent optimization
		_ = systemPrompt
		_ = userPrompt
	}
}
