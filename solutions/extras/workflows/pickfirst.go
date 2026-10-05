package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"wordflow/internal/game"
	"wordflow/solutions/extras/activities"
	"wordflow/solutions/extras/shared"
)

var dictionaryRegions = []string{"east", "west"}

// checkDictionary chooses how to look a word up. Games that started before
// the race existed keep making a single lookup.
func checkDictionary(ctx workflow.Context, word string) bool {
	if workflow.GetVersion(ctx, "race-dictionary", workflow.DefaultVersion, 1) == workflow.DefaultVersion {
		return isRealWord(ctx, word)
	}
	return isRealWordFastest(ctx, word)
}

// isRealWordFastest asks every region at once, takes the first answer, and
// cancels the rest.
func isRealWordFastest(ctx workflow.Context, word string) bool {
	ctx, cancel := workflow.WithCancel(ctx)
	defer cancel()

	var a *activities.DictionaryActivities
	selector := workflow.NewSelector(ctx)
	var valid bool
	var err error
	for _, region := range dictionaryRegions {
		regionCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			Summary:                "lookup in " + region,
			StartToCloseTimeout:    2 * time.Second,
			ScheduleToCloseTimeout: 10 * time.Second,
			HeartbeatTimeout:       time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    500 * time.Millisecond,
				BackoffCoefficient: 2,
				MaximumAttempts:    5,
			},
		})
		future := workflow.ExecuteActivity(regionCtx, a.LookupWordIn, shared.LookupInput{Word: game.Normalize(word), Region: region})
		selector.AddFuture(future, func(f workflow.Future) {
			err = f.Get(ctx, &valid)
		})
	}
	selector.Select(ctx)

	if err != nil {
		workflow.GetLogger(ctx).Warn("dictionary lookup failed", "word", word, "error", err)
		return false
	}
	return valid
}
