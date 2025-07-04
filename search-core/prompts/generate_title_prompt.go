package prompts

import (
	"context"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.uber.org/zap"
)

func GenerateTitle(ctx context.Context, client llm.LLMClient, introDocSnippet string) <-chan async.Result[string] {
	return async.Go(func() (string, error) {
		systemPrompt, err := loadPrompt("templates/generate_title_system.md", map[string]string{})
		if err != nil {
			logger.Error("Failed to load system prompt", zap.Error(err))
			return "", err
		}

		userPrompt, err := loadPrompt("templates/generate_title_user.md", map[string]string{
			"DOCUMENT_SNIPPET": introDocSnippet,
		})
		if err != nil {
			logger.Error("Failed to load user prompt", zap.Error(err))
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
		},
			llm.WithLLMModel("claude-3-5-haiku-20241022"),
			llm.WithMaxTokens(4000),
			llm.WithTemperature(0.2),
			llm.WithSystemPrompt(systemPrompt),
		)

		return response, err
	})
}
