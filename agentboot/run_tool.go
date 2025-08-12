package agentboot

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/prompts"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

func (a *Agent) RunTool(ctx context.Context, reporter ProgressReporter, query string, selection *api.ToolCall) (string, error) {
	reporter.Send(NewProgressUpdate(
		schema.Stage_tool_execution_starting,
		fmt.Sprintf("Running tool %s with arguments: %v", selection.Function.Name, selection.Function.Arguments)))

	tool := findMCPToolByName(a.config.Tools, selection.Function.Name)

	// Execute the tool handler
	toolResultChan := tool.Handler(ctx, selection.Function.Arguments)

	// Parallel stream processing of tool results
	linqCtx, cancel := context.WithCancel(ctx)
	toolResultChunks, err := linq.Pipe4(
		linq.NewStream(linqCtx, toolResultChan, cancel, 10),

		linq.SelectPar(func(raw *schema.ToolResultChunk) *schema.ToolResultChunk {
			if tool.SummarizeContext {
				return a.summarizeResult(linqCtx, raw, query)
			}

			// If summarization is not enabled, return the raw result
			return raw
		}),

		linq.Where(func(chunk *schema.ToolResultChunk) bool {
			// Filter out nil results and those marked as irrelevant
			if chunk == nil {
				return false
			}

			return true
		}),

		linq.Select(func(chunk *schema.ToolResultChunk) string {
			reporter.Send(NewToolExecutionResult(tool.Function.Name, chunk))
			s := formatToolResultToMD(chunk)
			return string(s)
		}),

		linq.ToSlice[string](),
	)

	if err != nil {
		logger.Error("Error executing tool", zap.String("tool", selection.Function.Name), zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "tool_execution_failed"))
		return "", err
	}

	return strings.Join(toolResultChunks, "\n\n"), nil
}

// summarizeToolResults summarizes tool results using the mini model to make them more relevant and concise.
// This method:
// 1. Combines all sentences from each ToolResult into a single text
// 2. Uses the mini model to summarize the content with respect to the user's query
// 3. Filters out irrelevant content completely
// 4. Preserves original metadata and attributions
// 5. Adds summarization metadata for transparency
//
// This is particularly useful for:
// - RAG search results that may contain verbose or tangential information
// - Web search results with mixed relevant/irrelevant content
// - Large document chunks that need to be condensed for context windows
func (a *Agent) summarizeResult(ctx context.Context, chunk *schema.ToolResultChunk, userQuery string) *schema.ToolResultChunk {
	if chunk == nil {
		logger.Error("Received nil tool result chunk for summarization")
		return nil
	}

	if chunk.Error != "" {
		return chunk // Return as is if there's an error
	}

	if len(chunk.Sentences) == 0 {
		// Skip empty results
		logger.Info("Skipping summarization for empty tool result", zap.String("title", chunk.Title))
		return nil
	}

	// Join all sentences into a single text
	combinedText := strings.Join(chunk.Sentences, " ")

	// Create summarization prompt using templates
	promptData := prompts.SummarizationPromptData{
		Query:   userQuery,
		Content: combinedText,
	}

	systemPrompt, userPrompt, err := prompts.RenderSummarizationPrompt(promptData)
	if err != nil {
		// If template rendering fails, keep the original result
		logger.Error("Failed to render summarization prompt", zap.String("title", chunk.Title), zap.Error(err))
		return chunk
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	var responseContent strings.Builder
	err = a.config.MiniModel.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithTemperature(0.3),
		llm.WithMaxTokens(200),
	)

	if err != nil {
		// If summarization fails, keep the original result
		logger.Error("Failed to summarize tool result", zap.String("title", chunk.Title), zap.Error(err))
		return chunk
	}

	summary := strings.TrimSpace(responseContent.String())

	// Drop irrelevant content
	if strings.ToUpper(summary) == "IRRELEVANT" {
		logger.Info("Dropping irrelevant tool result", zap.String("title", chunk.Title))
		return nil
	}

	// Create new summarized result
	summarizedResult := &schema.ToolResultChunk{
		Sentences:   []string{summary},
		Attribution: chunk.Attribution, // Preserve attributions
		Title:       chunk.Title,
		Metadata:    make(map[string]string),
	}

	// Copy metadata and add summarization info
	maps.Copy(summarizedResult.Metadata, chunk.Metadata)
	summarizedResult.Metadata["summarized"] = "true"
	summarizedResult.Metadata["original_sentence_count"] = strconv.Itoa(len(chunk.Sentences))

	return summarizedResult
}

func formatToolResultToMD(result *schema.ToolResultChunk) string {
	if result == nil {
		return ""
	}

	var b strings.Builder

	title := strings.TrimSpace(result.Title)
	tool := strings.TrimSpace(result.ToolName)
	if title == "" && tool != "" {
		title = tool
	}
	if title != "" {
		b.WriteString("### ")
		b.WriteString(mdEscape(title))
		b.WriteString("\n\n")
	}
	// Show "via <tool>" only if it's different from the title we used.
	if tool != "" && tool != title {
		b.WriteString("_via `")
		b.WriteString(mdEscape(tool))
		b.WriteString("`_\n\n")
	}

	if errText := strings.TrimSpace(result.Error); errText != "" {
		b.WriteString("> **Error:** ")
		b.WriteString(mdEscape(errText))
		b.WriteString("\n\n")
	}

	// Sentences
	if n := len(result.Sentences); n > 0 {
		if n == 1 {
			b.WriteString(mdEscape(strings.TrimSpace(result.Sentences[0])))
			b.WriteString("\n\n")
		} else {
			for _, s := range result.Sentences {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(mdEscape(s))
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
	}

	// Metadata (sorted for deterministic output)
	if len(result.Metadata) > 0 {
		keys := make([]string, 0, len(result.Metadata))
		for k := range result.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		b.WriteString("| Key | Value |\n|---|---|\n")
		for _, k := range keys {
			b.WriteString("| ")
			b.WriteString(mdEscape(k))
			b.WriteString(" | ")
			b.WriteString(mdEscape(result.Metadata[k]))
			b.WriteString(" |\n")
		}
		b.WriteByte('\n')
	}

	if att := strings.TrimSpace(result.Attribution); att != "" {
		b.WriteString("_Attribution: ")
		b.WriteString(mdEscape(att))
		b.WriteString("_\n")
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
