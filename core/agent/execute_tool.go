package agent

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/core/llm"
	"github.com/SaiNageswarS/agent-boot/core/prompts"
)

func (a *Agent) ExecuteTool(ctx context.Context, selection ToolSelection) ([]*ToolResultChunk, error) {
	results, err := selection.Tool.Handler(ctx, selection.Parameters)
	if err != nil {
		return nil, err
	}

	// Apply summarization if enabled for this tool
	if selection.Tool.SummarizeContext {
		// Get the user query from context metadata if available
		query := ""
		if queryParam, ok := selection.Parameters["query"].(string); ok {
			query = queryParam
		}

		summarizedResults, err := a.summarizeToolResults(ctx, results, query)
		if err != nil {
			// If summarization fails, return original results
			return results, nil
		}
		return summarizedResults, nil
	}

	return results, nil
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
func (a *Agent) summarizeToolResults(ctx context.Context, resultChunks []*ToolResultChunk, userQuery string) ([]*ToolResultChunk, error) {
	if len(resultChunks) == 0 {
		return resultChunks, nil
	}

	summarizedResults := make([]*ToolResultChunk, 0, len(resultChunks))

	for _, result := range resultChunks {
		if len(result.Sentences) == 0 {
			// Skip empty results
			continue
		}

		// Join all sentences into a single text
		combinedText := strings.Join(result.Sentences, " ")

		// Create summarization prompt using templates
		promptData := prompts.SummarizationPromptData{
			Query:   userQuery,
			Content: combinedText,
		}

		systemPrompt, userPrompt, err := prompts.RenderSummarizationPrompt(promptData)
		if err != nil {
			// If template rendering fails, keep the original result
			summarizedResults = append(summarizedResults, result)
			continue
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
			summarizedResults = append(summarizedResults, result)
			continue
		}

		summary := strings.TrimSpace(responseContent.String())

		// Drop irrelevant content
		if strings.ToUpper(summary) == "IRRELEVANT" {
			continue
		}

		// Create new summarized result
		summarizedResult := &ToolResultChunk{
			Sentences:   []string{summary},
			Attribution: result.Attribution, // Preserve attributions
			Title:       result.Title,
			Metadata:    make(map[string]interface{}),
		}

		// Copy metadata and add summarization info
		for k, v := range result.Metadata {
			summarizedResult.Metadata[k] = v
		}
		summarizedResult.Metadata["summarized"] = true
		summarizedResult.Metadata["original_sentence_count"] = len(result.Sentences)

		summarizedResults = append(summarizedResults, summarizedResult)
	}

	return summarizedResults, nil
}
