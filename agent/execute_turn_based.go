package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

// ExecuteTurnBased executes the agent using turn-based mode with support for native tool calling
func (a *Agent) Execute(ctx context.Context, reporter ProgressReporter, req *schema.GenerateAnswerRequest) (*schema.AnswerChunk, error) {
	startTime := getCurrentTimeMs()

	response := &schema.AnswerChunk{ToolsUsed: []string{}, Metadata: map[string]string{}}

	// Build initial conversation
	conversation := a.initialConversation(req)

	client, usedBig := a.pickClient(req.Question, nil)
	response.ModelUsed = client.GetModel()

	maxTurns := a.effectiveMaxTurns()
	useNative := (client.Capabilities()&llm.NativeToolCalling) != 0 && len(a.config.Tools) > 0

	reporter.Send(NewProgressUpdate(
		schema.Stage_tool_selection_starting,
		fmt.Sprintf("Starting turn-based execution (max %d turns, native tool calling: %v)", maxTurns, useNative),
		1,
	))

	for turn := 1; turn <= maxTurns; turn++ {
		reporter.Send(NewProgressUpdate(
			schema.Stage_tool_execution_starting,
			fmt.Sprintf("Executing turn %d/%d", turn, maxTurns),
			2,
		))

		var (
			turnResult *TurnResult
			err        error
		)
		if useNative {
			turnResult, err = a.nativeTurn(ctx, client, conversation, turn, reporter, startTime)
		} else {
			turnResult, err = a.promptTurn(ctx, client, conversation, turn, reporter, startTime)
		}
		if err != nil {
			logger.Error("Turn failed", zap.Error(err), zap.Int("turn", turn))
			reporter.Send(NewStreamError(err.Error(), fmt.Sprintf("Turn %d error", turn)))
			break
		}

		response.ToolsUsed = append(response.ToolsUsed, turnResult.ToolsUsed...)

		// Append conversation updates
		if len(turnResult.ToolCalls) > 0 { // native tool path tool results added
			conversation = append(conversation, llm.Message{Role: "assistant", Content: "I used tools to gather information."})
			for _, tc := range turnResult.ToolCalls {
				conversation = append(conversation, llm.Message{Role: "user", Content: fmt.Sprintf("Tool %s result: %s", tc.Function.Name, turnResult.ToolResults[tc.Function.Name])})
			}
		} else if turnResult.Answer != "" && !turnResult.IsComplete { // intermediate answer or reasoning
			conversation = append(conversation, llm.Message{Role: "assistant", Content: turnResult.Answer})
		}

		if turnResult.IsComplete && turnResult.Answer != "" { // final answer already produced
			response.Answer = turnResult.Answer
			break
		}
	}

	// Fallback final answer generation
	if response.Answer == "" {
		a.streamFinalAnswer(ctx, client, conversation, reporter, response, startTime)
	}

	// Metadata & finalization
	response.ProcessingTime = getCurrentTimeMs() - startTime
	response.IsFinal = true
	response.Metadata["tool_count"] = fmt.Sprintf("%d", len(response.ToolsUsed))
	response.Metadata["has_context"] = fmt.Sprintf("%v", req.Context != "")
	response.Metadata["used_big_model"] = fmt.Sprintf("%v", usedBig)
	response.Metadata["native_tool_calling"] = fmt.Sprintf("%v", useNative)

	reporter.Send(NewProgressUpdate(schema.Stage_answer_generation_completed, "Turn-based execution completed successfully", 3))
	reporter.Send(NewAnswerChunk(response))
	reporter.Send(NewStreamComplete("Turn-based execution completed"))
	return response, nil
}

// TurnResult represents the result of a single turn
type TurnResult struct {
	TurnNumber  int               `json:"turn_number"`
	ToolCalls   []api.ToolCall    `json:"tool_calls,omitempty"`
	ToolResults map[string]string `json:"tool_results,omitempty"`
	ToolsUsed   []string          `json:"tools_used,omitempty"`
	Answer      string            `json:"answer,omitempty"`
	IsComplete  bool              `json:"is_complete"`
	Error       string            `json:"error,omitempty"`
}

// nativeTurn handles a single native tool calling turn
func (a *Agent) nativeTurn(ctx context.Context, client llm.LLMClient, messages []llm.Message, turn int, reporter ProgressReporter, startTime int64) (*TurnResult, error) {
	res := &TurnResult{TurnNumber: turn, ToolResults: map[string]string{}, ToolsUsed: []string{}}

	var content strings.Builder
	var toolCalls []api.ToolCall

	err := client.GenerateInferenceWithTools(
		ctx,
		messages,
		func(chunk string) error { // content callback
			content.WriteString(chunk)
			if len(toolCalls) == 0 { // still generating potential direct answer
				a.streamPartialAnswer(content.String(), client, nil, startTime, reporter)
			}
			return nil
		},
		func(calls []api.ToolCall) error { toolCalls = append(toolCalls, calls...); return nil },
		llm.WithTools(ToAPITools(a.config.Tools)),
		llm.WithTemperature(0.3),
		llm.WithMaxTokens(a.config.MaxTokens),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate inference: %w", err)
	}

	if len(toolCalls) == 0 { // model produced answer directly
		res.Answer = content.String()
		res.IsComplete = true
		return res, nil
	}

	res.ToolCalls = toolCalls
	for _, tc := range toolCalls {
		name := tc.Function.Name
		tool := FindMCPToolByName(a.config.Tools, name)
		if tool == nil {
			logger.Error("Tool not found", zap.String("tool", name))
			continue
		}
		reporter.Send(NewProgressUpdate(schema.Stage_tool_execution_starting, fmt.Sprintf("Executing tool: %s", name), 2))
		params := parseToolArguments(tc.Function.Arguments)
		resultChan := tool.Handler(ctx, params)
		var acc strings.Builder
		for tr := range resultChan {
			if tr.Error != "" {
				logger.Error("Tool execution error", zap.String("tool", name), zap.String("error", tr.Error))
				res.ToolResults[name] = "Error: " + tr.Error
				break
			}
			reporter.Send(NewToolExecutionResult(name, tr))
			if len(tr.Sentences) > 0 {
				acc.WriteString(strings.Join(tr.Sentences, " "))
				acc.WriteByte(' ')
			}
		}
		res.ToolResults[name] = acc.String()
		res.ToolsUsed = append(res.ToolsUsed, name)
	}
	return res, nil
}

// promptTurn handles prompt-based tool selection & execution fallback
func (a *Agent) promptTurn(ctx context.Context, client llm.LLMClient, messages []llm.Message, turn int, reporter ProgressReporter, startTime int64) (*TurnResult, error) {
	res := &TurnResult{TurnNumber: turn, ToolsUsed: []string{}}
	selectionReq := ToolSelectionRequest{Query: messages[len(messages)-1].Content, MaxTools: 3}
	selected, err := a.SelectTools(ctx, selectionReq)
	if err != nil { // selection failed -> stream direct answer
		var answer strings.Builder
		err := client.GenerateInference(ctx, messages, func(chunk string) error {
			answer.WriteString(chunk)
			a.streamPartialAnswer(answer.String(), client, nil, startTime, reporter)
			return nil
		}, llm.WithTemperature(0.7), llm.WithMaxTokens(a.config.MaxTokens))
		if err != nil {
			return nil, fmt.Errorf("failed to generate answer: %w", err)
		}
		res.Answer = answer.String()
		res.IsComplete = true
		return res, nil
	}

	var collected []string
	for _, st := range selected {
		for tr := range a.ExecuteTool(ctx, st) {
			if tr.Error != "" {
				logger.Error("Tool execution error", zap.String("tool", st.Name), zap.String("error", tr.Error))
				continue
			}
			reporter.Send(NewToolExecutionResult(st.Name, tr))
			if len(tr.Sentences) > 0 {
				collected = append(collected, a.formatToolResult(st.Name, tr))
				res.ToolsUsed = append(res.ToolsUsed, st.Name)
			}
		}
	}
	if len(collected) == 0 {
		res.IsComplete = true
		res.Answer = "I wasn't able to use tools to help with your request. Let me provide a direct answer."
	}
	return res, nil
}

// streamFinalAnswer builds & streams the final answer when no prior turn produced it
func (a *Agent) streamFinalAnswer(ctx context.Context, client llm.LLMClient, conversation []llm.Message, reporter ProgressReporter, base *schema.AnswerChunk, startTime int64) {
	reporter.Send(NewProgressUpdate(schema.Stage_answer_generation_starting, "Generating final answer...", 3))
	prompt := a.buildFinalPrompt(conversation)
	conversation = append(conversation, llm.Message{Role: "user", Content: prompt})
	var buf strings.Builder
	_ = client.GenerateInference(ctx, conversation, func(chunk string) error {
		buf.WriteString(chunk)
		a.streamPartialAnswer(buf.String(), client, base.ToolsUsed, startTime, reporter)
		return nil
	}, llm.WithTemperature(0.7), llm.WithMaxTokens(a.config.MaxTokens))
	base.Answer = buf.String()
}

// streamPartialAnswer emits a non-final partial answer chunk
func (a *Agent) streamPartialAnswer(content string, client llm.LLMClient, toolsUsed []string, startTime int64, reporter ProgressReporter) {
	if reporter == nil {
		return
	}
	partial := &schema.AnswerChunk{Answer: content, ToolsUsed: toolsUsed, ModelUsed: client.GetModel(), ProcessingTime: getCurrentTimeMs() - startTime, Metadata: map[string]string{}, IsFinal: false}
	reporter.Send(NewAnswerChunk(partial))
}

// Helpers
func (a *Agent) initialConversation(req *schema.GenerateAnswerRequest) []llm.Message {
	msgs := []llm.Message{{Role: "user", Content: req.Question}}
	if req.Context != "" {
		msgs = append([]llm.Message{{Role: "system", Content: fmt.Sprintf("Additional context: %s", req.Context)}}, msgs...)
	}
	return msgs
}

func (a *Agent) pickClient(query string, toolResults []string) (llm.LLMClient, bool) {
	if a.shouldUseBigModel(query, toolResults) && a.config.BigModel != nil {
		return a.config.BigModel, true
	}
	return a.config.MiniModel, false
}

func (a *Agent) effectiveMaxTurns() int {
	if a.config.MaxTurns > 0 {
		return a.config.MaxTurns
	}
	return 5
}

// buildFinalPrompt kept simple for future extensibility
func (a *Agent) buildFinalPrompt(_ []llm.Message) string {
	return "Based on our conversation and any tool results above, please provide a comprehensive final answer to the original question."
}

// parseToolArguments converts tool call arguments into string map
func parseToolArguments(args map[string]any) map[string]string {
	res := map[string]string{}
	for k, v := range args {
		if s, ok := v.(string); ok {
			res[k] = s
			continue
		}
		if b, err := json.Marshal(v); err == nil {
			res[k] = string(b)
		}
	}
	return res
}
