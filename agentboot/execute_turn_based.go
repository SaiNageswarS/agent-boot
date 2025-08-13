package agentboot

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

// ExecuteTurnBased executes the agent using turn-based mode with support for native tool calling
func (a *Agent) Execute(ctx context.Context, reporter ProgressReporter, req *schema.GenerateAnswerRequest) (*schema.StreamComplete, error) {
	startTime := getCurrentTimeMs()

	response := &schema.StreamComplete{ToolsUsed: []string{}, Metadata: map[string]string{}}

	msgs := []llm.Message{
		{Role: "user", Content: req.Question},
	}
	if a.config.SystemPrompt != "" {
		msgs = append([]llm.Message{{Role: "system", Content: a.config.SystemPrompt}}, msgs...)
	}

	// Step 1: Select tools using gpt-oss
	toolCalls := a.SelectTools(ctx, reporter, msgs)
	// Run Tool Calls
	for _, toolCall := range toolCalls {
		toolResultContext, err := a.RunTool(ctx, reporter, req.Question, &toolCall)
		if err != nil {
			continue
		}

		msgs = append(msgs, llm.Message{
			Role:    "user",
			Content: toolResultContext,
		})
	}

	// Step 2: Run LLM with the selected tools
	var inference strings.Builder
	err := a.config.BigModel.GenerateInference(
		ctx, msgs,
		func(chunk string) error {
			inference.WriteString(chunk)
			reporter.Send(NewAnswerChunk(&schema.AnswerChunk{Content: chunk}))
			return nil
		},
		llm.WithMaxTokens(a.config.MaxTokens),
		llm.WithTemperature(0.7),
	)

	if err != nil {
		logger.Error("Failed to run inference", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "inference_failed"))
	}

	response.Answer = inference.String()
	response.ProcessingTime = getCurrentTimeMs() - startTime

	reporter.Send(NewStreamComplete(response))
	return response, nil
}

func (a *Agent) SelectTools(ctx context.Context, reporter ProgressReporter, msgs []llm.Message) []api.ToolCall {
	var toolCalls []api.ToolCall
	toolSelector := llm.NewOllamaClient("gpt-oss:20b")
	err := toolSelector.GenerateInferenceWithTools(
		ctx, msgs,
		func(chunk string) error { return nil }, // ignore Answer
		func(calls []api.ToolCall) error {
			toolCalls = append(toolCalls, calls...)
			return nil
		},
		llm.WithTools(toAPITools(a.config.Tools)),
		llm.WithMaxTokens(a.config.MaxTokens),
	)

	if err != nil {
		logger.Error("Failed to select tools", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "tool_selection_failed"))
	}

	return toolCalls
}
