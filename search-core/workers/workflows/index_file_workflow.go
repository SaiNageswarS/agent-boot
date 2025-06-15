package workflows

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/search-core/workers/activities"
	"go.temporal.io/sdk/workflow"
)

func IndexFileWorkflow(ctx workflow.Context, state IndexerWorkflowState) (IndexerWorkflowState, error) {
	pyActivityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 20,
		TaskQueue:           "searchCorePySideCar",
	}
	pyCtx := workflow.WithActivityOptions(ctx, pyActivityOpts)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	if state.MarkdownFile == "" && state.PdfFile != "" {
		// running in pySideCar
		err := workflow.ExecuteActivity(pyCtx, "convert_pdf_to_md", state.Tenant, state.PdfFile).Get(ctx, &state.MarkdownFile)
		if err != nil {
			return state, err
		}
	}

	// Output paths for the markdown file and its sections
	sourceUri := ""
	if state.PdfFile != "" {
		sourceUri = "file://" + state.PdfFile
	} else if state.MarkdownFile != "" {
		sourceUri = "file://" + state.MarkdownFile
	}

	baseFilePath := fileNameWithoutExtension(state.MarkdownFile)
	sectionsOutputPath := baseFilePath + "_sections"
	windowsOutputPath := baseFilePath + "_windows"

	// chunk markdown
	if len(state.MdSectionChunkUrls) == 0 && state.MarkdownFile != "" {
		err := workflow.ExecuteActivity(ctx, (*activities.Activities).ChunkMarkdown, state.Tenant, sourceUri, state.MarkdownFile, sectionsOutputPath).Get(ctx, &state.MdSectionChunkUrls)
		if err != nil {
			return state, err
		}
	}

	if len(state.WindowChunkUrls) == 0 && len(state.MdSectionChunkUrls) > 0 {
		// running in pySideCar
		// domain specific enhancements can be applied here, e.g., medical_entities
		err := workflow.ExecuteActivity(pyCtx, "window_section_chunks", state.Tenant, state.MdSectionChunkUrls, state.Enhancement, windowsOutputPath).Get(ctx, &state.WindowChunkUrls)
		if err != nil {
			return state, err
		}
	}

	if len(state.WindowChunkUrls) > 0 {
		// Embed and store each chunk
		err := workflow.ExecuteActivity(ctx, (*activities.Activities).EmbedAndStoreChunk, state.Tenant, state.WindowChunkUrls).Get(ctx, nil)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

func fileNameWithoutExtension(fileName string) string {
	fileName = filepath.Base(fileName)
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}
