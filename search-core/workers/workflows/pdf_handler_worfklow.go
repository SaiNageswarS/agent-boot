package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func PdfHandlerWorkflow(ctx workflow.Context, state PdfHandlerWorkflowInput) (string, error) {
	pyActivityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 20,
		TaskQueue:           "searchCorePySideCar",
	}
	pyCtx := workflow.WithActivityOptions(ctx, pyActivityOpts)

	var markdownFilePath string
	err := workflow.ExecuteActivity(pyCtx, "convert_pdf_to_md", state.Tenant, state.PdfFile).Get(ctx, &markdownFilePath)
	if err != nil {
		return "", err
	}

	return markdownFilePath, nil
}
