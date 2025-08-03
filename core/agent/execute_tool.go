package agent

import (
	"agent-boot/proto/schema"
	"context"
	"maps"
	"strings"

	"github.com/SaiNageswarS/agent-boot/core/llm"
	"github.com/SaiNageswarS/agent-boot/core/prompts"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

func (a *Agent) ExecuteTool(ctx context.Context, selection *schema.SelectedTool) <-chan *schema.ToolExecutionResultChunk {
	out := make(chan *schema.ToolExecutionResultChunk, 1)
	defer close(out)

	tool := a.GetToolByName(selection.Name)

	// Execute the tool handler
	toolResultChan := tool.Handler(ctx, selection.Parameters)

	for toolResult := range toolResultChan {
		// Summarize the result if the tool has summarization enabled
		if tool.SummarizeContext {
			summarizedResult := a.summarizeResult(ctx, toolResult, selection.Query)
			if summarizedResult != nil {
				out <- summarizedResult
			}
		} else {
			out <- toolResult
		}
	}

	return out
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
func (a *Agent) summarizeResult(ctx context.Context, chunk *schema.ToolExecutionResultChunk, userQuery string) *schema.ToolExecutionResultChunk {
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
	err = a.config.MiniModel.Client.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithLLMModel(a.config.MiniModel.Model),
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
	summarizedResult := &schema.ToolExecutionResultChunk{
		Sentences:   []string{summary},
		Attribution: chunk.Attribution, // Preserve attributions
		Title:       chunk.Title,
		Metadata:    make(map[string]string),
	}

	// Copy metadata and add summarization info
	maps.Copy(summarizedResult.Metadata, chunk.Metadata)
	summarizedResult.Metadata["summarized"] = "true"
	summarizedResult.Metadata["original_sentence_count"] = string(len(chunk.Sentences))

	return summarizedResult
}
