package agentboot

import (
	"strings"
	"testing"

	"github.com/SaiNageswarS/agent-boot/prompts"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

func TestFormatToolInputsToMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		params      api.ToolCallFunctionArguments
		expected    []string // strings that should be present in the output
		notExpected []string // strings that should not be present
	}{
		{
			name:     "empty parameters",
			toolName: "calculator",
			params:   api.ToolCallFunctionArguments{},
			expected: []string{
				"Tool: `calculator` (no parameters)",
			},
		},
		{
			name:     "single string parameter",
			toolName: "search",
			params: api.ToolCallFunctionArguments{
				"query": "machine learning",
			},
			expected: []string{
				"Tool: `search`",
				"Parameters:",
				"- **query**: machine learning",
			},
		},
		{
			name:     "multiple parameters",
			toolName: "database",
			params: api.ToolCallFunctionArguments{
				"sql":    "SELECT * FROM users",
				"limit":  100,
				"offset": 0,
			},
			expected: []string{
				"Tool: `database`",
				"Parameters:",
				"- **limit**: 100",
				"- **offset**: 0",
				"- **sql**: SELECT \\* FROM users", // * is escaped
			},
		},
		{
			name:     "string slice parameter",
			toolName: "filter",
			params: api.ToolCallFunctionArguments{
				"categories": []interface{}{"tech", "science", "ai"},
			},
			expected: []string{
				"Tool: `filter`",
				"Parameters:",
				"- **categories**: tech, science, ai",
			},
		},
		{
			name:     "special characters in tool name and parameters",
			toolName: "test<>&\"'",
			params: api.ToolCallFunctionArguments{
				"input": "test<>&\"'",
			},
			expected: []string{
				"Tool: `test&lt;&gt;&\"'`",
				"- **input**: test&lt;&gt;&\"'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolInputsToMarkdown(tt.toolName, tt.params)

			// Check expected strings are present
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Output should contain: %s", expected)
			}

			// Check not expected strings are not present
			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, result, notExpected, "Output should not contain: %s", notExpected)
			}

			// Basic validation
			assert.NotEmpty(t, result, "Result should not be empty")
			if len(tt.params) > 0 {
				assert.Contains(t, result, "Parameters:", "Should contain Parameters section when params exist")
			}
		})
	}
}

func TestFormatToolInputsToMarkdownDeterministic(t *testing.T) {
	// Test that output is deterministic (parameters are sorted)
	params := api.ToolCallFunctionArguments{
		"z_param": "last",
		"a_param": "first",
		"m_param": "middle",
	}

	result1 := formatToolInputsToMarkdown("test", params)
	result2 := formatToolInputsToMarkdown("test", params)

	assert.Equal(t, result1, result2, "Output should be deterministic")

	// Check that parameters appear in sorted order (note: _ is escaped to \_)
	aPos := strings.Index(result1, "- **a\\_param**:")
	mPos := strings.Index(result1, "- **m\\_param**:")
	zPos := strings.Index(result1, "- **z\\_param**:")

	assert.True(t, aPos < mPos && mPos < zPos, "Parameters should be sorted alphabetically")
}

func TestEndToEndToolInputsInSummarization(t *testing.T) {
	// Test that tool inputs are properly included in summarization
	toolInputs := formatToolInputsToMarkdown("search", api.ToolCallFunctionArguments{
		"query": "machine learning",
		"limit": 5,
	})

	userQuery := "What are the latest ML trends?"
	content := "Recent developments in machine learning include advances in transformer models and neural architecture search."

	systemPrompt, userPrompt, err := prompts.RenderSummarizationPrompt(userQuery, content, toolInputs)
	assert.NoError(t, err, "Should render prompt without error")

	// Verify system prompt mentions tool inputs
	assert.Contains(t, systemPrompt, "tool inputs", "System prompt should mention tool inputs")

	// Verify user prompt includes tool inputs section
	assert.Contains(t, userPrompt, "Tool Inputs:", "User prompt should have Tool Inputs section")
	assert.Contains(t, userPrompt, "Tool: `search`", "Should contain tool name")
	assert.Contains(t, userPrompt, "machine learning", "Should contain tool parameters")

	// Verify instructions mention both question and tool inputs
	assert.Contains(t, userPrompt, "user's question and tool inputs", "Instructions should mention both")
}
