package workers

import (
	"context"

	"github.com/SaiNageswarS/gizmo/mupdf"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/pkoukk/tiktoken-go"
	"go.uber.org/zap"
)

const (
	maxTokens = 1200
	overlap   = 150
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
	blocks, err := mupdf.ExtractStructuredText(ctx, pdfFilePath)
	if err != nil {
		logger.Error("Failed to extract structured text from PDF", zap.Error(err))
		return nil, err
	}

	logger.Info("Extracted blocks from PDF", zap.Int("num_blocks", len(blocks)))
	var out []Chunk

	for _, block := range blocks {
		if block.Text == "" {
			continue // Skip empty blocks
		}

		// Tokenize the text to check if it exceeds max tokens
		tokens := c.tok.Encode(block.Text, nil, nil)
		if len(tokens) <= maxTokens {
			chunk := Chunk{
				Header:    block.HeaderHierarchy,
				Passage:   block.Text,
				Page:      block.PageNumber,
				SourcePdf: pdfFilePath,
			}
			out = append(out, chunk)
			continue
		}

		// If it exceeds max tokens, split the text into smaller chunks
		splitChunks := c.splitTextIntoChunks(block.Text, block.HeaderHierarchy, block.PageNumber, pdfFilePath)
		out = append(out, splitChunks...)
	}

	return out, nil
}

func (c *Chunker) splitTextIntoChunks(text, header string, page int, sourcePdf string) []Chunk {
	tokens := c.tok.Encode(text, nil, nil)
	chunks := make([]Chunk, 0)

	for i := 0; i < len(tokens); i += maxTokens - overlap {
		end := min(i+maxTokens, len(tokens))

		chunkText := c.tok.Decode(tokens[i:end])
		chunk := Chunk{
			Header:    header,
			Page:      page,
			Passage:   chunkText,
			SourcePdf: sourcePdf,
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}
