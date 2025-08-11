package agent

import (
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentBuilder(t *testing.T) {
	builder := NewAgentBuilder()

	assert.NotNil(t, builder)
	assert.Equal(t, 5, builder.config.MaxTurns)
	assert.Equal(t, 2000, builder.config.MaxTokens)
	assert.Nil(t, builder.config.MiniModel)
	assert.Nil(t, builder.config.BigModel)
	assert.Empty(t, builder.config.Tools)
}

func TestAgentBuilderWithMiniModel(t *testing.T) {
	mockModel := &mockLLMClient{model: "mini-model"}
	builder := NewAgentBuilder()

	result := builder.WithMiniModel(mockModel)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, mockModel, builder.config.MiniModel)
}

func TestAgentBuilderWithBigModel(t *testing.T) {
	mockModel := &mockLLMClient{model: "big-model"}
	builder := NewAgentBuilder()

	result := builder.WithBigModel(mockModel)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, mockModel, builder.config.BigModel)
}

func TestAgentBuilderAddTool(t *testing.T) {
	builder := NewAgentBuilder()
	tool1 := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{
				Name: "tool1",
			},
		},
	}
	tool2 := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{
				Name: "tool2",
			},
		},
	}

	result1 := builder.AddTool(tool1)
	result2 := builder.AddTool(tool2)

	assert.Equal(t, builder, result1) // Should return self for chaining
	assert.Equal(t, builder, result2) // Should return self for chaining
	assert.Len(t, builder.config.Tools, 2)
	assert.Equal(t, "tool1", builder.config.Tools[0].Function.Name)
	assert.Equal(t, "tool2", builder.config.Tools[1].Function.Name)
}

func TestAgentBuilderWithMaxTokens(t *testing.T) {
	builder := NewAgentBuilder()

	result := builder.WithMaxTokens(1500)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, 1500, builder.config.MaxTokens)
}

func TestAgentBuilderWithMaxTurns(t *testing.T) {
	builder := NewAgentBuilder()

	result := builder.WithMaxTurns(10)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, 10, builder.config.MaxTurns)
}

func TestAgentBuilderBuild(t *testing.T) {
	mockMiniModel := &mockLLMClient{model: "mini"}
	mockBigModel := &mockLLMClient{model: "big"}
	tool := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{
				Name: "test-tool",
			},
		},
	}

	builder := NewAgentBuilder()
	agent := builder.
		WithMiniModel(mockMiniModel).
		WithBigModel(mockBigModel).
		AddTool(tool).
		WithMaxTokens(3000).
		WithMaxTurns(7).
		Build()

	assert.NotNil(t, agent)
	assert.Equal(t, mockMiniModel, agent.config.MiniModel)
	assert.Equal(t, mockBigModel, agent.config.BigModel)
	assert.Len(t, agent.config.Tools, 1)
	assert.Equal(t, "test-tool", agent.config.Tools[0].Function.Name)
	assert.Equal(t, 3000, agent.config.MaxTokens)
	assert.Equal(t, 7, agent.config.MaxTurns)
}

func TestAgentBuilderFluentInterface(t *testing.T) {
	// Test that all methods return the builder for method chaining
	builder := NewAgentBuilder()
	mockModel := &mockLLMClient{model: "test"}

	result := builder.
		WithMiniModel(mockModel).
		WithBigModel(mockModel).
		WithMaxTokens(1000).
		WithMaxTurns(3)

	assert.Equal(t, builder, result)
	assert.Equal(t, mockModel, builder.config.MiniModel)
	assert.Equal(t, mockModel, builder.config.BigModel)
	assert.Equal(t, 1000, builder.config.MaxTokens)
	assert.Equal(t, 3, builder.config.MaxTurns)
}

func TestAgentBuilderMultipleTools(t *testing.T) {
	builder := NewAgentBuilder()

	tools := []MCPTool{
		{
			Tool: api.Tool{
				Function: api.ToolFunction{Name: "calculator"},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{Name: "weather"},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{Name: "search"},
			},
		},
	}

	for _, tool := range tools {
		builder.AddTool(tool)
	}

	assert.Len(t, builder.config.Tools, 3)
	assert.Equal(t, "calculator", builder.config.Tools[0].Function.Name)
	assert.Equal(t, "weather", builder.config.Tools[1].Function.Name)
	assert.Equal(t, "search", builder.config.Tools[2].Function.Name)
}

func TestAgentBuilderDefaultValues(t *testing.T) {
	builder := NewAgentBuilder()
	agent := builder.Build()

	// Test that default values are preserved
	assert.Equal(t, 5, agent.config.MaxTurns)
	assert.Equal(t, 2000, agent.config.MaxTokens)
	assert.Nil(t, agent.config.MiniModel)
	assert.Nil(t, agent.config.BigModel)
	assert.Empty(t, agent.config.Tools)
}

func TestAgentBuilderOverrideValues(t *testing.T) {
	builder := NewAgentBuilder()

	// Set initial values
	builder.WithMaxTokens(1000).WithMaxTurns(3)
	assert.Equal(t, 1000, builder.config.MaxTokens)
	assert.Equal(t, 3, builder.config.MaxTurns)

	// Override values
	builder.WithMaxTokens(2000).WithMaxTurns(8)
	assert.Equal(t, 2000, builder.config.MaxTokens)
	assert.Equal(t, 8, builder.config.MaxTurns)
}

func TestAgentBuilderNilValues(t *testing.T) {
	builder := NewAgentBuilder()

	// Test with nil models
	builder.WithMiniModel(nil).WithBigModel(nil)
	agent := builder.Build()

	assert.Nil(t, agent.config.MiniModel)
	assert.Nil(t, agent.config.BigModel)
}

func TestAgentBuilderZeroValues(t *testing.T) {
	builder := NewAgentBuilder()

	// Test with zero values
	builder.WithMaxTokens(0).WithMaxTurns(0)
	agent := builder.Build()

	assert.Equal(t, 0, agent.config.MaxTokens)
	assert.Equal(t, 0, agent.config.MaxTurns)
}

// Benchmark tests
func BenchmarkNewAgentBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewAgentBuilder()
		_ = builder
	}
}

func BenchmarkAgentBuilderBuild(b *testing.B) {
	mockModel := &mockLLMClient{model: "test"}
	tool := MCPTool{
		Tool: api.Tool{
			Function: api.ToolFunction{Name: "test"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewAgentBuilder()
		agent := builder.
			WithMiniModel(mockModel).
			WithBigModel(mockModel).
			AddTool(tool).
			WithMaxTokens(1000).
			WithMaxTurns(5).
			Build()
		_ = agent
	}
}

func BenchmarkAgentBuilderChaining(b *testing.B) {
	mockModel := &mockLLMClient{model: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewAgentBuilder()
		result := builder.
			WithMiniModel(mockModel).
			WithBigModel(mockModel).
			WithMaxTokens(1000).
			WithMaxTurns(5)
		_ = result
	}
}
