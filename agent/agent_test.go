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

func TestPromptTemplate(t *testing.T) {
	template := PromptTemplate{
		Name:      "test-template",
		Template:  "Hello {{.Name}}",
		Variables: []string{"Name"},
		Metadata:  map[string]string{"version": "1.0"},
	}

	assert.Equal(t, "test-template", template.Name)
	assert.Equal(t, "Hello {{.Name}}", template.Template)
	assert.Equal(t, []string{"Name"}, template.Variables)
	assert.Equal(t, "1.0", template.Metadata["version"])
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

	prompt := PromptTemplate{
		Name:     "test-prompt",
		Template: "Test template",
	}

	config := AgentConfig{
		MiniModel: mockMiniModel,
		BigModel:  mockBigModel,
		Tools:     []MCPTool{tool},
		Prompt:    prompt,
		MaxTokens: 1000,
		MaxTurns:  3,
	}

	assert.Equal(t, mockMiniModel, config.MiniModel)
	assert.Equal(t, mockBigModel, config.BigModel)
	assert.Len(t, config.Tools, 1)
	assert.Equal(t, prompt, config.Prompt)
	assert.Equal(t, 1000, config.MaxTokens)
	assert.Equal(t, 3, config.MaxTurns)
}

func TestNewAgent(t *testing.T) {
	config := AgentConfig{
		MaxTokens: 2000,
		MaxTurns:  5,
	}

	agent := NewAgent(config)

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
	emptyConfig := AgentConfig{}
	agent := NewAgent(emptyConfig)
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
	agentWithNilModels := NewAgent(configWithNilModels)
	assert.NotNil(t, agentWithNilModels)
	assert.Nil(t, agentWithNilModels.config.MiniModel)
	assert.Nil(t, agentWithNilModels.config.BigModel)
}

func TestPromptTemplateWithComplexMetadata(t *testing.T) {
	metadata := map[string]string{
		"version":     "2.1",
		"author":      "test-author",
		"description": "Complex template for testing",
		"tags":        "test,agent,prompt",
	}

	template := PromptTemplate{
		Name:      "complex-template",
		Template:  "Complex template with {{.Variable1}} and {{.Variable2}}",
		Variables: []string{"Variable1", "Variable2"},
		Metadata:  metadata,
	}

	assert.Equal(t, "complex-template", template.Name)
	assert.Contains(t, template.Template, "{{.Variable1}}")
	assert.Contains(t, template.Template, "{{.Variable2}}")
	assert.Len(t, template.Variables, 2)
	assert.Equal(t, "Variable1", template.Variables[0])
	assert.Equal(t, "Variable2", template.Variables[1])
	assert.Len(t, template.Metadata, 4)
	assert.Equal(t, "2.1", template.Metadata["version"])
	assert.Equal(t, "test-author", template.Metadata["author"])
	assert.Equal(t, "Complex template for testing", template.Metadata["description"])
	assert.Equal(t, "test,agent,prompt", template.Metadata["tags"])
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
		agent := NewAgent(config)
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
