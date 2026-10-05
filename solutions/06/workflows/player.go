package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/06/shared"
)

// PlayerWorkflow is one player for as long as they play.
func PlayerWorkflow(ctx workflow.Context, input shared.PlayerInput) error {
	player := game.NewPlayer(input.Name, workflow.Now(ctx))

	err := workflow.SetQueryHandler(ctx, shared.QueryPlayer, func() (game.PlayerView, error) {
		return player.View(), nil
	})
	if err != nil {
		return err
	}

	err = workflow.SetUpdateHandler(ctx, shared.UpdateJoin, func(ctx workflow.Context) (game.PlayerView, error) {
		return player.View(), nil
	})
	if err != nil {
		return err
	}

	// The Workflow stays open, holding the player's state, until something
	// ends it. Lesson 10 adds Continue-As-New to keep its history small.
	return workflow.Await(ctx, func() bool { return false })
}
