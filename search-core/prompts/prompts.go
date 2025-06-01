package prompts

import (
	"bytes"
	"embed"
	"text/template"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

//go:embed templates/*
var templatesFS embed.FS

/*
- Deep Extract Prompt extracts passages from a document and returns a JSON object with the extracted passages.
- It runs at a low temperature to ensure the exact content is extracted from the source.
- The prompt is designed to extract diverse topics from the document.
*/
func DeepExtractPassages(client *llm.AnthropicClient, doc string) (string, error) {
	systemPrompt, err := loadPrompt("templates/deep_extract_system.tmpl", map[string]string{})
	if err != nil {
		logger.Error("Failed to load system prompt", zap.Error(err))
		return "", err
	}

	userPrompt, err := loadPrompt("templates/deep_extract_user.tmpl", map[string]string{"PDF_TEXT_EXTRACT": doc})
	if err != nil {
		logger.Error("Failed to load user prompt", zap.Error(err))
		return "", err
	}

	request := llm.AnthropicRequest{
		Model:       "claude-3-5-haiku-20241022", // Using Haiku as the "mini" model
		MaxTokens:   4000,
		System:      systemPrompt,
		Temperature: 0.15, // Low temperature for exact extraction
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	return client.GenerateInference(&request)
}

func loadPrompt(templatePath string, data interface{}) (string, error) {
	tmpl, err := template.ParseFS(templatesFS, templatePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
