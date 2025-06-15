package workflows

import (
	"time"

	"github.com/SaiNageswarS/agent-boot/search-core/workers/activities"
	"go.temporal.io/sdk/workflow"
)

func InitTenantWorkflow(ctx workflow.Context, input InitTenantWorkflowInput) error {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)
	return workflow.ExecuteActivity(ctx, (*activities.Activities).InitTenant, input.Tenant).Get(ctx, nil)
}
