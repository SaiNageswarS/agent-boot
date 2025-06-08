package workers

import (
	"path/filepath"
	"strings"
	"time"

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
	baseFilePath := fileNameWithoutExtension(state.MarkdownFile)
	sectionsOutputPath := baseFilePath + "_sections"
	windowsOutputPath := baseFilePath + "_windows"

	// chunk markdown
	if len(state.MdSectionChunkUrls) == 0 && state.MarkdownFile != "" {
		err := workflow.ExecuteActivity(ctx, (*IndexerActivities).ChunkMarkdown, state.Tenant, state.MarkdownFile, sectionsOutputPath).Get(ctx, &state.MdSectionChunkUrls)
		if err != nil {
			return state, err
		}
	}

	if len(state.MdSectionChunkUrls) != 0 {
		// running in pySideCar
		// domain specific enhancements can be applied here, e.g., medical_entities
		for _, sectionChunkUrl := range state.MdSectionChunkUrls {
			var windowChunksUrls []string
			err := workflow.ExecuteActivity(pyCtx, "window_section_chunks", state.Tenant, sectionChunkUrl, state.Enhancement, windowsOutputPath).Get(ctx, &windowChunksUrls)
			if err != nil {
				return state, err
			}

			state.WindowChunkUrls = append(state.WindowChunkUrls, windowChunksUrls...)
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
