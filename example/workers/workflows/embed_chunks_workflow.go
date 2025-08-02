package workflows

import (
	"time"

	"github.com/SaiNageswarS/agent-boot/example/workers/activities"
	"go.temporal.io/sdk/workflow"
)

func EmbedChunksWorkflow(ctx workflow.Context, input EmbedChunksWorkflowInput) error {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 100,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Find chunks missing embeddings
	var missingChunkIds []string
	err := workflow.ExecuteActivity(ctx, (*activities.Activities).GetChunksWithMissingEmbeddings, input.Tenant, input.SourceUri).Get(ctx, &missingChunkIds)
	if err != nil {
		return err
	}

	// Embed chunks
	err = workflow.ExecuteActivity(ctx, (*activities.Activities).EmbedChunks, input.Tenant, missingChunkIds).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
