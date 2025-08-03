package agent

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/core/llm"
)

func (a *Agent) GenerateAnswer(ctx context.Context, client llm.LLMClient, modelName, prompt string) (string, error) {
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
		return "", err
	}

	response := responseContent.String()
	return response, nil
}
