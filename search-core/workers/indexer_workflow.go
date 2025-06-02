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
	err := workflow.ExecuteActivity(ctx, (*IndexerActivities).ChunkPDF, input.Tenant, input.PdfFile).Get(ctx, &chunksJsonPath)
	if err != nil {
		return "", err
	}
	return chunksJsonPath, nil
}
