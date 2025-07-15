package prompts

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.uber.org/zap"
)

func GenerateSectionTitle(ctx context.Context, client *llm.OllamaLLMClient, docTitle, originalSectionHeader, sectionSnippet, model string) <-chan async.Result[string] {
	return async.Go(func() (string, error) {
		systemPrompt, err := loadPrompt("templates/generate_title_system.md", map[string]string{})
		if err != nil {
			logger.Error("Failed to load system prompt", zap.Error(err))
			return "", err
		}

		userPrompt, err := loadPrompt("templates/generate_title_user.md", map[string]string{
			"DOCUMENT_TITLE":   docTitle,
			"DOCUMENT_SNIPPET": sectionSnippet,
			"ORIGINAL_HEADING": originalSectionHeader,
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
			llm.WithLLMModel(model),
			llm.WithMaxTokens(4000),
			llm.WithTemperature(0.2),
			llm.WithSystemPrompt(systemPrompt),
		)

		if err != nil {
			return "", err
		}

		// Extract title from the TITLE block
		titleLines := extractSection(response, "TITLE:")
		title := strings.TrimSpace(strings.Join(titleLines, " "))

		return title, err
	})
}
