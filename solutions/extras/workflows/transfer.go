package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/solutions/extras/activities"
	"wordflow/solutions/extras/shared"
)

// TransferPointsWorkflow gifts points from one player to another. Each step
// that succeeds adds a compensation. If a later step fails, the compensations
// run in reverse order.
func TransferPointsWorkflow(ctx workflow.Context, input shared.TransferInput) (err error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})
	// The Run ID is unique to this transfer, even when the Workflow ID is reused.
	transferID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	points := func(playerID, step string) shared.PointsActivityInput {
		return shared.PointsActivityInput{PlayerID: playerID, Amount: input.Amount, UpdateID: transferID + "-" + step}
	}
	var a *activities.PlayerActivities

	var compensations []func(workflow.Context) error
	defer func() {
		if err == nil {
			return
		}
		// Compensate even if this Workflow was canceled.
		ctx, _ := workflow.NewDisconnectedContext(ctx)
		for i := len(compensations) - 1; i >= 0; i-- {
			if cerr := compensations[i](ctx); cerr != nil {
				workflow.GetLogger(ctx).Error("compensation failed", "error", cerr)
			}
		}
	}()

	if err := workflow.ExecuteActivity(ctx, a.Withdraw, points(input.From, "withdraw")).Get(ctx, nil); err != nil {
		return err
	}
	compensations = append(compensations, func(ctx workflow.Context) error {
		return workflow.ExecuteActivity(ctx, a.Deposit, points(input.From, "refund")).Get(ctx, nil)
	})

	return workflow.ExecuteActivity(ctx, a.Deposit, points(input.To, "deposit")).Get(ctx, nil)
}
