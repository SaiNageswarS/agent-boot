package agentboot

import (
	"time"

	"github.com/ollama/ollama/api"
)

func getCurrentTimeMs() int64 {
	// Use time package to get current time in milliseconds
	return time.Now().UnixMilli()
}

// findMCPToolByName finds an MCPTool by its function name
func findMCPToolByName(tools []MCPTool, name string) *MCPTool {
	for _, tool := range tools {
		if tool.Function.Name == name {
			return &tool
		}
	}
	return nil
}

// toAPITools converts MCPTools to api.Tools for native tool calling
func toAPITools(tools []MCPTool) []api.Tool {
	apiTools := make([]api.Tool, len(tools))
	for i, tool := range tools {
		apiTools[i] = tool.Tool
	}
	return apiTools
}
