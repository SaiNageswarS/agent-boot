package agentboot

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/agent-boot/session"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

// ExecuteTurnBased executes the agent using turn-based mode with support for native tool calling
func (a *Agent) Execute(ctx context.Context, reporter ProgressReporter, req *schema.GenerateAnswerRequest) (*schema.StreamComplete, error) {
	startTime := getCurrentTimeMs()

	response := &schema.StreamComplete{ToolsUsed: []string{}, Metadata: map[string]string{}}

	msgs := make([]llm.Message, 0, a.config.MaxSessionMsgs+1)
	if a.config.SessionCollection != nil {
		session, err := async.Await(a.config.SessionCollection.FindOneByID(ctx, req.SessionId))
		if err != nil {
			logger.Error("Failed to find session", zap.Error(err))
		} else {
			msgs = append(msgs, session.Messages...)
		}
	}

	msgs = append(msgs, llm.Message{Role: "user", Content: req.Question})

	// Step 1: Select tools using gpt-oss
	toolCalls := a.SelectTools(ctx, reporter, msgs)
	// Run Tool Calls
	for _, toolCall := range toolCalls {
		toolResultContext, err := a.RunTool(ctx, reporter, req.Question, &toolCall)
		if err != nil {
			continue
		}

		msgs = append(msgs, llm.Message{
			Role:         "user",
			Content:      toolResultContext,
			IsToolResult: true,
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
		llm.WithSystemPrompt(a.config.SystemPrompt),
	)

	if err != nil {
		logger.Error("Failed to run inference", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "inference_failed"))
	}

	response.Answer = inference.String()
	response.ProcessingTime = getCurrentTimeMs() - startTime

	// save session
	if a.config.SessionCollection != nil {
		msgs = append(msgs, llm.Message{Role: "assistant", Content: response.Answer})
		msgs = trimForSession(msgs, a.config.MaxSessionMsgs)

		session := session.SessionModel{
			ID:       req.SessionId,
			Messages: msgs,
		}
		_, err := async.Await(a.config.SessionCollection.Save(ctx, session))
		if err != nil {
			logger.Error("Failed to save session", zap.Error(err))
		}
	}

	reporter.Send(NewStreamComplete(response))
	return response, nil
}

func (a *Agent) SelectTools(ctx context.Context, reporter ProgressReporter, msgs []llm.Message) []api.ToolCall {
	var toolCalls []api.ToolCall

	err := a.config.ToolSelector.GenerateInferenceWithTools(
		ctx, msgs,
		func(chunk string) error { return nil }, // ignore Answer
		func(calls []api.ToolCall) error {
			toolCalls = append(toolCalls, calls...)
			return nil
		},
		llm.WithTools(toAPITools(a.config.Tools)),
		llm.WithMaxTokens(a.config.MaxTokens),
		llm.WithSystemPrompt(a.config.SystemPrompt),
	)

	if err != nil {
		logger.Error("Failed to select tools", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "tool_selection_failed"))
	}

	return toolCalls
}
