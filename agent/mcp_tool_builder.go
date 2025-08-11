package agent

import (
	"context"
	"fmt"
	"slices"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// MCPTool builder to define MCP tool schema.
type MCPToolBuilder struct {
	tool MCPTool
}

func NewMCPToolBuilder(name, description string) *MCPToolBuilder {
	b := &MCPToolBuilder{
		tool: MCPTool{
			Tool: api.Tool{
				Type: "function",
				Function: api.ToolFunction{
					Name:        name,
					Description: description,
				},
			},
		},
	}

	// Initialize parameters object
	b.tool.Function.Parameters.Type = "object"
	b.tool.Function.Parameters.Properties = make(map[string]api.ToolProperty, 8)
	// Required slice stays nil until first add
	return b
}

func (b *MCPToolBuilder) StringParam(name, desc string, required bool) *MCPToolBuilder {
	prop := api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: desc,
	}

	b.setProp(name, prop, required)
	return b
}

func (b *MCPToolBuilder) StringSliceParam(name, desc string, required bool) *MCPToolBuilder {
	prop := api.ToolProperty{
		Type:        api.PropertyType{"array"},
		Items:       map[string]any{"type": "string"},
		Description: desc,
	}

	b.setProp(name, prop, required)
	return b
}

func (b *MCPToolBuilder) Summarize(enabled bool) *MCPToolBuilder {
	b.tool.SummarizeContext = enabled
	return b
}

func (b *MCPToolBuilder) WithHandler(fn func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk) *MCPToolBuilder {
	b.tool.Handler = fn
	return b
}

func (b *MCPToolBuilder) Build() MCPTool {
	return b.tool
}

func (b *MCPToolBuilder) setProp(name string, p api.ToolProperty, required bool) {
	props := b.tool.Function.Parameters.Properties
	props[name] = p
	if required {
		req := b.tool.Function.Parameters.Required
		if !slices.Contains(req, name) {
			b.tool.Function.Parameters.Required = append(req, name)
		}
	}
}

// ToolResultChunkBuilder is a builder for creating MCP Tool Response chunks.
type ToolResultChunkBuilder struct {
	chk *schema.ToolResultChunk
}

func NewToolResultChunk() *ToolResultChunkBuilder {
	return &ToolResultChunkBuilder{
		chk: &schema.ToolResultChunk{
			Metadata: make(map[string]string),
		},
	}
}

func (b *ToolResultChunkBuilder) Sentences(sentences ...string) *ToolResultChunkBuilder {
	b.chk.Sentences = append(b.chk.Sentences, sentences...)
	return b
}

func (b *ToolResultChunkBuilder) Attribution(attr string) *ToolResultChunkBuilder {
	b.chk.Attribution = attr
	return b
}

func (b *ToolResultChunkBuilder) Title(t string) *ToolResultChunkBuilder {
	b.chk.Title = t
	return b
}

func (b *ToolResultChunkBuilder) MetadataKV(key, value string) *ToolResultChunkBuilder {
	b.chk.Metadata[key] = value
	return b
}

func (b *ToolResultChunkBuilder) MetadataMap(m map[string]string) *ToolResultChunkBuilder {
	for k, v := range m {
		b.chk.Metadata[k] = v
	}
	return b
}

func (b *ToolResultChunkBuilder) ToolName(name string) *ToolResultChunkBuilder {
	b.chk.ToolName = name
	return b
}

func (b *ToolResultChunkBuilder) Error(errMsg string) *ToolResultChunkBuilder {
	b.chk.Error = errMsg
	return b
}

func (b *ToolResultChunkBuilder) Build() *schema.ToolResultChunk {
	return b.chk
}

func NewMathToolResult(expression, result string, steps []string) *schema.ToolResultChunk {
	b := NewToolResultChunk().
		Title("Mathematical Calculation").
		Sentences(fmt.Sprintf("%s = %s", expression, result)).
		MetadataKV("expression", expression).
		MetadataKV("result", result).
		MetadataKV("calculation_type", "arithmetic")

	if len(steps) > 0 {
		b.Sentences("Calculation steps:").Sentences(steps...)
	}
	return b.Build()
}
