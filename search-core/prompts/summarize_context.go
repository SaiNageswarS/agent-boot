package prompts

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.uber.org/zap"
)

func SummarizeContext(ctx context.Context, client llm.LLMClient, modelVersion, userQuery string, sentences []string) <-chan async.Result[[]string] {
	return async.Go(func() ([]string, error) {
		systemPrompt, err := loadPrompt("templates/summarize_context_system.md", map[string]string{})
		if err != nil {
			return nil, err
		}

		userPrompt, err := loadPrompt("templates/summarize_context_user.md", map[string]string{
			"USER_QUERY":   userQuery,
			"SECTION_TEXT": strings.Join(sentences, "\n"),
		})
		if err != nil {
			return nil, err
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
			llm.WithTemperature(0.2),
			llm.WithSystemPrompt(systemPrompt),
		)

		if err != nil {
			return nil, err
		}

		logger.Info("Context Summary", zap.String("response", response))
		// Extract sentences in SUMMARY block
		summaryLines := extractSection(response, "SUMMARY:")
		return summaryLines, nil
	})
}
