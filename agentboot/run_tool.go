package agentboot

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

func (a *Agent) RunTool(ctx context.Context, reporter ProgressReporter, query string, selection *api.ToolCall) (string, error) {
	reporter.Send(NewProgressUpdate(
		schema.Stage_tool_execution_starting,
		fmt.Sprintf("Running tool %s with arguments: %v", selection.Function.Name, selection.Function.Arguments)))

	tool := findMCPToolByName(a.config.Tools, selection.Function.Name)

	// Format tool inputs for summarization context
	toolInputsMD := formatToolInputsToMarkdown(selection.Function.Name, selection.Function.Arguments)

	// Execute the tool handler
	toolResultChan := tool.Handler(ctx, selection.Function.Arguments)

	r := &ToolResultRenderer{
		reporter:           reporter,
		summarizationModel: a.config.MiniModel,
		toolName:           selection.Function.Name,
	}

	toolResultChunks, err := r.Render(ctx, query, toolInputsMD, toolResultChan, tool.SummarizeContext)
	if err != nil {
		logger.Error("Error rendering tool result", zap.String("tool", selection.Function.Name), zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "tool_execution_failed"))
		return "", err
	}

	reporter.Send(NewProgressUpdate(
		schema.Stage_tool_execution_completed,
		fmt.Sprintf("Tool %s completed successfully", selection.Function.Name)))
	return strings.Join(toolResultChunks, "\n\n"), nil
}

// formatToolInputsToMarkdown formats tool inputs as markdown for use in summarization prompts
func formatToolInputsToMarkdown(toolName string, params api.ToolCallFunctionArguments) string {
	if len(params) == 0 {
		return fmt.Sprintf("Tool: `%s` (no parameters)", mdEscape(toolName))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tool: `%s`\n\n", mdEscape(toolName)))

	// Sort parameters for deterministic output
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b.WriteString("Parameters:\n")
	for _, k := range keys {
		value := params[k]
		var valueStr string

		// Format the value appropriately
		switch v := value.(type) {
		case string:
			valueStr = v
		case []string:
			valueStr = strings.Join(v, ", ")
		case []interface{}:
			strs := make([]string, len(v))
			for i, item := range v {
				strs[i] = fmt.Sprintf("%v", item)
			}
			valueStr = strings.Join(strs, ", ")
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		b.WriteString(fmt.Sprintf("- **%s**: %s\n", mdEscape(k), mdEscape(valueStr)))
	}

	return b.String()
}

// Minimal Markdown escaper for headings, lists, and table cells.
func mdEscape(s string) string {
	if s == "" {
		return s
	}
	// Backslash first
	s = strings.ReplaceAll(s, `\`, `\\`)
	// Table & inline syntax
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "*", `\*`)
	s = strings.ReplaceAll(s, "_", `\_`)
	s = strings.ReplaceAll(s, "~", `\~`)
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "[", `\[`)
	s = strings.ReplaceAll(s, "]", `\]`)
	s = strings.ReplaceAll(s, "#", `\#`)
	// Angle brackets -> HTML entities to avoid autolinks
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
