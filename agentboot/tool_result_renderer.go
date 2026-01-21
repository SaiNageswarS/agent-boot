package agentboot

import (
	"context"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/prompts"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"go.uber.org/zap"
)

type ToolResultRenderer struct {
	reporter           ProgressReporter
	summarizationModel llm.LLMClient
	toolName           string
}

// ToolResultRendererOption is a functional option for configuring ToolResultRenderer
type ToolResultRendererOption func(*ToolResultRenderer)

// NewToolResultRenderer creates a new ToolResultRenderer with the given options.
// By default, it uses NoOpProgressReporter if no reporter is provided.
func NewToolResultRenderer(opts ...ToolResultRendererOption) *ToolResultRenderer {
	r := &ToolResultRenderer{
		reporter: &NoOpProgressReporter{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithReporter sets the progress reporter and tool name.
// toolName is required when using a reporter for proper progress reporting.
func WithReporter(reporter ProgressReporter, toolName string) ToolResultRendererOption {
	return func(r *ToolResultRenderer) {
		r.reporter = reporter
		r.toolName = toolName
	}
}

// WithSummarizationModel sets the LLM client for summarizing tool results.
// This is required when calling Render with summarizeResult=true.
func WithSummarizationModel(model llm.LLMClient) ToolResultRendererOption {
	return func(r *ToolResultRenderer) {
		r.summarizationModel = model
	}
}

func (r *ToolResultRenderer) Render(ctx context.Context, query, toolInputsMD string, toolResultChan <-chan *schema.ToolResultChunk, summarizeResult bool) ([]string, error) {
	// Parallel stream processing of tool results
	linqCtx, cancel := context.WithCancel(ctx)
	toolResultChunks, err := linq.Pipe4(
		linq.NewStream(linqCtx, toolResultChan, cancel, 10),

		linq.SelectPar(func(raw *schema.ToolResultChunk) *schema.ToolResultChunk {
			if summarizeResult {
				return r.summarizeResult(linqCtx, raw, query, toolInputsMD)
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
			r.reporter.Send(NewToolExecutionResult(r.toolName, chunk))
			s := formatToolResultToMD(chunk)
			return string(s)
		}),

		linq.ToSlice[string](),
	)

	return toolResultChunks, err
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
func (r *ToolResultRenderer) summarizeResult(ctx context.Context, chunk *schema.ToolResultChunk, userQuery, toolInputs string) *schema.ToolResultChunk {
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

	logger.Info("Summarizing Result",
		zap.String("title", chunk.Title),
		zap.Int("sentence_count", len(chunk.Sentences)),
		zap.String("query", userQuery),
		zap.String("tool_inputs", toolInputs))

	// Join all sentences into a single text
	combinedText := strings.Join(chunk.Sentences, " ")

	systemPrompt, userPrompt, err := prompts.RenderSummarizationPrompt(userQuery, combinedText, toolInputs)
	if err != nil {
		// If template rendering fails, keep the original result
		logger.Error("Failed to render summarization prompt", zap.String("title", chunk.Title), zap.Error(err))
		return chunk
	}

	messages := []llm.Message{
		{Role: "user", Content: userPrompt},
	}

	var responseContent strings.Builder
	err = r.summarizationModel.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithTemperature(0.3),
		llm.WithSystemPrompt(systemPrompt),
	)

	if err != nil {
		// If summarization fails, keep the original result
		logger.Error("Failed to summarize tool result", zap.String("title", chunk.Title), zap.Error(err))
		return chunk
	}

	summary := strings.TrimSpace(responseContent.String())

	// Drop irrelevant content
	if strings.Contains(summary, "# IRRELEVANT") {
		logger.Info("Dropping irrelevant tool result", zap.String("title", chunk.Title))
		return nil
	}

	// Create new summarized result
	summarizedResult := &schema.ToolResultChunk{
		Sentences:   strings.Split(summary, "\n"),
		Attribution: chunk.Attribution,
		Title:       chunk.Title,
		Metadata:    make(map[string]string),
	}

	logger.Info("Summarized tool result", zap.Int("original_sentence_count", len(chunk.Sentences)), zap.Int("summarized_sentence_count", len(summarizedResult.Sentences)))
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
		b.WriteString(title)
		b.WriteString("\n\n")
	}
	// Show "via <tool>" only if it's different from the title we used.
	if tool != "" && tool != title {
		b.WriteString("_via `")
		b.WriteString(tool)
		b.WriteString("`_\n\n")
	}

	if errText := strings.TrimSpace(result.Error); errText != "" {
		b.WriteString("> **Error:** ")
		b.WriteString(errText)
		b.WriteString("\n\n")
	}

	// Sentences
	if n := len(result.Sentences); n > 0 {
		if n == 1 {
			b.WriteString(strings.TrimSpace(result.Sentences[0]))
			b.WriteString("\n\n")
		} else {
			for _, s := range result.Sentences {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(s)
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
			b.WriteString(k)
			b.WriteString(" | ")
			b.WriteString(result.Metadata[k])
			b.WriteString(" |\n")
		}
		b.WriteByte('\n')
	}

	if att := strings.TrimSpace(result.Attribution); att != "" {
		b.WriteString("**Attribution**: ")
		b.WriteString(att)
	}

	return b.String()
}
