package workflows

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/search-core/workers/activities"
	"go.temporal.io/sdk/workflow"
)

func ChunkMarkdownWorkflow(ctx workflow.Context, input ChunkMarkdownWorkflowInput) error {
	pyActivityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		TaskQueue:           "searchCorePySideCar",
	}
	pyCtx := workflow.WithActivityOptions(ctx, pyActivityOpts)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	baseFilePath := fileNameWithoutExtension(input.MarkdownFile)
	sectionsOutputPath := baseFilePath + "_sections"
	windowsOutputPath := baseFilePath + "_windows"

	// chunk markdown by sections.
	var mdSectionUrls []string
	err := workflow.ExecuteActivity(ctx, (*activities.Activities).ChunkMarkdown, input.Tenant, input.SourceUri, input.MarkdownFile, sectionsOutputPath).Get(ctx, &mdSectionUrls)
	if err != nil {
		return err
	}

	// window sections.
	var windowChunkUrls []string
	err = workflow.ExecuteActivity(pyCtx, "window_section_chunks", input.Tenant, mdSectionUrls, windowsOutputPath).Get(ctx, &windowChunkUrls)
	if err != nil {
		return err
	}

	// Save chunks
	err = workflow.ExecuteActivity(ctx, (*activities.Activities).SaveChunks, input.Tenant, windowChunkUrls).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

func fileNameWithoutExtension(fileName string) string {
	fileName = filepath.Base(fileName)
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}
