package prompts

import (
	"context"
	"strings"

	"github.com/SaiNageswarS/agent-boot/core/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"go.uber.org/zap"
)

type ExtractSearchQueriesResponse struct {
	Reasoning     string   `json:"reasoning"`
	SearchQueries []string `json:"search_queries"`
}

func ExtractSearchQueries(ctx context.Context, client llm.LLMClient, modelVersion, userInput, agentCapability string) <-chan async.Result[*ExtractSearchQueriesResponse] {
	return async.Go(func() (*ExtractSearchQueriesResponse, error) {
		systemPrompt, err := loadPrompt("templates/extract_agent_search_query_system.md", map[string]string{
			"AGENT_CAPABILITY": agentCapability,
		})
		if err != nil {
			logger.Error("Failed to load system prompt", zap.Error(err))
			return nil, err
		}

		userPrompt, err := loadPrompt("templates/extract_agent_search_query_user.md", map[string]string{
			"USER_INPUT": userInput,
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
		callback := func(chunk string) error {
			response += chunk
			return nil
		}

		err = client.GenerateInference(ctx, messages, callback,
			llm.WithLLMModel(modelVersion),
			llm.WithMaxTokens(4000),
			llm.WithTemperature(0.3),
			llm.WithSystemPrompt(systemPrompt))

		if err != nil {
			logger.Error("Failed to generate inference", zap.Error(err))
			return nil, err
		}

		logger.Info("ExtractAgentInput response", zap.String("response", response))

		out := parseResponse(ctx, response)
		return out, nil
	})
}

func parseResponse(ctx context.Context, responseText string) *ExtractSearchQueriesResponse {
	lines := strings.Split(strings.TrimSpace(responseText), "\n")
	out := &ExtractSearchQueriesResponse{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if after0, ok0 := strings.CutPrefix(line, "REASONING:"); ok0 {
			out.Reasoning = strings.TrimSpace(after0)
		} else if after1, ok1 := strings.CutPrefix(line, "QUERIES:"); ok1 {
			queryStr := strings.TrimSpace(after1)

			if queryStr != "" {
				searchQueries := strings.Split(queryStr, "|")
				searchQueries, err := linq.Pipe3(
					linq.FromSlice(ctx, searchQueries),

					linq.Select(func(q string) string {
						return strings.TrimSpace(q)
					}),

					linq.Where(func(q string) bool {
						return q != ""
					}),

					linq.ToSlice[string](),
				)

				if err != nil {
					logger.Error("Failed to parse search queries", zap.Error(err))
					out.SearchQueries = []string{queryStr}
				} else {
					out.SearchQueries = searchQueries
				}
			}
		}
	}

	return out
}
