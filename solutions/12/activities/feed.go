package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// FeedActivities post messages to the activity feed service.
type FeedActivities struct {
	BaseURL string
	HTTP    *http.Client
}

func (a *FeedActivities) Announce(ctx context.Context, message string) error {
	body, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("feed returned %d", resp.StatusCode)
	}
	return nil
}
