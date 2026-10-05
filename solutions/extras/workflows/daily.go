package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/solutions/extras/activities"
)

// DailyPuzzleWorkflow announces the puzzle of the day. A Schedule starts it.
func DailyPuzzleWorkflow(ctx workflow.Context) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})
	var a *activities.PuzzleActivities
	var title string
	if err := workflow.ExecuteActivity(ctx, a.PickPuzzle, workflow.Now(ctx).YearDay()).Get(ctx, &title); err != nil {
		return "", err
	}
	announce(ctx, "Puzzle of the day: "+title)
	return title, nil
}
