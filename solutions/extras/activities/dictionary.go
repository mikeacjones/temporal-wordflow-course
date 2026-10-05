package activities

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"wordflow/solutions/extras/shared"
)

// DictionaryActivities call the dictionary service over HTTP.
type DictionaryActivities struct {
	BaseURL string
	HTTP    *http.Client
}

// LookupWord reports whether word is a real word.
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
