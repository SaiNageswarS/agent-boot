package agent

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

// mockLLMClient implements the LLMClient interface for testing
type mockLLMClient struct {
	model string
}

func (m *mockLLMClient) GenerateInference(
	ctx context.Context,
	messages []llm.Message,
	callback func(chunk string) error,
	opts ...llm.LLMOption,
) error {
	return callback("test response")
}

func (m *mockLLMClient) GenerateInferenceWithTools(
	ctx context.Context,
	messages []llm.Message,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	opts ...llm.LLMOption,
) error {
	return contentCallback("test response with tools")
}

func (m *mockLLMClient) Capabilities() llm.Capability {
	return llm.NativeToolCalling
}

func (m *mockLLMClient) GetModel() string {
	return m.model
}

func TestAgentConfig(t *testing.T) {
	mockMiniModel := &mockLLMClient{model: "mini-model"}
	mockBigModel := &mockLLMClient{model: "big-model"}

	tool := MCPTool{
		SummarizeContext: true,
		Handler: func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			close(ch)
			return ch
		},
	}

	config := AgentConfig{
		MiniModel: mockMiniModel,
		BigModel:  mockBigModel,
		Tools:     []MCPTool{tool},
		MaxTokens: 1000,
		MaxTurns:  3,
	}

	assert.Equal(t, mockMiniModel, config.MiniModel)
	assert.Equal(t, mockBigModel, config.BigModel)
	assert.Len(t, config.Tools, 1)
	assert.Equal(t, 1000, config.MaxTokens)
	assert.Equal(t, 3, config.MaxTurns)
}

func TestNewAgent(t *testing.T) {
	config := AgentConfig{
		MaxTokens: 2000,
		MaxTurns:  5,
	}

	agent := NewAgentBuilder().
		WithMaxTokens(config.MaxTokens).
		WithMaxTurns(config.MaxTurns).
		Build()

	assert.NotNil(t, agent)
	assert.Equal(t, config, agent.config)
	assert.Equal(t, 2000, agent.config.MaxTokens)
	assert.Equal(t, 5, agent.config.MaxTurns)
}

func TestMCPTool(t *testing.T) {
	// Test MCPTool structure
	handlerCalled := false
	handler := func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
		handlerCalled = true
		ch := make(chan *schema.ToolResultChunk, 1)
		close(ch)
		return ch
	}

	tool := MCPTool{
		SummarizeContext: true,
		Handler:          handler,
	}

	assert.True(t, tool.SummarizeContext)
	assert.NotNil(t, tool.Handler)

	// Test handler execution
	ch := tool.Handler(nil, nil)
	assert.True(t, handlerCalled)
	assert.NotNil(t, ch)

	// Verify channel is closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")
}

func TestMCPToolWithAPI(t *testing.T) {
	tool := MCPTool{
		Tool: api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "test-tool",
				Description: "A test tool",
			},
		},
		SummarizeContext: false,
		Handler: func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			close(ch)
			return ch
		},
	}

	assert.Equal(t, "function", tool.Tool.Type)
	assert.Equal(t, "test-tool", tool.Tool.Function.Name)
	assert.Equal(t, "A test tool", tool.Tool.Function.Description)
	assert.False(t, tool.SummarizeContext)
}

func TestAgentConfigValidation(t *testing.T) {
	// Test empty config
	agent := Agent{}
	assert.NotNil(t, agent)
	assert.Equal(t, 0, agent.config.MaxTokens)
	assert.Equal(t, 0, agent.config.MaxTurns)

	// Test config with nil models
	configWithNilModels := AgentConfig{
		MiniModel: nil,
		BigModel:  nil,
		MaxTokens: 1000,
		MaxTurns:  3,
	}
	agentWithNilModels := Agent{
		config: configWithNilModels,
	}
	assert.NotNil(t, agentWithNilModels)
	assert.Nil(t, agentWithNilModels.config.MiniModel)
	assert.Nil(t, agentWithNilModels.config.BigModel)
}

func TestMCPToolDefaults(t *testing.T) {
	// Test MCPTool with default values
	tool := MCPTool{}

	assert.False(t, tool.SummarizeContext) // Should default to false
	assert.Nil(t, tool.Handler)            // Should default to nil
	assert.Equal(t, "", tool.Tool.Type)    // Should be empty by default
}

// Benchmark tests
func BenchmarkNewAgent(b *testing.B) {
	config := AgentConfig{
		MaxTokens: 2000,
		MaxTurns:  5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent := Agent{
			config: config,
		}
		_ = agent // Use the result to prevent optimization
	}
}

func BenchmarkMCPToolHandler(b *testing.B) {
	tool := MCPTool{
		Handler: func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			close(ch)
			return ch
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := tool.Handler(nil, nil)
		_ = ch
	}
}
