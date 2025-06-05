package workers

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.uber.org/zap"
	"golang.org/x/crypto/blake2s"
)

const maxSectionDepth = 4 // Maximum depth of section hierarchy to chunk

type MarkdownChunker struct {
	llmClient *llm.AnthropicClient
}

func ProvideMarkdownChunker(llmClient *llm.AnthropicClient) *MarkdownChunker {
	return &MarkdownChunker{
		llmClient: llmClient,
	}
}

// Chunks Markdown by sections.
func (c *MarkdownChunker) ChunkMarkdownSections(ctx context.Context, fileName string, markdown []byte) ([]Chunk, error) {
	titleResultChan := c.generateTitle(ctx, markdown)

	sections, err := parseMarkdownSections(markdown)
	if err != nil {
		logger.Error("Failed to parse markdown sections", zap.Error(err))
		return nil, err
	}

	// Wait for title generation
	title, err := odm.Await(titleResultChan)
	if err != nil {
		logger.Error("Failed to generate title", zap.Error(err))
		return nil, err
	}

	var out []Chunk
	for idx, sec := range sections {
		secHash := hash(fileName + strings.Join(sec.path, "|"))

		secChunk := Chunk{
			ChunkID:      fmt.Sprintf("%s-%s", fileName, secHash),
			SectionPath:  append([]string{title}, sec.path...),
			SectionIndex: idx + 1, // 1-based index
			CreatedAt:    time.Now().Format(time.RFC3339),
			PHIRemoved:   false,
			SourceURI:    fileName,
			Body:         sec.body,
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

func (c *MarkdownChunker) generateTitle(ctx context.Context, md []byte) <-chan odm.Result[string] {
	ch := make(chan odm.Result[string], 1)

	go func() {
		defer close(ch)

		// chunk md
		maxBytes := min(2500, len(md))
		introDocSnippet := string(md[0:maxBytes])

		title, err := prompts.GenerateTitle(c.llmClient, introDocSnippet)
		if err != nil {
			logger.Error("Failed to generate title", zap.Error(err))
			ch <- odm.Result[string]{Err: err}
			return
		}

		ch <- odm.Result[string]{Data: title}
	}()

	return ch
}

type markdownSection struct {
	path []string // section path
	body string   // section body
}
