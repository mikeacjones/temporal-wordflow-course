# Extra · Pick First (Go)

The dictionary service already supports `?region=NAME`.

## 1. Input

Add to `app/shared/extras.go`:

```go
// Extra: Pick First.
type LookupInput struct {
	Word   string `json:"word"`
	Region string `json:"region"`
}
```

## 2. A region-aware Activity

In `app/activities/dictionary.go`, rename the body of `LookupWord` to a private `lookup` that takes a region, then add `LookupWordIn`. Import `"go.temporal.io/sdk/activity"` and `"wordflow/app/shared"`:

```go
func (a *DictionaryActivities) LookupWord(ctx context.Context, word string) (bool, error) {
	return a.lookup(ctx, word, "")
}

// LookupWordIn asks one region of the dictionary. Canceling ctx abandons the
// request.
func (a *DictionaryActivities) LookupWordIn(ctx context.Context, input shared.LookupInput) (bool, error) {
	// A running Activity learns it was canceled from a heartbeat response.
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				activity.RecordHeartbeat(ctx)
			case <-done:
				return
			}
		}
	}()
	return a.lookup(ctx, input.Word, input.Region)
}

func (a *DictionaryActivities) lookup(ctx context.Context, word, region string) (bool, error) {
	u := a.BaseURL + "/words/" + url.PathEscape(word)
	if region != "" {
		u += "?region=" + url.QueryEscape(region)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	// ... the rest of the old LookupWord, unchanged
}
```

Goroutines are fine in Activities. Only Workflow code has restrictions. The SDK throttles heartbeats, so calling `RecordHeartbeat` often is cheap.

## 3. Race the regions

Create `app/workflows/pickfirst.go`:

```go
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"wordflow/app/activities"
	"wordflow/app/shared"
	"wordflow/internal/game"
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
```

- `workflow.Selector` is the Workflow-safe `select`. `Select` runs the callback of the first Future that's ready.
- `defer cancel()` cancels the losing Activity when the function returns.
- `GetVersion` can be called inside a handler. It only runs when a guess needs a lookup, which is deterministic.
- The slice gives a fixed order. Ranging over a map here would not be deterministic.

In the `guess` handler in `game.go`, call `checkDictionary` instead of `isRealWord`:

```go
			if outcome == game.OutcomeNotInPuzzle && checkDictionary(ctx, guess.Word) && !state.Over() {
```

## 4. Run it

Restart the Worker.
