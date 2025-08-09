package agent

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
)

func (a *Agent) GenerateAnswer(ctx context.Context, client llm.LLMClient, prompt string) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	var responseContent strings.Builder
	err := client.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(a.getMaxTokens()),
	)

	if err != nil {
		return "", err
	}

	response := responseContent.String()
	return response, nil
}

// GenerateAnswerWithStreaming generates an answer with real-time streaming to the reporter
func (a *Agent) GenerateAnswerWithStreaming(ctx context.Context, client llm.LLMClient, prompt string, reporter ProgressReporter, toolsUsed []string) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	var responseContent strings.Builder
	startTime := getCurrentTimeMs()

	err := client.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			// Stream each chunk as it's generated
			partialResponse := &schema.AnswerChunk{
				Answer:         responseContent.String(),
				ToolsUsed:      toolsUsed,
				ModelUsed:      client.GetModel(),
				ProcessingTime: getCurrentTimeMs() - startTime,
				Metadata:       make(map[string]string),
				IsFinal:        false, // Not final until complete
			}
			reporter.Send(NewAnswerChunk(partialResponse))
			return nil
		},
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(a.getMaxTokens()),
	)

	if err != nil {
		return "", err
	}

	response := responseContent.String()
	return response, nil
}
