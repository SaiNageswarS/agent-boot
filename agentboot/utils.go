package agentboot

import (
	"time"

	"github.com/SaiNageswarS/agent-boot/llm"
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

// TrimForSession keeps the last maxUser "user" messages and any number of
// "assistant" (and optional "tool") messages that follow them.
// If there are fewer than maxUser user messages total, it returns msgs unchanged.
func trimForSession(msgs []llm.Message, maxUser int) []llm.Message {
	if maxUser <= 0 || len(msgs) == 0 {
		return []llm.Message{}
	}

	// Walk backward and find the boundary index: the position right after the
	// (maxUser+1)-th user from the end. Everything after boundary is kept.
	usersSeen := 0
	start := 0 // default: keep all if we don't exceed maxUser users
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" && !msgs[i].IsToolResult {
			usersSeen++
			start = i
			if usersSeen == maxUser {
				break
			}
		}
	}

	return msgs[start:]
}
