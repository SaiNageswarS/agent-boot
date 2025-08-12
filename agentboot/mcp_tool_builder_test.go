package agentboot

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

func TestNewMCPTool(t *testing.T) {
	builder := NewMCPToolBuilder("test-tool", "A test tool for testing")

	assert.NotNil(t, builder)
	assert.Equal(t, "function", builder.tool.Tool.Type)
	assert.Equal(t, "test-tool", builder.tool.Tool.Function.Name)
	assert.Equal(t, "A test tool for testing", builder.tool.Tool.Function.Description)
	assert.Equal(t, "object", builder.tool.Tool.Function.Parameters.Type)
	assert.NotNil(t, builder.tool.Tool.Function.Parameters.Properties)
	assert.Len(t, builder.tool.Tool.Function.Parameters.Properties, 0) // Should start empty
	assert.Nil(t, builder.tool.Tool.Function.Parameters.Required)      // Should start nil
}

func TestMCPToolBuilderStringParam(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")

	result := builder.StringParam("name", "The name parameter", true)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Contains(t, builder.tool.Tool.Function.Parameters.Properties, "name")

	prop := builder.tool.Tool.Function.Parameters.Properties["name"]
	assert.Equal(t, api.PropertyType{"string"}, prop.Type)
	assert.Equal(t, "The name parameter", prop.Description)
	assert.Contains(t, builder.tool.Tool.Function.Parameters.Required, "name")
}

func TestMCPToolBuilderStringParamOptional(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")

	builder.StringParam("optional", "Optional parameter", false)

	assert.Contains(t, builder.tool.Tool.Function.Parameters.Properties, "optional")
	prop := builder.tool.Tool.Function.Parameters.Properties["optional"]
	assert.Equal(t, api.PropertyType{"string"}, prop.Type)
	assert.Equal(t, "Optional parameter", prop.Description)
	assert.NotContains(t, builder.tool.Tool.Function.Parameters.Required, "optional")
}

func TestMCPToolBuilderStringSliceParam(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")

	result := builder.StringSliceParam("tags", "List of tags", true)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Contains(t, builder.tool.Tool.Function.Parameters.Properties, "tags")

	prop := builder.tool.Tool.Function.Parameters.Properties["tags"]
	assert.Equal(t, api.PropertyType{"array"}, prop.Type)
	assert.Equal(t, "List of tags", prop.Description)
	assert.Equal(t, map[string]any{"type": "string"}, prop.Items)
	assert.Contains(t, builder.tool.Tool.Function.Parameters.Required, "tags")
}

func TestMCPToolBuilderSummarize(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")

	result := builder.Summarize(true)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.True(t, builder.tool.SummarizeContext)

	// Test setting to false
	builder.Summarize(false)
	assert.False(t, builder.tool.SummarizeContext)
}

func TestMCPToolBuilderWithHandler(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")
	handlerCalled := false

	handler := func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
		handlerCalled = true
		ch := make(chan *schema.ToolResultChunk, 1)
		close(ch)
		return ch
	}

	result := builder.WithHandler(handler)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.NotNil(t, builder.tool.Handler)

	// Test handler execution
	ch := builder.tool.Handler(context.Background(), nil)
	assert.True(t, handlerCalled)
	assert.NotNil(t, ch)
}

func TestMCPToolBuilderBuild(t *testing.T) {
	handler := func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
		ch := make(chan *schema.ToolResultChunk, 1)
		close(ch)
		return ch
	}

	builder := NewMCPToolBuilder("calculator", "Performs mathematical calculations")
	tool := builder.
		StringParam("expression", "Mathematical expression to evaluate", true).
		StringParam("format", "Output format", false).
		StringSliceParam("operations", "List of operations", false).
		Summarize(true).
		WithHandler(handler).
		Build()

	assert.Equal(t, "function", tool.Tool.Type)
	assert.Equal(t, "calculator", tool.Tool.Function.Name)
	assert.Equal(t, "Performs mathematical calculations", tool.Tool.Function.Description)
	assert.True(t, tool.SummarizeContext)
	assert.NotNil(t, tool.Handler)

	// Check parameters
	assert.Len(t, tool.Tool.Function.Parameters.Properties, 3)
	assert.Contains(t, tool.Tool.Function.Parameters.Properties, "expression")
	assert.Contains(t, tool.Tool.Function.Parameters.Properties, "format")
	assert.Contains(t, tool.Tool.Function.Parameters.Properties, "operations")

	// Check required parameters
	assert.Len(t, tool.Tool.Function.Parameters.Required, 1)
	assert.Contains(t, tool.Tool.Function.Parameters.Required, "expression")
}

func TestMCPToolBuilderMultipleRequired(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")

	builder.
		StringParam("param1", "First required param", true).
		StringParam("param2", "Second required param", true).
		StringParam("param3", "Optional param", false)

	tool := builder.Build()

	assert.Len(t, tool.Tool.Function.Parameters.Required, 2)
	assert.Contains(t, tool.Tool.Function.Parameters.Required, "param1")
	assert.Contains(t, tool.Tool.Function.Parameters.Required, "param2")
	assert.NotContains(t, tool.Tool.Function.Parameters.Required, "param3")
}

func TestMCPToolBuilderDuplicateRequired(t *testing.T) {
	builder := NewMCPToolBuilder("test", "test")

	// Add the same parameter multiple times as required
	builder.
		StringParam("duplicate", "First time", true).
		StringParam("duplicate", "Second time", true) // Should not duplicate in required array

	tool := builder.Build()

	// Count occurrences of "duplicate" in required array
	count := 0
	for _, req := range tool.Tool.Function.Parameters.Required {
		if req == "duplicate" {
			count++
		}
	}
	assert.Equal(t, 1, count, "Required parameter should not be duplicated")
}

// Test ToolResultChunkBuilder

func TestNewToolResultChunk(t *testing.T) {
	builder := NewToolResultChunk()

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.chk)
	assert.NotNil(t, builder.chk.Metadata)
	assert.Empty(t, builder.chk.Sentences)
	assert.Empty(t, builder.chk.Attribution)
	assert.Empty(t, builder.chk.Title)
	assert.Empty(t, builder.chk.ToolName)
	assert.Empty(t, builder.chk.Error)
}

func TestToolResultChunkBuilderSentences(t *testing.T) {
	builder := NewToolResultChunk()

	result := builder.Sentences("First sentence", "Second sentence")

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Len(t, builder.chk.Sentences, 2)
	assert.Equal(t, "First sentence", builder.chk.Sentences[0])
	assert.Equal(t, "Second sentence", builder.chk.Sentences[1])

	// Test adding more sentences
	builder.Sentences("Third sentence")
	assert.Len(t, builder.chk.Sentences, 3)
	assert.Equal(t, "Third sentence", builder.chk.Sentences[2])
}

func TestToolResultChunkBuilderAttribution(t *testing.T) {
	builder := NewToolResultChunk()

	result := builder.Attribution("Source: Example.com")

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, "Source: Example.com", builder.chk.Attribution)
}

func TestToolResultChunkBuilderTitle(t *testing.T) {
	builder := NewToolResultChunk()

	result := builder.Title("Test Result")

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, "Test Result", builder.chk.Title)
}

func TestToolResultChunkBuilderMetadataKV(t *testing.T) {
	builder := NewToolResultChunk()

	result := builder.MetadataKV("key1", "value1")

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, "value1", builder.chk.Metadata["key1"])

	// Test adding more metadata
	builder.MetadataKV("key2", "value2")
	assert.Equal(t, "value2", builder.chk.Metadata["key2"])
	assert.Len(t, builder.chk.Metadata, 2)
}

func TestToolResultChunkBuilderMetadataMap(t *testing.T) {
	builder := NewToolResultChunk()
	metadata := map[string]string{
		"source":   "test",
		"version":  "1.0",
		"category": "math",
	}

	result := builder.MetadataMap(metadata)

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, "test", builder.chk.Metadata["source"])
	assert.Equal(t, "1.0", builder.chk.Metadata["version"])
	assert.Equal(t, "math", builder.chk.Metadata["category"])
	assert.Len(t, builder.chk.Metadata, 3)
}

func TestToolResultChunkBuilderToolName(t *testing.T) {
	builder := NewToolResultChunk()

	result := builder.ToolName("calculator")

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, "calculator", builder.chk.ToolName)
}

func TestToolResultChunkBuilderError(t *testing.T) {
	builder := NewToolResultChunk()

	result := builder.Error("Something went wrong")

	assert.Equal(t, builder, result) // Should return self for chaining
	assert.Equal(t, "Something went wrong", builder.chk.Error)
}

func TestToolResultChunkBuilderBuild(t *testing.T) {
	builder := NewToolResultChunk()
	chunk := builder.
		Sentences("Test sentence").
		Attribution("Test source").
		Title("Test Title").
		MetadataKV("key", "value").
		ToolName("test-tool").
		Error("").
		Build()

	assert.NotNil(t, chunk)
	assert.Len(t, chunk.Sentences, 1)
	assert.Equal(t, "Test sentence", chunk.Sentences[0])
	assert.Equal(t, "Test source", chunk.Attribution)
	assert.Equal(t, "Test Title", chunk.Title)
	assert.Equal(t, "value", chunk.Metadata["key"])
	assert.Equal(t, "test-tool", chunk.ToolName)
	assert.Empty(t, chunk.Error)
}

func TestNewMathToolResult(t *testing.T) {
	expression := "2 + 2"
	result := "4"
	steps := []string{"2 + 2", "= 4"}

	chunk := NewMathToolResult(expression, result, steps)

	assert.NotNil(t, chunk)
	assert.Equal(t, "Mathematical Calculation", chunk.Title)
	assert.Contains(t, chunk.Sentences[0], expression)
	assert.Contains(t, chunk.Sentences[0], result)
	assert.Equal(t, expression, chunk.Metadata["expression"])
	assert.Equal(t, result, chunk.Metadata["result"])
	assert.Equal(t, "arithmetic", chunk.Metadata["calculation_type"])
	assert.Contains(t, chunk.Sentences, "Calculation steps:")
	assert.Contains(t, chunk.Sentences, "2 + 2")
	assert.Contains(t, chunk.Sentences, "= 4")
}

func TestNewMathToolResultNoSteps(t *testing.T) {
	expression := "5 * 3"
	result := "15"

	chunk := NewMathToolResult(expression, result, nil)

	assert.NotNil(t, chunk)
	assert.Equal(t, "Mathematical Calculation", chunk.Title)
	assert.Contains(t, chunk.Sentences[0], expression)
	assert.Contains(t, chunk.Sentences[0], result)
	assert.Equal(t, expression, chunk.Metadata["expression"])
	assert.Equal(t, result, chunk.Metadata["result"])
	assert.Equal(t, "arithmetic", chunk.Metadata["calculation_type"])
	assert.NotContains(t, chunk.Sentences, "Calculation steps:")
}

func TestNewMathToolResultEmptySteps(t *testing.T) {
	expression := "10 / 2"
	result := "5"
	steps := []string{} // Empty slice

	chunk := NewMathToolResult(expression, result, steps)

	assert.NotNil(t, chunk)
	assert.Equal(t, "Mathematical Calculation", chunk.Title)
	assert.Contains(t, chunk.Sentences[0], expression)
	assert.Contains(t, chunk.Sentences[0], result)
	assert.NotContains(t, chunk.Sentences, "Calculation steps:")
}

// Benchmark tests
func BenchmarkNewMCPTool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewMCPToolBuilder("test", "test description")
		_ = builder
	}
}

func BenchmarkMCPToolBuilderBuild(b *testing.B) {
	handler := func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
		ch := make(chan *schema.ToolResultChunk, 1)
		close(ch)
		return ch
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewMCPToolBuilder("test", "test")
		tool := builder.
			StringParam("param1", "desc1", true).
			StringParam("param2", "desc2", false).
			StringSliceParam("param3", "desc3", true).
			Summarize(true).
			WithHandler(handler).
			Build()
		_ = tool
	}
}

func BenchmarkNewToolResultChunk(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewToolResultChunk()
		_ = builder
	}
}

func BenchmarkToolResultChunkBuilderBuild(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewToolResultChunk()
		chunk := builder.
			Sentences("Test sentence 1", "Test sentence 2").
			Attribution("Test source").
			Title("Test Title").
			MetadataKV("key1", "value1").
			MetadataKV("key2", "value2").
			ToolName("test-tool").
			Build()
		_ = chunk
	}
}

func BenchmarkNewMathToolResult(b *testing.B) {
	expression := "2 + 2"
	result := "4"
	steps := []string{"2 + 2", "= 4"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk := NewMathToolResult(expression, result, steps)
		_ = chunk
	}
}
