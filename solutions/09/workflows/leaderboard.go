package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/09/shared"
)

// LeaderboardWorkflow is the single, global leaderboard.
func LeaderboardWorkflow(ctx workflow.Context, input shared.LeaderboardInput) error {
	board := &game.Leaderboard{}

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
	}
}
