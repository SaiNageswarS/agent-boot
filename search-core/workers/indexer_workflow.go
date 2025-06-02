package workers

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func IndexFileWorkflow(ctx workflow.Context, state IndexerWorkflowState) (IndexerWorkflowState, error) {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	if state.PdfFile != "" {
		// running in pySideCar
		err := workflow.ExecuteActivity(ctx, "convert_pdf_to_md", state.Tenant, state.PdfFile).Get(ctx, &state.Markdown)
		if err != nil {
			return state, err
		}
	}

	// chunk markdown

	return state, nil
}
