package agent

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

// ExecuteTurnBased executes the agent using turn-based mode with support for native tool calling
func (a *Agent) Execute(ctx context.Context, reporter ProgressReporter, req *schema.GenerateAnswerRequest) (*schema.StreamComplete, error) {
	startTime := getCurrentTimeMs()

	response := &schema.StreamComplete{ToolsUsed: []string{}, Metadata: map[string]string{}}

	msgs := []llm.Message{
		{Role: "user", Content: req.Question},
	}
	if req.Context != "" {
		msgs = append([]llm.Message{{Role: "system", Content: req.Context}}, msgs...)
	}

	var inference string
	var toolCalls []api.ToolCall
	var err error

	for turns := 0; turns < a.config.MaxTurns; turns++ {
		inference, toolCalls, err = a.RunLLM(ctx, msgs, reporter)
		if err != nil {
			reporter.Send(NewStreamError(err.Error(), "inference_failed"))
			return nil, err
		}

		// Final Answer Generated
		if len(toolCalls) == 0 {
			break
		}

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
	}

	response.Answer = inference
	response.ProcessingTime = getCurrentTimeMs() - startTime

	reporter.Send(NewStreamComplete(response))
	return response, nil
}

func (a *Agent) RunLLM(ctx context.Context, msgs []llm.Message, reporter ProgressReporter) (string, []api.ToolCall, error) {
	var inference strings.Builder
	var toolCalls []api.ToolCall

	err := a.config.BigModel.GenerateInferenceWithTools(
		ctx, msgs,
		func(chunk string) error {
			inference.WriteString(chunk)
			if len(toolCalls) == 0 {
				reporter.Send(NewAnswerChunk(&schema.AnswerChunk{Content: chunk}))
			}
			return nil
		},
		func(calls []api.ToolCall) error {
			toolCalls = append(toolCalls, calls...)
			return nil
		},
		llm.WithTools(toAPITools(a.config.Tools)),
		llm.WithMaxTokens(a.config.MaxTokens),
	)

	return inference.String(), toolCalls, err
}
