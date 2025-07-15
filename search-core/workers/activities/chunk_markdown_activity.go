package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.uber.org/zap"
)

const minSectionBytes = 4000 // Minimum bytes for a section to be considered valid

// ChunkMarkdown processes a markdown file, chunks it into sections, and uploads the chunks to Azure Blob Storage.
// It returns the paths to the uploaded chunks JSON file. Each chunk is uploaded as a separate file in the specified {tenant}/{markdownFile} directory.
func (s *Activities) ChunkMarkdown(ctx context.Context, tenant, sourceUri, markdownFile, sectionsOutputPath string) ([]string, error) {
	markDownBytes, err := getBytes(s.az.DownloadFile(ctx, tenant, markdownFile))
	if err != nil {
		return []string{}, errors.New("failed to download PDF file: " + err.Error())
	}

	// Chunk the Markdown file
	chunks, err := chunkMarkdownSections(ctx, s.ollama, sourceUri, s.ccfg.TitleGenModel, markDownBytes)
	if err != nil {
		return []string{}, errors.New("failed to chunk PDF file: " + err.Error())
	}

	// write combined sections to a single file for debugging purposes
	combinedSectionsFile := fmt.Sprintf("%s/combined_sections.json", sectionsOutputPath)
	writeToStorage(ctx, s.az, tenant, combinedSectionsFile, chunks)

	var sectionChunkPaths []string
	for _, chunk := range chunks {
		chunkPath := fmt.Sprintf("%s/%s.chunk.json", sectionsOutputPath, chunk.ChunkID)
		writeToStorage(ctx, s.az, tenant, chunkPath, chunk)
		sectionChunkPaths = append(sectionChunkPaths, chunkPath)
	}

	return sectionChunkPaths, nil
}

func chunkMarkdownSections(ctx context.Context, ollama *llm.OllamaLLMClient, sourceUri, titleGenModel string, markdown []byte) ([]db.ChunkModel, error) {
	sections, err := parseMarkdownSections(markdown, minSectionBytes)
	if err != nil {
		logger.Error("Failed to parse markdown sections", zap.Error(err))
		return nil, err
	}

	var out []db.ChunkModel
	for idx, sec := range sections {
		secHash, _ := odm.HashedKey(sec.body)

		// Generate a concise title for the section using LLM
		title, err := async.Await(prompts.GenerateSectionTitle(ctx, ollama, sourceUri, sec.path[len(sec.path)-1], sec.body, titleGenModel))
		if err != nil || len(title) > 100 {
			logger.Error("Failed to generate section title", zap.Error(err))
			title = sec.path[len(sec.path)-1] // fallback to last path segment
		}

		secChunk := db.ChunkModel{
			ChunkID:      secHash,
			SectionPath:  strings.Join(sec.path, " | "),
			SectionIndex: idx + 1, // 1-based index
			Title:        title,
			SourceURI:    sourceUri,
			Sentences:    []string{sec.body},
		}

		if idx > 0 {
			// Set the previous chunk ID for all but the first section
			secChunk.PrevChunkID = out[idx-1].ChunkID
			out[idx-1].NextChunkID = secChunk.ChunkID // link previous chunk to current
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
			// Append the current section's path to the previous section's path
			prev.path = append(prev.path, s.path...)
			prev.path = linq.Distinct(prev.path, func(a string) string { return a }) // Ensure unique paths
		} else {
			merged = append(merged, s)
		}
	}
	return merged, nil
}

type markdownSection struct {
	path []string // section path
	body string   // section body
}
