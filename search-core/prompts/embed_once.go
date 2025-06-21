package prompts

import (
	"context"
	"time"

	"github.com/ollama/ollama/api"
)

const EmbeddingModel = "nomic-embed-text"
const EmbeddingDimensions = 768 // Dimension of the embedding vector

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
