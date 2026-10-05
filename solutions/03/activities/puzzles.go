// Package activities holds code that talks to the outside world.
package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/temporal"

	"wordflow/internal/game"
	"wordflow/internal/puzzles"
)

// PuzzleActivities gets its dependencies when the Worker starts.
type PuzzleActivities struct {
	Store *puzzles.Store
}

func (a *PuzzleActivities) LoadPuzzle(ctx context.Context, puzzleID string) (game.Puzzle, error) {
	puzzle, err := a.Store.Get(puzzleID)
	if errors.Is(err, puzzles.ErrNotFound) {
		return game.Puzzle{}, temporal.NewNonRetryableApplicationError("no puzzle named "+puzzleID, "puzzle_not_found", err)
	}
	return puzzle, err
}
