package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SaiNageswarS/agent-boot/example/db"
	"github.com/SaiNageswarS/agent-boot/example/prompts"
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
	md, err := getBytes(s.az.DownloadFile(ctx, tenant, markdownFile))
	if err != nil {
		return nil, errors.New("failed to download markdown file: " + err.Error())
	}

	// parse sections in markdown
	sections, err := parseMarkdownSections(ctx, md, minSectionBytes)
	if err != nil {
		return nil, err
	}

	var (
		allChunks         []db.ChunkModel // for debugging
		sectionChunkPaths []string
	)

	linq.Pipe2(
		linq.FromSlice(ctx, sections),

		// TRANSFORM each markdownSection → db.ChunkModel (with LLM call)
		linq.Select(func(sec markdownSection) db.ChunkModel {
			secHash, _ := odm.HashedKey(sec.body)

			// generate a concise title – any error handled below
			titleBodyInputLen := min(len(sec.body), maxTitleInputBytes)
			title, _ := async.Await(prompts.GenerateSectionTitle(
				ctx, s.ollama, sourceUri,
				sec.path[len(sec.path)-1], sec.body[:titleBodyInputLen], s.ccfg.TitleGenModel,
			))
			if title == "" || len(title) > 100 {
				title = sec.path[len(sec.path)-1]
			}

			return db.ChunkModel{
				ChunkID:      secHash,
				SectionPath:  strings.Join(sec.path, " | "),
				SectionIndex: len(allChunks) + 1, // running index
				SectionID:    secHash,
				Title:        title,
				SourceURI:    sourceUri,
				Sentences:    []string{sec.body},
			}
		}),

		// SINK: perform side-effects as soon as each chunk arrives
		linq.ForEach(func(chunk db.ChunkModel) {
			allChunks = append(allChunks, chunk)

			chunkPath := fmt.Sprintf("%s/%s.chunk.json", sectionsOutputPath, chunk.ChunkID)
			writeToStorage(ctx, s.az, tenant, chunkPath, chunk) // persist immediately
			sectionChunkPaths = append(sectionChunkPaths, chunkPath)

			logger.Info("Extracted section chunk",
				zap.String("chunkID", chunk.ChunkID),
				zap.String("title", chunk.Title))
		}),
	)

	return sectionChunkPaths, nil
}

func parseMarkdownSections(ctx context.Context, md []byte, minBytes int) ([]markdownSection, error) {
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
			combinedPath, err := linq.Pipe2(
				linq.FromSlice(ctx, prev.path),
				linq.Distinct(func(a string) string { return a }), // Ensure unique paths
				linq.ToSlice[string](),
			)

			if err != nil {
				logger.Error("Failed to combine section paths", zap.Error(err))
			} else {
				prev.path = combinedPath
			}
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
