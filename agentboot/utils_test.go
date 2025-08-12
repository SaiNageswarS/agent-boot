package agentboot

import (
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentTimeMs(t *testing.T) {
	// Test that getCurrentTimeMs returns a reasonable timestamp
	before := time.Now().UnixMilli()
	result := getCurrentTimeMs()
	after := time.Now().UnixMilli()

	// The result should be between before and after
	assert.GreaterOrEqual(t, result, before)
	assert.LessOrEqual(t, result, after)
}

func TestGetCurrentTimeMsMultipleCalls(t *testing.T) {
	// Test that multiple calls return increasing values
	time1 := getCurrentTimeMs()
	time.Sleep(1 * time.Millisecond) // Small sleep to ensure time difference
	time2 := getCurrentTimeMs()

	assert.Greater(t, time2, time1)
}

func TestFindMCPToolByName(t *testing.T) {
	tools := []MCPTool{
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "calculator",
				},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "weather",
				},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "search",
				},
			},
		},
	}

	// Test finding existing tool
	result := findMCPToolByName(tools, "weather")
	assert.NotNil(t, result)
	assert.Equal(t, "weather", result.Function.Name)

	// Test finding first tool
	result = findMCPToolByName(tools, "calculator")
	assert.NotNil(t, result)
	assert.Equal(t, "calculator", result.Function.Name)

	// Test finding last tool
	result = findMCPToolByName(tools, "search")
	assert.NotNil(t, result)
	assert.Equal(t, "search", result.Function.Name)
}

func TestFindMCPToolByNameNotFound(t *testing.T) {
	tools := []MCPTool{
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "calculator",
				},
			},
		},
	}

	// Test finding non-existent tool
	result := findMCPToolByName(tools, "non-existent")
	assert.Nil(t, result)
}

func TestFindMCPToolByNameEmptySlice(t *testing.T) {
	tools := []MCPTool{}

	// Test with empty slice
	result := findMCPToolByName(tools, "anything")
	assert.Nil(t, result)
}

func TestFindMCPToolByNameNilSlice(t *testing.T) {
	var tools []MCPTool = nil

	// Test with nil slice
	result := findMCPToolByName(tools, "anything")
	assert.Nil(t, result)
}

func TestFindMCPToolByNameEmptyName(t *testing.T) {
	tools := []MCPTool{
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "",
				},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "calculator",
				},
			},
		},
	}

	// Test finding tool with empty name
	result := findMCPToolByName(tools, "")
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Function.Name)

	// Test finding normal tool
	result = findMCPToolByName(tools, "calculator")
	assert.NotNil(t, result)
	assert.Equal(t, "calculator", result.Function.Name)
}

func TestFindMCPToolByNameCaseSensitive(t *testing.T) {
	tools := []MCPTool{
		{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "Calculator",
				},
			},
		},
	}

	// Test case sensitivity
	result := findMCPToolByName(tools, "calculator")
	assert.Nil(t, result) // Should not find with different case

	result = findMCPToolByName(tools, "Calculator")
	assert.NotNil(t, result) // Should find with exact case
	assert.Equal(t, "Calculator", result.Function.Name)
}

func TestToAPITools(t *testing.T) {
	mcpTools := []MCPTool{
		{
			Tool: api.Tool{
				Type: "function",
				Function: api.ToolFunction{
					Name:        "calculator",
					Description: "Performs calculations",
				},
			},
			SummarizeContext: true,
		},
		{
			Tool: api.Tool{
				Type: "function",
				Function: api.ToolFunction{
					Name:        "weather",
					Description: "Gets weather info",
				},
			},
			SummarizeContext: false,
		},
	}

	apiTools := toAPITools(mcpTools)

	assert.Len(t, apiTools, 2)

	// Test first tool
	assert.Equal(t, "function", apiTools[0].Type)
	assert.Equal(t, "calculator", apiTools[0].Function.Name)
	assert.Equal(t, "Performs calculations", apiTools[0].Function.Description)

	// Test second tool
	assert.Equal(t, "function", apiTools[1].Type)
	assert.Equal(t, "weather", apiTools[1].Function.Name)
	assert.Equal(t, "Gets weather info", apiTools[1].Function.Description)
}

func TestToAPIToolsEmptySlice(t *testing.T) {
	mcpTools := []MCPTool{}

	apiTools := toAPITools(mcpTools)

	assert.Len(t, apiTools, 0)
	assert.NotNil(t, apiTools) // Should be empty slice, not nil
}

func TestToAPIToolsNilSlice(t *testing.T) {
	var mcpTools []MCPTool = nil

	apiTools := toAPITools(mcpTools)

	assert.Len(t, apiTools, 0)
	assert.NotNil(t, apiTools) // Should be empty slice, not nil
}

func TestToAPIToolsSingleTool(t *testing.T) {
	mcpTools := []MCPTool{
		{
			Tool: api.Tool{
				Type: "function",
				Function: api.ToolFunction{
					Name:        "single-tool",
					Description: "A single tool",
				},
			},
		},
	}

	apiTools := toAPITools(mcpTools)

	assert.Len(t, apiTools, 1)
	assert.Equal(t, "function", apiTools[0].Type)
	assert.Equal(t, "single-tool", apiTools[0].Function.Name)
	assert.Equal(t, "A single tool", apiTools[0].Function.Description)
}

func TestToAPIToolsPreservesOrder(t *testing.T) {
	mcpTools := []MCPTool{
		{
			Tool: api.Tool{
				Function: api.ToolFunction{Name: "first"},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{Name: "second"},
			},
		},
		{
			Tool: api.Tool{
				Function: api.ToolFunction{Name: "third"},
			},
		},
	}

	apiTools := toAPITools(mcpTools)

	assert.Len(t, apiTools, 3)
	assert.Equal(t, "first", apiTools[0].Function.Name)
	assert.Equal(t, "second", apiTools[1].Function.Name)
	assert.Equal(t, "third", apiTools[2].Function.Name)
}

func TestToAPIToolsComplexParameters(t *testing.T) {
	mcpTools := []MCPTool{
		{
			Tool: api.Tool{
				Type: "function",
				Function: api.ToolFunction{
					Name:        "complex-tool",
					Description: "A tool with complex parameters",
				},
			},
		},
	}

	apiTools := toAPITools(mcpTools)

	assert.Len(t, apiTools, 1)
	tool := apiTools[0]

	assert.Equal(t, "complex-tool", tool.Function.Name)
	assert.Equal(t, "A tool with complex parameters", tool.Function.Description)
}

// Benchmark tests
func BenchmarkGetCurrentTimeMs(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := getCurrentTimeMs()
		_ = result // Use result to prevent optimization
	}
}

func BenchmarkFindMCPToolByName(b *testing.B) {
	// Create a slice with many tools
	tools := make([]MCPTool, 100)
	for i := 0; i < 100; i++ {
		tools[i] = MCPTool{
			Tool: api.Tool{
				Function: api.ToolFunction{
					Name: "tool-" + string(rune('0'+(i%10))),
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := findMCPToolByName(tools, "tool-5")
		_ = result
	}
}

func BenchmarkToAPITools(b *testing.B) {
	// Create a slice with many MCP tools
	mcpTools := make([]MCPTool, 50)
	for i := 0; i < 50; i++ {
		mcpTools[i] = MCPTool{
			Tool: api.Tool{
				Type: "function",
				Function: api.ToolFunction{
					Name:        "tool-" + string(rune('0'+(i%10))),
					Description: "Test tool",
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		apiTools := toAPITools(mcpTools)
		_ = apiTools
	}
}
