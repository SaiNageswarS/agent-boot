package prompts

import (
	"bytes"
	"context"
	"embed"
	"text/template"
	"time"

	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

const EmbeddingModel = "nomic-embed-text"
const EmbeddingDimensions = 768 // Dimension of the embedding vector

//go:embed templates/*
var templatesFS embed.FS

func GenerateTitle(ctx context.Context, client *llm.AnthropicClient, introDocSnippet string) <-chan async.Result[string] {
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

		return async.Await(client.GenerateInference(ctx, &request))
	})
}

func EmbedOnce(ctx context.Context, cli *api.Client, text string) ([]float32, error) {
	req := &api.EmbeddingRequest{
		Model:     EmbeddingModel,
		Prompt:    text,
		KeepAlive: &api.Duration{Duration: 60 * time.Minute}, // keep connection alive for reuse
	}
	resp, err := cli.Embeddings(ctx, req) // blocking, non-streaming
	if err != nil {
		return nil, err
	}

	emb64 := resp.Embedding // []float64
	emb32 := make([]float32, len(emb64))
	for i, v := range emb64 {
		emb32[i] = float32(v)
	}
	return emb32, nil
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
