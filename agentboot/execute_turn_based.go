package agentboot

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/memory"
	"github.com/SaiNageswarS/agent-boot/prompts"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

// ExecuteTurnBased executes the agent using turn-based mode with support for native tool calling
func (a *Agent) Execute(ctx context.Context, reporter ProgressReporter, req *schema.GenerateAnswerRequest) (*schema.StreamComplete, error) {
	startTime := getCurrentTimeMs()

	response := &schema.StreamComplete{ToolsUsed: []string{}, Metadata: map[string]string{}}

	// Load previous conversation messages
	conversation := &memory.Conversation{}
	if a.config.ConversationManager != nil {
		conversation = a.config.ConversationManager.LoadSession(ctx, req.SessionId)
	}

	// Add user message to conversation
	conversation.AddUserMessage(req.Question)

	for turn := 0; turn < a.config.MaxTurns; turn++ {
		// Step 1: Select tools using gpt-oss
		toolCalls := a.SelectTools(ctx, reporter, conversation.Messages, turn)
		// Run Tool Calls
		for _, toolCall := range toolCalls {
			toolResultContext, err := a.RunTool(ctx, reporter, req.Question, &toolCall)
			if err != nil {
				continue
			}

			// Add tool result to conversation
			conversation.AddToolResult(toolResultContext)
		}
	}

	// Step 2: Run LLM with the selected tools
	var inference strings.Builder
	err := a.config.BigModel.GenerateInference(
		ctx, conversation.Messages,
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

	conversation.AddAssistantMessage(response.Answer)
	// Save session with assistant response
	if a.config.ConversationManager != nil {
		a.config.ConversationManager.SaveSession(ctx, conversation)
	}

	reporter.Send(NewStreamComplete(response))
	return response, nil
}

func (a *Agent) SelectTools(ctx context.Context, reporter ProgressReporter, msgs []llm.Message, turn int) []api.ToolCall {
	var toolCalls []api.ToolCall

	// Render tool selection system prompt
	systemPrompt, err := prompts.RenderToolSelectionPrompt(turn)
	if err != nil {
		logger.Error("Failed to render tool selection prompt", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "prompt_rendering_failed"))
		return toolCalls
	}

	err = a.config.ToolSelector.GenerateInferenceWithTools(
		ctx, msgs,
		func(chunk string) error { return nil }, // ignore Answer
		func(calls []api.ToolCall) error {
			toolCalls = append(toolCalls, calls...)
			return nil
		},
		llm.WithTools(toAPITools(a.config.Tools)),
		llm.WithMaxTokens(a.config.MaxTokens),
		llm.WithSystemPrompt(systemPrompt),
	)

	if err != nil {
		logger.Error("Failed to select tools", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "tool_selection_failed"))
	}

	return toolCalls
}
