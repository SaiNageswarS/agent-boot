package prompts

import (
	"context"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-collection-boot/async"
)

func GenerateAnswer(ctx context.Context, client *llm.AnthropicClient, agentCapability, userInput, searchResultJson string) <-chan async.Result[string] {
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

		request := llm.AnthropicRequest{
			Model:       "claude-3-5-sonnet-20241022",
			MaxTokens:   8000,
			System:      systemPrompt,
			Temperature: 0.5,
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: userPrompt,
				},
			},
		}

		return async.Await(client.GenerateInference(ctx, &request))
	})
}
