package prompts

import (
	"context"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-collection-boot/async"
)

func GenerateAnswer(ctx context.Context, client llm.LLMClient, modelVersion, agentCapability, userInput, searchResultJson string) <-chan async.Result[string] {
	return async.Go(func() (string, error) {
		systemPrompt, err := loadPrompt("templates/generate_answer_system.md", map[string]string{
			"AGENT_CAPABILITY": agentCapability,
		})
		if err != nil {
			return "", err
		}

		userPrompt, err := loadPrompt("templates/generate_answer_user.md", map[string]string{
			"USER_INPUT":         userInput,
			"SEARCH_RESULT_JSON": searchResultJson,
		})
		if err != nil {
			return "", err
		}

		messages := []llm.Message{
			{
				Role:    "user",
				Content: userPrompt,
			},
		}

		var response string
		err = client.GenerateInference(ctx, messages, func(chunk string) error {
			response += chunk
			return nil
		}, llm.WithLLMModel(modelVersion),
			llm.WithMaxTokens(8000),
			llm.WithTemperature(0.5),
			llm.WithSystemPrompt(systemPrompt),
		)

		return response, err
	})
}
