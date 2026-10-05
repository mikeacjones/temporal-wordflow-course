package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/extras/activities"
	"wordflow/solutions/extras/shared"
)

const reviewTimeout = 24 * time.Hour

// PuzzleSubmissionWorkflow checks a submitted puzzle, waits for a person to
// approve it, then publishes it.
func PuzzleSubmissionWorkflow(ctx workflow.Context, input shared.SubmissionInput) (shared.SubmissionView, error) {
	puzzle := input.Puzzle
	view := shared.SubmissionView{Status: "checking"}
	err := workflow.SetQueryHandler(ctx, shared.QuerySubmission, func() (shared.SubmissionView, error) {
		return view, nil
	})
	if err != nil {
		return view, err
	}

	if err := game.ValidatePuzzle(puzzle); err != nil {
		view.Status = "invalid"
		return view, err
	}

	// Parallel Execution: start every lookup, then collect the results.
	lookupCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Second,
	})
	var a *activities.DictionaryActivities
	lookups := make([]workflow.Future, len(puzzle.Words))
	for i, word := range puzzle.Words {
		lookups[i] = workflow.ExecuteActivity(lookupCtx, a.LookupWord, word)
	}
	for i, lookup := range lookups {
		var valid bool
		if err := lookup.Get(ctx, &valid); err != nil {
			return view, err
		}
		if !valid {
			view.UnknownWords = append(view.UnknownWords, puzzle.Words[i])
		}
	}

	// Approval: wait for a review Signal, or give up after reviewTimeout.
	view.Status = "awaiting_review"
	var review shared.ReviewInput
	reviewed := false
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timer := workflow.NewTimerWithOptions(timerCtx, reviewTimeout, workflow.TimerOptions{Summary: "review deadline"})
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(workflow.GetSignalChannel(ctx, shared.SignalReview), func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &review)
		reviewed = true
	})
	selector.AddFuture(timer, func(workflow.Future) {})
	selector.Select(ctx)
	cancelTimer()

	if !reviewed {
		view.Status = "expired"
		return view, nil
	}
	view.Review = &review
	if !review.Approved {
		view.Status = "rejected"
		return view, nil
	}

	saveCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})
	var p *activities.PuzzleActivities
	if err := workflow.ExecuteActivity(saveCtx, p.SavePuzzle, puzzle).Get(ctx, nil); err != nil {
		return view, err
	}
	view.Status = "published"
	announce(ctx, "New puzzle: "+puzzle.Title+", approved by "+review.Reviewer)
	return view, nil
}
