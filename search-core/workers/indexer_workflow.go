package workers

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func IndexFileWorkflow(ctx workflow.Context, input IndexerWorkflowInput) (string, error) {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var chunksJsonPath string

	// running in pySideCar
	err := workflow.ExecuteActivity(ctx, "convert_pdf_to_md", input.Tenant, input.PdfFile).Get(ctx, &chunksJsonPath)
	if err != nil {
		return "", err
	}
	return chunksJsonPath, nil
}
