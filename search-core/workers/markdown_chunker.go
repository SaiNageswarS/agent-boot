package workers

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.uber.org/zap"
	"golang.org/x/crypto/blake2s"
)

const maxSectionDepth = 4 // Maximum depth of section hierarchy to chunk

type MarkdownChunker struct{}

func ProvideMarkdownChunker() *MarkdownChunker {
	return &MarkdownChunker{}
}

// Chunks Markdown and returns paths of two files: // 1) section-level chunks, and 2) Window-Level chunks.
func (c *MarkdownChunker) ChunkMarkdownSections(ctx context.Context, fileName string, markdown []byte) ([]Chunk, error) {
	sections, err := parseMarkdownSections(markdown)
	if err != nil {
		logger.Error("Failed to parse markdown sections", zap.Error(err))
		return nil, err
	}

	var out []Chunk
	for _, sec := range sections {
		secHash := hash(fileName + strings.Join(sec.path, "|"))

		secChunk := Chunk{
			ChunkID:     fmt.Sprintf("%s-%s", fileName, secHash),
			SectionPath: sec.path,
			TokenStart:  0, // with respect to section.
			CreatedAt:   time.Now().Format(time.RFC3339),
			PHIRemoved:  false,
			SourceURI:   fileName,
			Body:        sec.body,
		}

		out = append(out, secChunk)
	}

	return out, nil
}

func hash(s string) string {
	h, _ := blake2s.New256(nil)
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:10]
}

func parseMarkdownSections(md []byte) ([]markdownSection, error) {
	var out []markdownSection

	reader := text.NewReader(md)
	root := goldmark.DefaultParser().Parse(reader)

	var currentPath []string
	var buf bytes.Buffer

	flush := func() {
		if len(currentPath) > 0 && buf.Len() > 0 {
			// copy path
			dst := append([]string(nil), currentPath...)
			out = append(out, markdownSection{path: dst, body: buf.String()})
			buf.Reset()
		}
	}

	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering {
			flush() // finish previous
			headingText := string(h.Text(md))
			// keep path up to this heading level
			level := h.Level
			if level <= maxSectionDepth { // we cap hierarchy depth to 3
				if len(currentPath) >= level {
					currentPath = currentPath[:level-1]
				}
				currentPath = append(currentPath, headingText)
			}
			// skip printing heading itself into body; body starts after the heading node
			return ast.WalkContinue, nil
		}
		if entering {
			segment := n.Text(md)
			if len(segment) > 0 {
				buf.Write(segment)
			}
			if n.Type() == ast.TypeBlock {
				buf.WriteByte('\n')
			}
		}
		return ast.WalkContinue, nil
	})
	flush()
	if len(out) == 0 {
		return nil, errors.New("no headings found")
	}
	return out, nil
}

type markdownSection struct {
	path []string // section path
	body string   // section body
}
