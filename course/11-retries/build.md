# 11 · Retries in depth (Go)

## 1. The Activity

Create `app/activities/dictionary.go`:

```go
package activities

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.temporal.io/sdk/temporal"
)

// DictionaryActivities call the dictionary service over HTTP.
type DictionaryActivities struct {
	BaseURL string
	HTTP    *http.Client
}

// LookupWord reports whether word is a real word.
func (a *DictionaryActivities) LookupWord(ctx context.Context, word string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL+"/words/"+url.PathEscape(word), nil)
	if err != nil {
		return false, err
	}
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusUnauthorized:
		// Retrying will not fix bad credentials.
		return false, temporal.NewNonRetryableApplicationError("dictionary rejected our credentials", "unauthorized", nil)
	case http.StatusTooManyRequests:
		// Wait as long as the service asks before the next attempt.
		seconds, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		return false, temporal.NewApplicationErrorWithOptions("dictionary rate limit", "rate_limited", temporal.ApplicationErrorOptions{
			NextRetryDelay: time.Duration(seconds) * time.Second,
		})
	default:
		return false, fmt.Errorf("dictionary returned %d", resp.StatusCode)
	}
}
```

- Use the Activity's `ctx` for the request. When the attempt times out, the request is cancelled.
- `NextRetryDelay` replaces the retry policy's wait for the next attempt only.
- Instead of making the error non-retryable in the Activity, you could list `"unauthorized"` in the retry policy's `NonRetryableErrorTypes`. Use the Activity when the error is always permanent, and the policy when it depends on the caller.

## 2. Register it

In `app/cmd/worker/main.go`, import `"net/http"` and `"os"`, then:

```go
	servicesURL := os.Getenv("SERVICES_URL")
	if servicesURL == "" {
		servicesURL = "http://localhost:8080/services"
	}
```

```go
	w.RegisterActivity(&activities.DictionaryActivities{BaseURL: servicesURL + "/dictionary", HTTP: http.DefaultClient})
```

Configuration belongs in the Worker and Activities. Workflow code can't read environment variables.

## 3. Call it

In `app/workflows/game.go`, import `"go.temporal.io/sdk/temporal"` and add:

```go
// isRealWord asks the dictionary service. If it cannot answer in time, the
// guess simply does not earn a bonus.
func isRealWord(ctx workflow.Context, word string) bool {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    2 * time.Second,  // one attempt
		ScheduleToCloseTimeout: 10 * time.Second, // every attempt: the player is waiting
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 2,
			MaximumAttempts:    5,
		},
	})
	var a *activities.DictionaryActivities
	var valid bool
	if err := workflow.ExecuteActivity(ctx, a.LookupWord, game.Normalize(word)).Get(ctx, &valid); err != nil {
		workflow.GetLogger(ctx).Warn("dictionary lookup failed", "word", word, "error", err)
		return false
	}
	return valid
}
```

In the `guess` handler, after `state.Guess`:

```go
			outcome := state.Guess(guess.Word, workflow.Now(ctx))
			if outcome == game.OutcomeNotInPuzzle && isRealWord(ctx, guess.Word) && !state.Over() {
				state.MarkBonus(guess.Word)
				outcome = game.OutcomeBonus
			}
```

The handler holds the lock while the lookup runs, so other guesses wait their turn. The clock doesn't wait, so check `state.Over()` after the lookup.

## 4. Run it

```sh
make clean-workflows
```

Restart the Worker. Compare with `solutions/11`.
