package workers

import (
	"context"
	"errors"
	"strings"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/async"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.uber.org/zap"
)

type MarkdownChunker struct {
	llmClient *llm.AnthropicClient
}

func ProvideMarkdownChunker(llmClient *llm.AnthropicClient) *MarkdownChunker {
	return &MarkdownChunker{
		llmClient: llmClient,
	}
}

// Chunks Markdown by sections.
func (c *MarkdownChunker) ChunkMarkdownSections(ctx context.Context, sourceUri string, markdown []byte) ([]db.ChunkModel, error) {
	maxIntroBytes := min(2500, len(markdown)) // Limit intro snippet to 1000 bytes or less
	titleResultChan := prompts.GenerateTitle(ctx, c.llmClient, string(markdown[0:maxIntroBytes]))

	sections, err := parseMarkdownSections(markdown, 2000)
	if err != nil {
		logger.Error("Failed to parse markdown sections", zap.Error(err))
		return nil, err
	}

	// Wait for title generation
	title, err := async.Await(titleResultChan)
	logger.Info("Title generated", zap.String("title", title))

	if err != nil {
		logger.Error("Failed to generate title", zap.Error(err))
		return nil, err
	}

	var out []db.ChunkModel
	for idx, sec := range sections {
		secHash, _ := odm.HashedKey(sec.body)
		secPath := getSectionPath(title, sec.path)

		secChunk := db.ChunkModel{
			ChunkID:      secHash,
			SectionPath:  secPath,
			SectionIndex: idx + 1, // 1-based index
			PHIRemoved:   false,
			SourceURI:    sourceUri,
			Body:         sec.body,
		}

		out = append(out, secChunk)
	}

	logger.Info("Markdown sections chunked", zap.Int("sectionCount", len(out)), zap.String("fileName", sourceUri))
	return out, nil
}

func parseMarkdownSections(md []byte, minBytes int) ([]markdownSection, error) {
	reader := text.NewReader(md)
	root := goldmark.DefaultParser().Parse(reader)

	type head struct {
		start   int // byte offset of heading line start
		lineEnd int // byte offset just *after* the end-of-line
		level   int
		title   string
	}
	var heads []head

	// ── collect all headings with byte offsets ────────────────────────────
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok {
			seg := h.Lines().At(0) // first (and only) line
			// seg.Start .. seg.Stop covers the characters of the line
			lineEnd := seg.Stop
			// skip trailing CR/LF so body starts at the next content byte
			for lineEnd < len(md) && (md[lineEnd] == '\n' || md[lineEnd] == '\r') {
				lineEnd++
			}
			heads = append(heads, head{
				start:   seg.Start,
				lineEnd: lineEnd,
				level:   h.Level,
				title:   strings.TrimSpace(string(h.Text(md))),
			})
		}
		return ast.WalkContinue, nil
	})
	if len(heads) == 0 {
		return nil, errors.New("no headings found")
	}

	// ── slice raw markdown into sections (body = after heading) ───────────
	var sections []markdownSection
	var path []string
	for i, h := range heads {
		// update hierarchy
		if len(path) >= h.level {
			path = path[:h.level-1]
		}
		path = append(path, h.title)

		start := h.lineEnd // <─ body starts *after* heading
		end := len(md)
		if i+1 < len(heads) {
			end = heads[i+1].start
		}

		sections = append(sections, markdownSection{
			path: append([]string(nil), path...), // copy
			body: string(md[start:end]),
		})
	}

	// ── merge small chunks ────────────────────────────────────────────────
	if minBytes <= 0 {
		return sections, nil
	}
	var merged []markdownSection
	for _, s := range sections {
		if len(s.body) < minBytes && len(merged) > 0 {
			prev := &merged[len(merged)-1]
			prev.body += "\n\n" + s.body
		} else {
			merged = append(merged, s)
		}
	}
	return merged, nil
}

func getSectionPath(title string, sectionHeaders []string) string {
	if len(sectionHeaders) == 0 {
		return ""
	}

	out := "# " + title + "\n"
	for i, header := range sectionHeaders {
		out = out + strings.Repeat("#", i+2) + " " + header + "\n"
	}

	// Join section headers with " ##" to create the path
	return out
}

type markdownSection struct {
	path []string // section path
	body string   // section body
}
