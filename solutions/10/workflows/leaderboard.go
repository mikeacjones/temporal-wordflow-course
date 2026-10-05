package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/10/shared"
)

// LeaderboardWorkflow is the single, global leaderboard.
func LeaderboardWorkflow(ctx workflow.Context, input shared.LeaderboardInput) error {
	board := input.State
	if board == nil {
		board = &game.Leaderboard{}
	}

	err := workflow.SetQueryHandler(ctx, shared.QueryLeaderboard, func() (game.LeaderboardView, error) {
		return board.View(), nil
	})
	if err != nil {
		return err
	}

	scores := workflow.GetSignalChannel(ctx, shared.SignalScore)
	for {
		var entry game.LeaderboardEntry
		scores.Receive(ctx, &entry)
		board.Upsert(entry)

		if shouldContinueAsNew(ctx) {
			// Apply Signals that already arrived. They do not carry over to the new run.
			for {
				var pending game.LeaderboardEntry
				if !scores.ReceiveAsync(&pending) {
					break
				}
				board.Upsert(pending)
			}
			return workflow.NewContinueAsNewError(ctx, LeaderboardWorkflow, shared.LeaderboardInput{State: board})
		}
	}
}
