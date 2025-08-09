package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// mock llm client for testing different LLM calls
type mockLLMClient struct {
	responses []string
	callCount int
	err       error
	model     string
}

func (m *mockLLMClient) Capabilities() llm.Capability {
	return 0
}

func (m *mockLLMClient) GetModel() string {
	return m.model
}

func (m *mockLLMClient) GenerateInference(
	ctx context.Context,
	messages []llm.Message,
	callback func(string) error,
	options ...llm.LLMOption,
) error {
	if m.err != nil {
		return m.err
	}

	if m.callCount < len(m.responses) {
		response := m.responses[m.callCount]
		m.callCount++
		return callback(response)
	}

	// Default response if we run out of responses
	return callback("Default response")
}

func (m *mockLLMClient) GenerateInferenceWithTools(
	ctx context.Context,
	messages []llm.Message,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	opts ...llm.LLMOption,
) error {
	// For testing, just use the regular inference method
	return m.GenerateInference(ctx, messages, contentCallback, opts...)
}

func TestNewAgent(t *testing.T) {
	config := AgentConfig{
		Tools:     []MCPTool{},
		MaxTokens: 1000,
	}

	agent := NewAgent(config)

	if agent == nil {
		t.Fatal("NewAgent should return a non-nil agent")
	}

	if agent.config.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens to be 1000, got %d", agent.config.MaxTokens)
	}
}

func TestAddTool(t *testing.T) {
	tool := MCPTool{
		Tool: api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "test-tool",
				Description: "A test tool",
				Parameters: struct {
					Type       string                      `json:"type"`
					Defs       any                         `json:"$defs,omitempty"`
					Items      any                         `json:"items,omitempty"`
					Required   []string                    `json:"required"`
					Properties map[string]api.ToolProperty `json:"properties"`
				}{
					Type:       "object",
					Required:   []string{},
					Properties: map[string]api.ToolProperty{},
				},
			},
		},
		Handler: func(ctx context.Context, params map[string]string) <-chan *schema.ToolExecutionResultChunk {
			result := make(chan *schema.ToolExecutionResultChunk, 1)
			defer close(result)

			result <- NewToolResult("Test", []string{"result"})
			return result
		},
	}

	agent := NewAgentBuilder().
		AddTool(tool).
		Build()

	tools := agent.GetAvailableTools()
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Function.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tools[0].Function.Name)
	}
}

func TestGenerateAnswer(t *testing.T) {
	mockResponse := "This is a test answer"

	agent := NewAgentBuilder().
		WithMiniModel(&mockLLMClient{responses: []string{mockResponse}, model: "mini-model"}).
		WithBigModel(&mockLLMClient{responses: []string{mockResponse}, model: "big-model"}).
		WithMaxTokens(1000).
		WithMaxTurns(0). // Disable turn-based execution for this test
		Build()

	req := &schema.GenerateAnswerRequest{
		Question: "Test question",
		Context:  "Test context",
	}

	response, err := agent.Execute(context.Background(), &NoOpProgressReporter{}, req)
	if err != nil {
		t.Fatalf("GenerateAnswer failed: %v", err)
	}

	if response.Answer != mockResponse {
		t.Errorf("Expected answer '%s', got '%s'", mockResponse, response.Answer)
	}

	if response.ModelUsed != "mini-model" {
		t.Errorf("Expected model 'mini-model', got '%s'", response.ModelUsed)
	}

	if response.ProcessingTime < 0 {
		t.Error("Processing time should not be negative")
	}
}

func TestGenerateAnswerWithTools(t *testing.T) {
	toolResponse := "TOOL_SELECTION_START\nTOOL: calculator\nCONFIDENCE: 0.9\nREASONING: Math needed\nPARAMETERS:\nexpression: 2+2\nTOOL_SELECTION_END"
	answerResponse := "The answer is 4"

	// Use the same client but modify the response after the first call
	mockClient := &mockLLMClient{
		responses: []string{toolResponse, answerResponse},
	}

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
			result := make(chan *schema.ToolExecutionResultChunk, 1)
			defer close(result)

			result <- NewMathToolResult("2+2", "4", []string{"Step 1: 2 + 2 = 4"})
			return result
		},
	}

	agent := NewAgentBuilder().
		WithMiniModel(mockClient).
		WithBigModel(mockClient).
		AddTool(calcTool).
		WithMaxTurns(0). // Disable turn-based execution for this test
		Build()

	req := &schema.GenerateAnswerRequest{
		Question: "What is 2+2?",
	}

	response, err := agent.Execute(context.Background(), &NoOpProgressReporter{}, req)
	if err != nil {
		t.Fatalf("GenerateAnswer with tools failed: %v", err)
	}

	if len(response.ToolsUsed) != 1 {
		t.Errorf("Expected 1 tool used, got %d", len(response.ToolsUsed))
	}

	if response.ToolsUsed[0] != "calculator" {
		t.Errorf("Expected calculator tool, got '%s'", response.ToolsUsed[0])
	}

	if response.Answer != answerResponse {
		t.Errorf("Expected answer '%s', got '%s'", answerResponse, response.Answer)
	}

	if response.Answer != answerResponse {
		t.Errorf("Expected answer '%s', got '%s'", answerResponse, response.Answer)
	}
}

// TestTurnBasedExecution tests the new turn-based execution mode
func TestTurnBasedExecution(t *testing.T) {
	finalAnswer := "The final answer is 4"

	// Mock client that will provide final answer after tools are used
	mockClient := &mockLLMClient{
		responses: []string{finalAnswer},
		model:     "test-model",
	}

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
			result := make(chan *schema.ToolExecutionResultChunk, 1)
			defer close(result)

			result <- NewMathToolResult("2+2", "4", []string{"Step 1: 2 + 2 = 4"})
			return result
		},
	}

	agent := NewAgentBuilder().
		WithMiniModel(mockClient).
		WithBigModel(mockClient).
		AddTool(calcTool).
		WithMaxTurns(2). // Enable turn-based execution
		Build()

	req := &schema.GenerateAnswerRequest{
		Question: "What is 2+2?",
	}

	response, err := agent.Execute(context.Background(), &NoOpProgressReporter{}, req)
	if err != nil {
		t.Fatalf("ExecuteTurnBased failed: %v", err)
	}

	// Should have used turn-based execution
	if response.Metadata["native_tool_calling"] != "false" {
		t.Errorf("Expected native_tool_calling to be false, got %s", response.Metadata["native_tool_calling"])
	}

	// Should have final answer
	if response.Answer == "" {
		t.Error("Expected non-empty answer")
	}

	// Should have processing metadata
	if response.ProcessingTime < 0 {
		t.Error("Expected non-negative processing time")
	}

	// Should have model information
	if response.ModelUsed == "" {
		t.Error("Expected model information")
	}
}

// TestStreamingExecution tests that answer generation streams properly
func TestStreamingExecution(t *testing.T) {
	finalAnswer := "This is a streaming answer"

	// Mock client that will provide multiple responses for the turn-based execution
	mockClient := &mockLLMClient{
		responses: []string{
			"Tool selection failed", // First call (tool selection)
			finalAnswer,             // Second call (final answer generation)
			finalAnswer,             // Extra responses in case more calls are needed
		},
		model: "test-model",
	}

	// Mock reporter that captures streaming events
	streamingReporter := &StreamingTestReporter{
		chunks: make([]*schema.AnswerChunk, 0),
	}

	// Create agent without tools to force direct answer generation
	agent := NewAgentBuilder().
		WithMiniModel(mockClient).
		WithBigModel(mockClient).
		WithMaxTurns(1).
		Build()

	req := &schema.GenerateAnswerRequest{
		Question: "Tell me something",
	}

	response, err := agent.Execute(context.Background(), streamingReporter, req)
	if err != nil {
		t.Fatalf("ExecuteTurnBased failed: %v", err)
	}

	// Should have received streaming chunks
	if len(streamingReporter.chunks) == 0 {
		t.Error("Expected to receive streaming answer chunks")
	}

	// Final response should have the complete answer (or the fallback final answer)
	if response.Answer == "" {
		t.Error("Expected non-empty final answer")
	}

	// Should have at least one final chunk (the complete answer)
	hasFinal := false
	for _, chunk := range streamingReporter.chunks {
		if chunk.IsFinal {
			hasFinal = true
		}
	}

	if !hasFinal {
		t.Error("Expected to receive final answer chunk")
	}

	t.Logf("Received %d streaming chunks", len(streamingReporter.chunks))
	t.Logf("Final answer: %s", response.Answer)
}

// StreamingTestReporter captures streaming events for testing
type StreamingTestReporter struct {
	chunks []*schema.AnswerChunk
}

func (r *StreamingTestReporter) Send(event *schema.AgentStreamChunk) error {
	switch chunk := event.ChunkType.(type) {
	case *schema.AgentStreamChunk_Answer:
		r.chunks = append(r.chunks, chunk.Answer)
	}
	return nil
}

func TestShouldUseBigModel(t *testing.T) {
	agent := NewAgent(AgentConfig{})

	// Short simple query should use mini model
	if agent.shouldUseBigModel("Hi", []string{}) {
		t.Error("Short query should use mini model")
	}

	// Long query should use big model
	longQuery := strings.Repeat("This is a very long query that should trigger the big model because it exceeds the character limit. ", 5)
	if !agent.shouldUseBigModel(longQuery, []string{}) {
		t.Error("Long query should use big model")
	}

	// Multiple tool results should use big model
	if !agent.shouldUseBigModel("Simple query", []string{"result1", "result2"}) {
		t.Error("Multiple tool results should use big model")
	}

	// Complex keywords should use big model
	if !agent.shouldUseBigModel("Please analyze this data", []string{}) {
		t.Error("Complex keywords should use big model")
	}
}

func TestGetMaxTokens(t *testing.T) {
	// Test with configured max tokens
	agent := NewAgent(AgentConfig{MaxTokens: 500})
	if agent.getMaxTokens() != 500 {
		t.Errorf("Expected 500 max tokens, got %d", agent.getMaxTokens())
	}

	// Test with default max tokens
	agent = NewAgent(AgentConfig{})
	if agent.getMaxTokens() != 2000 {
		t.Errorf("Expected default 2000 max tokens, got %d", agent.getMaxTokens())
	}
}

func TestGetMaxTools(t *testing.T) {
	if getMaxTools(5) != 5 {
		t.Errorf("Expected 5 max tools, got %d", getMaxTools(5))
	}

	if getMaxTools(0) != 3 {
		t.Errorf("Expected default 3 max tools, got %d", getMaxTools(0))
	}

	if getMaxTools(15) != 3 {
		t.Errorf("Expected capped 3 max tools, got %d", getMaxTools(15))
	}
}

func TestGetCurrentTimeMs(t *testing.T) {
	start := time.Now().UnixMilli()
	result := getCurrentTimeMs()
	end := time.Now().UnixMilli()

	if result < start || result > end {
		t.Errorf("getCurrentTimeMs should return current time, got %d (expected between %d and %d)", result, start, end)
	}
}
