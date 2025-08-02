package agent

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/core/llm"
	"github.com/SaiNageswarS/agent-boot/core/prompts"
	"github.com/SaiNageswarS/go-collection-boot/linq"
)

func (a *Agent) ExecuteTool(ctx context.Context, selection ToolSelection) ([]*ToolResultChunk, error) {
	a.reportProgress(NewToolExecutionEvent(
		"execution_starting",
		fmt.Sprintf("Starting execution of tool: %s", selection.Tool.Name),
		&ToolExecutionProgress{
			ToolName:   selection.Tool.Name,
			Parameters: selection.Parameters,
			Status:     "starting",
		},
	))

	toolStartTime := time.Now()
	results, err := selection.Tool.Handler(ctx, selection.Parameters)
	if err != nil {
		a.reportProgress(NewErrorEvent(
			"tool_execution",
			fmt.Sprintf("Tool %s execution failed", selection.Tool.Name),
			err.Error(),
		))

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

	toolDuration := time.Since(toolStartTime)

	// Report tool execution completed with results
	a.reportProgress(NewToolExecutionEvent(
		"execution_completed",
		fmt.Sprintf("Tool execution completed: %s", selection.Tool.Name),
		&ToolExecutionProgress{
			ToolName:   selection.Tool.Name,
			Parameters: selection.Parameters,
			Status:     "completed",
		},
	))

	// Report tool results
	a.reportProgress(NewToolResultEvent(
		"results_available",
		fmt.Sprintf("Tool %s returned %d results", selection.Tool.Name, len(results)),
		&ToolResultProgress{
			ToolName:   selection.Tool.Name,
			Results:    results,
			Success:    true,
			Duration:   toolDuration,
			Summarized: selection.Tool.SummarizeContext,
		},
	))
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

	summarizeResult := func(chunk *ToolResultChunk) *ToolResultChunk {
		if len(chunk.Sentences) == 0 {
			// Skip empty results
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
			fmt.Printf("Failed to render summarization prompt: %v\n", err)
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
			fmt.Printf("Failed to summarize tool result: %v\n", err)
			return chunk
		}

		summary := strings.TrimSpace(responseContent.String())

		// Drop irrelevant content
		if strings.ToUpper(summary) == "IRRELEVANT" {
			return nil
		}

		// Create new summarized result
		summarizedResult := &ToolResultChunk{
			Sentences:   []string{summary},
			Attribution: chunk.Attribution, // Preserve attributions
			Title:       chunk.Title,
			Metadata:    make(map[string]interface{}),
		}

		// Copy metadata and add summarization info
		maps.Copy(summarizedResult.Metadata, chunk.Metadata)
		summarizedResult.Metadata["summarized"] = true
		summarizedResult.Metadata["original_sentence_count"] = len(chunk.Sentences)

		return summarizedResult
	}

	summarizedResults, err := linq.Pipe3(
		linq.FromSlice(ctx, resultChunks),
		linq.SelectPar(func(chunk *ToolResultChunk) *ToolResultChunk {
			return summarizeResult(chunk)
		}),
		linq.Where(func(chunk *ToolResultChunk) bool {
			return chunk != nil // Filter out nil results
		}),
		linq.ToSlice[*ToolResultChunk](),
	)

	return summarizedResults, err
}
