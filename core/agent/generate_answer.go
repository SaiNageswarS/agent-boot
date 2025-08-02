package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/core/llm"
)

func (a *Agent) GenerateAnswer(ctx context.Context, client llm.LLMClient, modelName, prompt string) (string, error) {
	a.reportProgress(NewAnswerGenerationEvent(
		"generation_starting",
		fmt.Sprintf("Starting answer generation using %s", modelName),
		&AnswerGenerationProgress{
			ModelUsed:    modelName,
			PromptLength: len(prompt),
			Status:       "starting",
		},
	))

	startTime := time.Now()
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
		llm.WithLLMModel(modelName),
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(a.getMaxTokens()),
	)

	if err != nil {
		a.reportProgress(NewErrorEvent(
			"answer_generation",
			"Answer generation failed",
			err.Error(),
		))
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	response := responseContent.String()

	duration := time.Since(startTime)
	a.reportProgress(NewCompletionEvent(
		"completed",
		"Answer generation process completed",
		&CompletionProgress{
			TotalDuration: time.Duration(duration) * time.Millisecond,
			ModelUsed:     modelName,
			AnswerLength:  len(response),
			Success:       true,
		},
	))
	return response, nil
}
