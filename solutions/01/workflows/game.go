package workflows

import (
	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/01/shared"
)

// GameWorkflow builds a new game and returns its first view.
func GameWorkflow(ctx workflow.Context, input shared.GameInput) (game.View, error) {
	gameID := workflow.GetInfo(ctx).WorkflowExecution.ID
	state := game.NewState(gameID, "", input.Puzzle, workflow.Now(ctx))
	return state.View(), nil
}
