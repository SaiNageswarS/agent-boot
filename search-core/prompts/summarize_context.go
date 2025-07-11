package prompts

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-collection-boot/async"
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
			llm.WithTemperature(0.5),
			llm.WithSystemPrompt(systemPrompt),
		)

		if err != nil {
			return nil, err
		}

		// Extract sentences in SUMMARY block
		lines := strings.Split(response, "\n")
		var summaryLines []string
		inSummary := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SUMMARY:") {
				inSummary = true
				continue
			}
			if inSummary {
				if line == "" || strings.HasPrefix(line, "THOUGHTS:") {
					break // End of SUMMARY block
				}
				summaryLines = append(summaryLines, line)
			}
		}

		return summaryLines, nil
	})
}
