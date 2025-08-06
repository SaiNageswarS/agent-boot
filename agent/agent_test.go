package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
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
		Name:        "test-tool",
		Description: "A test tool",
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

	if tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tools[0].Name)
	}
}

func TestGenerateAnswer(t *testing.T) {
	mockResponse := "This is a test answer"

	agent := NewAgentBuilder().
		WithMiniModel(&mockLLMClient{responses: []string{mockResponse}, model: "mini-model"}).
		WithBigModel(&mockLLMClient{responses: []string{mockResponse}, model: "big-model"}).
		WithMaxTokens(1000).
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
		Name:        "calculator",
		Description: "Performs calculations",
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
