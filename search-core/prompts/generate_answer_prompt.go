package prompts

import (
	"context"
	"regexp"
	"strings"

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

		return formatThinkToMd(response), err
	})
}

func formatThinkToMd(md string) string {
	if !strings.Contains(md, "<think>") {
		return md // fast-path: no tag to process
	}

	re := regexp.MustCompile(`(?s)<think>(.*?)</think>`)

	return re.ReplaceAllStringFunc(md, func(match string) string {
		inner := re.FindStringSubmatch(match)[1]
		inner = strings.TrimSpace(inner)

		// Prefix each line with "> "
		lines := strings.Split(inner, "\n")
		for i, line := range lines {
			lines[i] = "> " + line
		}

		return "\n> **Chain-of-thought**\n>\n" +
			strings.Join(lines, "\n") + "\n"
	})
}
