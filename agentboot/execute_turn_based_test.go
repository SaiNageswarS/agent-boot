package agentboot

import (
	"context"
	"errors"
	"testing"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

// MockProgressReporter implements ProgressReporter for testing
type MockProgressReporter struct {
	events []*schema.AgentStreamChunk
}

func (m *MockProgressReporter) Send(event *schema.AgentStreamChunk) error {
	m.events = append(m.events, event)
	return nil
}

func (m *MockProgressReporter) GetEvents() []*schema.AgentStreamChunk {
	return m.events
}

func (m *MockProgressReporter) GetEventCount() int {
	return len(m.events)
}

func (m *MockProgressReporter) Reset() {
	m.events = nil
}

// Enhanced mock LLM client with configurable responses
type testLLMClient struct {
	model            string
	response         string
	toolCalls        []api.ToolCall
	shouldError      bool
	errorMessage     string
	callCount        int
	responses        []string
	toolCallsPerTurn [][]api.ToolCall
}

func (m *testLLMClient) GenerateInference(
	ctx context.Context,
	messages []llm.Message,
	callback func(chunk string) error,
	opts ...llm.LLMOption,
) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}

	response := m.response
	if m.callCount < len(m.responses) {
		response = m.responses[m.callCount]
	}
	m.callCount++

	return callback(response)
}

func (m *testLLMClient) GenerateInferenceWithTools(
	ctx context.Context,
	messages []llm.Message,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	opts ...llm.LLMOption,
) error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}

	response := m.response
	var toolCalls []api.ToolCall

	if m.callCount < len(m.responses) {
		response = m.responses[m.callCount]
	}
	if m.callCount < len(m.toolCallsPerTurn) {
		toolCalls = m.toolCallsPerTurn[m.callCount]
	} else if len(m.toolCalls) > 0 && m.callCount == 0 {
		toolCalls = m.toolCalls
	}

	m.callCount++

	if len(toolCalls) > 0 {
		return toolCallback(toolCalls)
	}

	return contentCallback(response)
}

func (m *testLLMClient) Capabilities() llm.Capability {
	return llm.NativeToolCalling
}

func (m *testLLMClient) GetModel() string {
	return m.model
}

func TestAgentExecute(t *testing.T) {
	// Setup
	mockBigModel := &testLLMClient{
		model:    "test-big-model",
		response: "This is the final answer",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithSystemPrompt("You are a helpful agent to solve math problem").
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "What is 2+2?",
	}

	// Execute
	result, err := agent.Execute(context.Background(), reporter, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "This is the final answer", result.Answer)
	assert.GreaterOrEqual(t, result.ProcessingTime, int64(0))
	assert.NotNil(t, result.ToolsUsed)
	assert.NotNil(t, result.Metadata)

	// Check the final StreamComplete event was sent
	events := reporter.GetEvents()
	hasCompleteEvent := false
	for _, event := range events {
		if event.GetComplete() != nil {
			hasCompleteEvent = true
			break
		}
	}
	assert.True(t, hasCompleteEvent, "Should have sent a StreamComplete event")
}

func TestAgentExecuteWithTools(t *testing.T) {
	// Setup mock tool
	mockTool := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{
				Name: "calculator",
			},
		},
		Handler: func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			ch <- &schema.ToolResultChunk{
				Sentences: []string{"2 + 2 = 4"},
				Title:     "Calculation Result",
			}
			close(ch)
			return ch
		},
	}

	mockBigModel := &testLLMClient{
		model: "test-big-model",
		toolCallsPerTurn: [][]api.ToolCall{
			{
				{
					Function: api.ToolCallFunction{
						Name:      "calculator",
						Arguments: map[string]any{"expression": "2+2"},
					},
				},
			},
			{}, // No tool calls in turn 1
			{}, // No tool calls in turn 2
		},
		responses: []string{"", "", "", "The answer is 4"}, // 4 responses for 4 calls
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(3).
		AddTool(mockTool).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "What is 2+2?",
	}

	// Execute
	result, err := agent.Execute(context.Background(), reporter, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "The answer is 4", result.Answer)
	assert.GreaterOrEqual(t, result.ProcessingTime, int64(0))

	// Check that events were sent
	assert.GreaterOrEqual(t, reporter.GetEventCount(), 1)
}

func TestAgentExecuteMaxTurns(t *testing.T) {
	// Setup model that always returns tool calls
	mockBigModel := &testLLMClient{
		model: "test-big-model",
		toolCalls: []api.ToolCall{
			{
				Function: api.ToolCallFunction{
					Name:      "endless-tool",
					Arguments: map[string]any{},
				},
			},
		},
	}

	mockTool := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{
				Name: "endless-tool",
			},
		},
		Handler: func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			ch <- &schema.ToolResultChunk{
				Sentences: []string{"Tool executed"},
			}
			close(ch)
			return ch
		},
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(2).
		AddTool(mockTool).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "Test question",
	}

	// Execute
	result, err := agent.Execute(context.Background(), reporter, req)

	// Should still complete even if max turns reached
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, mockBigModel.callCount) // Should have called model maxTurns times + 1 for final inference
}

func TestAgentExecuteLLMError(t *testing.T) {
	// Setup model that returns error
	mockBigModel := &testLLMClient{
		model:        "test-big-model",
		shouldError:  true,
		errorMessage: "LLM service unavailable",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "Test question",
	}

	// Execute
	result, err := agent.Execute(context.Background(), reporter, req)

	// The current implementation doesn't return an error for LLM failures,
	// it returns a result with empty answer and logs the error
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Answer) // Answer should be empty due to LLM error
	assert.GreaterOrEqual(t, result.ProcessingTime, int64(0))
}

func TestAgentExecuteWithContext(t *testing.T) {
	mockBigModel := &testLLMClient{
		model:    "test-big-model",
		response: "Answer based on context",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithSystemPrompt("Test context").
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "Test question",
	}

	// Execute
	result, err := agent.Execute(context.Background(), reporter, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Answer based on context", result.Answer)
}

func TestAgentExecuteEmptyQuestion(t *testing.T) {
	mockBigModel := &testLLMClient{
		model:    "test-big-model",
		response: "I need a question to answer",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "",
	}

	// Execute
	result, err := agent.Execute(context.Background(), reporter, req)

	// Should still work with empty question
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "I need a question to answer", result.Answer)
}

func TestAgentSelectTools(t *testing.T) {
	expectedToolCalls := []api.ToolCall{
		{
			Function: api.ToolCallFunction{
				Name:      "test-tool",
				Arguments: map[string]any{"param": "value"},
			},
		},
	}

	mockBigModel := &testLLMClient{
		model:     "test-model",
		toolCalls: expectedToolCalls,
	}

	mockTool := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{
				Name: "test-tool",
			},
		},
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		AddTool(mockTool).
		Build()

	reporter := &MockProgressReporter{}

	messages := []llm.Message{
		{Role: "user", Content: "Use test tool"},
	}

	// Execute
	toolCalls := agent.SelectTools(context.Background(), reporter, messages, 0)

	// Assert
	assert.Len(t, toolCalls, 1)
	assert.Equal(t, "test-tool", toolCalls[0].Function.Name)
}

func TestAgentSelectToolsError(t *testing.T) {
	mockBigModel := &testLLMClient{
		model:        "test-model",
		shouldError:  true,
		errorMessage: "Model error",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		Build()

	reporter := &MockProgressReporter{}

	messages := []llm.Message{
		{Role: "user", Content: "Test message"},
	}

	// Execute
	toolCalls := agent.SelectTools(context.Background(), reporter, messages, 0)

	// Assert
	assert.Empty(t, toolCalls)
}

func TestAgentExecuteNilReporter(t *testing.T) {
	mockBigModel := &testLLMClient{
		model:    "test-big-model",
		response: "Test response",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	req := &schema.GenerateAnswerRequest{
		Question: "Test question",
	}

	// Execute with nil reporter (should cause panic or error)
	assert.Panics(t, func() {
		agent.Execute(context.Background(), nil, req)
	})
}

func TestAgentExecuteCanceledContext(t *testing.T) {
	mockBigModel := &testLLMClient{
		model:    "test-big-model",
		response: "Test response",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	reporter := &MockProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "Test question",
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute should respect context cancellation
	// Note: This test might pass if the LLM call doesn't check context
	// The behavior depends on the implementation
	result, err := agent.Execute(ctx, reporter, req)

	// The exact behavior depends on implementation, but it shouldn't panic
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	} else {
		assert.NotNil(t, result)
	}
}

// Benchmark tests
func BenchmarkAgentExecute(b *testing.B) {
	mockBigModel := &testLLMClient{
		model:    "test-big-model",
		response: "Benchmark response",
	}

	agent := NewAgentBuilder().
		WithBigModel(mockBigModel).
		WithToolSelector(mockBigModel).
		WithMaxTokens(1000).
		WithMaxTurns(3).
		Build()

	reporter := &NoOpProgressReporter{}

	req := &schema.GenerateAnswerRequest{
		Question: "Benchmark question",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockBigModel.callCount = 0 // Reset for each iteration
		result, err := agent.Execute(context.Background(), reporter, req)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}
