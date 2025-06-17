package prompts

import (
	"bytes"
	"context"
	"embed"
	"text/template"
	"time"

	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

const EmbeddingModel = "nomic-embed-text"
const EmbeddingDimensions = 768 // Dimension of the embedding vector

//go:embed templates/*
var templatesFS embed.FS

func GenerateTitle(ctx context.Context, client *api.Client, introDocSnippet string) <-chan async.Result[string] {
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

		stream := false
		request := &api.ChatRequest{
			Model:  "llama3:latest", // Using Haiku as the "mini" model
			Stream: &stream,
			Messages: []api.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
		}

		var ollamaResponse api.ChatResponse
		err = client.Chat(ctx, request, func(cr api.ChatResponse) error {
			ollamaResponse = cr
			return nil
		})

		if err != nil {
			logger.Error("Failed to generate title", zap.Error(err))
			return "", err
		}

		return ollamaResponse.Message.Content, nil
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
