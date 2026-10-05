package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"wordflow/internal/game"
	"wordflow/solutions/10/shared"
)

// PlayerActivities send messages to Player Workflows using a Temporal Client.
type PlayerActivities struct {
	Client client.Client
}

// SpendPoints asks a Player Workflow to spend points.
func (a *PlayerActivities) SpendPoints(ctx context.Context, input shared.SpendPointsActivityInput) (shared.SpendPointsResult, error) {
	handle, err := a.Client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   shared.PlayerWorkflowID(input.PlayerID),
		UpdateID:     input.UpdateID,
		UpdateName:   shared.UpdateSpendPoints,
		Args:         []any{shared.SpendPointsInput{GameID: input.GameID, Amount: input.Amount}},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	var result shared.SpendPointsResult
	if err == nil {
		err = handle.Get(ctx, &result)
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		// The Player said no. Asking again will get the same answer.
		return result, temporal.NewNonRetryableApplicationError(appErr.Message(), appErr.Type(), err)
	}
	return result, err
}

// PublishScore sends a player's score to the leaderboard, starting the
// leaderboard Workflow if it is not running yet.
func (a *PlayerActivities) PublishScore(ctx context.Context, entry game.LeaderboardEntry) error {
	options := client.StartWorkflowOptions{
		ID:        shared.LeaderboardWorkflowID,
		TaskQueue: shared.TaskQueue,
	}
	_, err := a.Client.SignalWithStartWorkflow(ctx, shared.LeaderboardWorkflowID, shared.SignalScore, entry,
		options, shared.LeaderboardWorkflowType, shared.LeaderboardInput{})
	return err
}
