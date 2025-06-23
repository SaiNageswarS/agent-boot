package prompts

import (
	"context"
	"encoding/json"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.uber.org/zap"
)

type ExtractSearchQueriesResponse struct {
	Relevant      bool     `json:"relevant"`
	Reasoning     string   `json:"reasoning"`
	SearchQueries []string `json:"search_queries"`
}

func ExtractSearchQueries(ctx context.Context, client *llm.AnthropicClient, userInput, agentCapability string) <-chan async.Result[*ExtractSearchQueriesResponse] {
	return async.Go(func() (*ExtractSearchQueriesResponse, error) {
		systemPrompt, err := loadPrompt("templates/extract_agent_search_query_system.md", map[string]string{})
		if err != nil {
			logger.Error("Failed to load system prompt", zap.Error(err))
			return nil, err
		}

		userPrompt, err := loadPrompt("templates/extract_agent_search_query_user.md", map[string]string{
			"USER_INPUT":       userInput,
			"AGENT_CAPABILITY": agentCapability,
		})
		if err != nil {
			return nil, err
		}

		request := llm.AnthropicRequest{
			Model:       "claude-3-5-sonnet-20241022",
			MaxTokens:   4000,
			System:      systemPrompt,
			Temperature: 0.3,
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: userPrompt,
				},
			},
		}

		response, err := async.Await(client.GenerateInference(ctx, &request))
		if err != nil {
			logger.Error("Failed to generate inference", zap.Error(err))
			return nil, err
		}

		logger.Info("ExtractAgentInput response", zap.String("response", response))

		out := &ExtractSearchQueriesResponse{}
		json.Unmarshal([]byte(response), out)

		return out, nil
	})

}
