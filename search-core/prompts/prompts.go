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

func GenerateTitle(client *llm.AnthropicClient, introDocSnippet string) (string, error) {
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

	request := llm.AnthropicRequest{
		Model:       "claude-3-5-haiku-20241022", // Using Haiku as the "mini" model
		MaxTokens:   4000,
		System:      systemPrompt,
		Temperature: 0.2, // For stable outputs
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
