package workers

import (
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

	if state.Markdown == "" && state.PdfFile != "" {
		// running in pySideCar
		err := workflow.ExecuteActivity(pyCtx, "convert_pdf_to_md", state.Tenant, state.PdfFile).Get(ctx, &state.Markdown)
		if err != nil {
			return state, err
		}
	}

	// chunk markdown
	if state.Markdown != "" {
		err := workflow.ExecuteActivity(ctx, (*IndexerActivities).ChunkMarkdown, state.Tenant, state.Markdown).Get(ctx, &state.MdSectionChunksUrl)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}
