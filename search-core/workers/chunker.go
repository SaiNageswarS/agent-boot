package workers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"

	"github.com/SaiNageswarS/gizmo/mupdf"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/pkoukk/tiktoken-go"
	"go.uber.org/zap"
)

const (
	maxTokens = 400
	overlap   = 50
)

type Chunker struct {
	// To load encoder only once across all chunking operations.
	tok *tiktoken.Tiktoken
}

func ProvideChunker() *Chunker {
	tok, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		logger.Error("Failed to get token encoder", zap.Error(err))
		return nil
	}
	return &Chunker{tok: tok}
}

func (c *Chunker) ChunkPDF(ctx context.Context, pdfFilePath string) ([]Chunk, error) {
	pdfText, err := mupdf.ExtractText(ctx, pdfFilePath)
	if err != nil {
		logger.Error("Failed to extract text", zap.String("filename", pdfFilePath), zap.Error(err))
		return nil, err
	}

	var out []Chunk
	tokens := c.tok.Encode(pdfText, nil, nil)
	for i := 0; i < len(tokens); i += maxTokens - overlap {
		end := min(i+maxTokens, len(tokens))
		sub := tokens[i:end]
		txt := c.tok.Decode(sub)
		id := sha1.Sum([]byte(pdfFilePath + ":" + txt))
		out = append(out, Chunk{
			ID:        hex.EncodeToString(id[:]),
			Text:      txt,
			Page:      -1, // TODO: Should we extract page by page and chunk by tokens?
			SourcePDF: pdfFilePath,
		})
		if end == len(tokens) {
			break
		}
	}

	return out, nil
}
