// Package api implements the HTTP endpoints that talk to Temporal.
//
// The HTTP routing is done for you in internal/httpapi. Each method here is
// one endpoint. Return httpapi.ErrNotImplemented until its lesson.
package api

import (
	"context"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"wordflow/internal/game"
	"wordflow/internal/httpapi"
	"wordflow/internal/puzzles"
	"wordflow/solutions/04/shared"
	"wordflow/solutions/04/workflows"
)

type API struct {
	Temporal client.Client
	Puzzles  *puzzles.Store
}

func New(temporalClient client.Client, store *puzzles.Store) *API {
	return &API{Temporal: temporalClient, Puzzles: store}
}

// StartGame handles POST /api/games. Lesson 1.
func (a *API) StartGame(ctx context.Context, req httpapi.StartGameRequest) (httpapi.StartGameResponse, error) {
	startOp := a.Temporal.NewWithStartWorkflowOperation(client.StartWorkflowOptions{
		ID:                       shared.GuestGameID(req.RequestID),
		TaskQueue:                shared.TaskQueue,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, workflows.GameWorkflow, shared.GameInput{PuzzleID: req.PuzzleID})

	handle, err := a.Temporal.UpdateWithStartWorkflow(ctx, client.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startOp,
		UpdateOptions: client.UpdateWorkflowOptions{
			UpdateName:   shared.UpdateReady,
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	})
	if err != nil {
		return httpapi.StartGameResponse{}, err
	}

	var view game.View
	if err := handle.Get(ctx, &view); err != nil {
		return httpapi.StartGameResponse{}, err
	}
	return httpapi.StartGameResponse{GameID: handle.WorkflowID(), Game: &view}, nil
}

// GetGame handles GET /api/games/{id}. Lesson 3.
func (a *API) GetGame(ctx context.Context, gameID string) (game.View, error) {
	response, err := a.Temporal.QueryWorkflow(ctx, gameID, "", shared.QueryState)
	if err != nil {
		return game.View{}, err
	}
	var view game.View
	err = response.Get(&view)
	return view, err
}

// SubmitGuess handles POST /api/games/{id}/guesses. Lesson 3.
func (a *API) SubmitGuess(ctx context.Context, req httpapi.GuessRequest) (*game.GuessResult, error) {
	handle, err := a.Temporal.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   req.GameID,
		UpdateID:     req.RequestID,
		UpdateName:   shared.UpdateGuess,
		Args:         []any{shared.GuessInput{Word: req.Word}},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return nil, err
	}
	var result game.GuessResult
	if err := handle.Get(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ExtendTime handles POST /api/games/{id}/extend. Lesson 5.
func (a *API) ExtendTime(ctx context.Context, req httpapi.GameRequest) (game.View, error) {
	return game.View{}, httpapi.ErrNotImplemented
}

// JoinPlayer handles POST /api/players. Lesson 6.
func (a *API) JoinPlayer(ctx context.Context, name string) (game.PlayerView, error) {
	return game.PlayerView{}, httpapi.ErrNotImplemented
}

// GetPlayer handles GET /api/players/{name}. Lesson 6.
func (a *API) GetPlayer(ctx context.Context, name string) (game.PlayerView, error) {
	return game.PlayerView{}, httpapi.ErrNotImplemented
}

// UseHint handles POST /api/games/{id}/hints. Lesson 8.
func (a *API) UseHint(ctx context.Context, req httpapi.GameRequest) (game.HintResult, error) {
	return game.HintResult{}, httpapi.ErrNotImplemented
}

// GetLeaderboard handles GET /api/leaderboard. Lesson 9.
func (a *API) GetLeaderboard(ctx context.Context) (game.LeaderboardView, error) {
	return game.LeaderboardView{}, httpapi.ErrNotImplemented
}
