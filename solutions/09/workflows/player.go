package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/09/activities"
	"wordflow/solutions/09/shared"
)

// PlayerWorkflow is one player for as long as they play.
func PlayerWorkflow(ctx workflow.Context, input shared.PlayerInput) error {
	player := game.NewPlayer(input.Name, workflow.Now(ctx))
	var activeGame workflow.ChildWorkflowFuture

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

	err = workflow.SetUpdateHandlerWithOptions(ctx, shared.UpdateStartGame,
		func(ctx workflow.Context, input shared.StartGameInput) (shared.StartGameResult, error) {
			// Change state before blocking, so a second startGame sees this game.
			gameID := player.NextGameID()
			player.StartGame(gameID)

			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: gameID})
			future := workflow.ExecuteChildWorkflow(childCtx, GameWorkflow, shared.GameInput{
				PuzzleID: input.PuzzleID,
				PlayerID: player.Name,
			})
			// Wait until the child has started, not until it finishes.
			if err := future.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
				player.ClearGame(gameID)
				return shared.StartGameResult{}, err
			}
			activeGame = future
			return shared.StartGameResult{GameID: gameID}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, input shared.StartGameInput) error {
				return player.CheckStartGame()
			},
		},
	)
	if err != nil {
		return err
	}

	err = workflow.SetUpdateHandlerWithOptions(ctx, shared.UpdateSpendPoints,
		func(ctx workflow.Context, input shared.SpendPointsInput) (shared.SpendPointsResult, error) {
			player.Spend(input.Amount)
			return shared.SpendPointsResult{Remaining: player.Points}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, input shared.SpendPointsInput) error {
				return player.CheckSpend(input.GameID, input.Amount)
			},
		},
	)
	if err != nil {
		return err
	}

	for {
		if err := workflow.Await(ctx, func() bool { return activeGame != nil }); err != nil {
			return err
		}
		var result game.Result
		if err := activeGame.Get(ctx, &result); err != nil {
			workflow.GetLogger(ctx).Error("game failed", "gameID", player.ActiveGameID, "error", err)
			player.ClearGame(player.ActiveGameID)
		} else if player.FinishGame(result) {
			publishScore(ctx, player)
		}
		activeGame = nil
	}
}

func publishScore(ctx workflow.Context, player *game.Player) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})
	var a *activities.PlayerActivities
	err := workflow.ExecuteActivity(ctx, a.PublishScore, player.LeaderboardEntry()).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("publish score failed", "error", err)
	}
}
