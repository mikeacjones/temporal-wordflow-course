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
