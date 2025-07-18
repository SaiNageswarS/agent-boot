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

const minSectionBytes = 4000    // Minimum bytes for a section to be considered valid
const maxTitleInputBytes = 2500 // Maximum bytes for title generation input

// ChunkMarkdown processes a markdown file, chunks it into sections, and uploads the chunks to Azure Blob Storage.
// It returns the paths to the uploaded chunks JSON file. Each chunk is uploaded as a separate file in the specified {tenant}/{markdownFile} directory.
func (s *Activities) ChunkMarkdown(ctx context.Context, tenant, sourceUri, markdownFile, sectionsOutputPath string) ([]string, error) {
	// download markdown bytes
	markDownBytes, err := getBytes(s.az.DownloadFile(ctx, tenant, markdownFile))
	if err != nil {
		return nil, errors.New("failed to download markdown file: " + err.Error())
	}

	// start the chunk generator
	chunksCh, errCh := chunkMarkdownSections(ctx, s.ollama, sourceUri, s.ccfg.TitleGenModel, markDownBytes)

	// for debugging: collect all chunks into one big slice and write combined
	var allChunks []db.ChunkModel

	// stream out each chunk as soon as it's ready
	var sectionChunkPaths []string
	for chunk := range chunksCh {
		allChunks = append(allChunks, chunk)

		// write each chunk out immediately
		chunkPath := fmt.Sprintf("%s/%s.chunk.json", sectionsOutputPath, chunk.ChunkID)
		writeToStorage(ctx, s.az, tenant, chunkPath, chunk)
		sectionChunkPaths = append(sectionChunkPaths, chunkPath)
	}

	// check for any generation error
	if genErr := <-errCh; genErr != nil {
		return nil, fmt.Errorf("failed to chunk markdown sections: %w", genErr)
	}

	// write combined sections for debugging
	combinedSectionsFile := fmt.Sprintf("%s/combined_sections.json", sectionsOutputPath)
	writeToStorage(ctx, s.az, tenant, combinedSectionsFile, allChunks)

	return sectionChunkPaths, nil
}

func chunkMarkdownSections(
	ctx context.Context,
	ollama *llm.OllamaLLMClient,
	sourceUri, titleGenModel string,
	markdown []byte,
) (<-chan db.ChunkModel, <-chan error) {
	chunksCh := make(chan db.ChunkModel)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunksCh)
		defer close(errCh)

		sections, err := parseMarkdownSections(markdown, minSectionBytes)
		if err != nil {
			logger.Error("Failed to parse markdown sections", zap.Error(err))
			errCh <- err
			return
		}

		for idx, sec := range sections {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			secHash, _ := odm.HashedKey(sec.body)

			// Generate a concise title
			titleBodyInputLen := min(len(sec.body), maxTitleInputBytes)
			title, titleErr := async.Await(prompts.GenerateSectionTitle(
				ctx, ollama, sourceUri,
				sec.path[len(sec.path)-1], sec.body[0:titleBodyInputLen], titleGenModel,
			))
			if titleErr != nil || len(title) == 0 || len(title) > 100 {
				logger.Error("Failed to generate section title", zap.Error(titleErr))
				title = sec.path[len(sec.path)-1]
			}

			chunk := db.ChunkModel{
				ChunkID:      secHash,
				SectionPath:  strings.Join(sec.path, " | "),
				SectionIndex: idx + 1,
				SectionID:    secHash,
				WindowIndex:  0, // Populated later in window activity
				Title:        title,
				SourceURI:    sourceUri,
				Sentences:    []string{sec.body},
				PrevChunkID:  "", // Populated later in window activity
				NextChunkID:  "", // Populated later in window activity
			}

			chunksCh <- chunk
			logger.Info("Extracted section chunk", zap.String("chunkID", chunk.ChunkID), zap.String("title", chunk.Title))
		}

		errCh <- nil
	}()

	return chunksCh, errCh
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
