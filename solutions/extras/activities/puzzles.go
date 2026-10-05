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

// SavePuzzle publishes a puzzle so it can be played.
func (a *PuzzleActivities) SavePuzzle(ctx context.Context, puzzle game.Puzzle) error {
	return a.Store.Save(puzzle)
}

// PickPuzzle chooses a puzzle for a given day and returns its title.
func (a *PuzzleActivities) PickPuzzle(ctx context.Context, day int) (string, error) {
	list, err := a.Store.List()
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", temporal.NewNonRetryableApplicationError("there are no puzzles", "no_puzzles", nil)
	}
	return list[day%len(list)].Title, nil
}
