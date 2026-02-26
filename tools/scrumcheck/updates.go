package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TimestampCheckResult struct {
	URL          string
	ExpiresAt    time.Time
	DurationLeft time.Duration
	DaysLeft     float64
	MinDays      int
	OK           bool
	Error        string
}

type timestampJSON struct {
	Signed struct {
		Expires string `json:"expires"`
	} `json:"signed"`
}

func checkUpdatesTimestamp(ctx context.Context, now time.Time) TimestampCheckResult {
	result := TimestampCheckResult{
		URL:     updatesTimestampURL,
		MinDays: minTimestampDays,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updatesTimestampURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("build request: %v", err)
		return result
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("fetch timestamp.json: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		return result
	}

	var payload timestampJSON
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		result.Error = fmt.Sprintf("parse timestamp.json: %v", err)
		return result
	}

	expiresAt, err := time.Parse(time.RFC3339, payload.Signed.Expires)
	if err != nil {
		result.Error = fmt.Sprintf("parse expires value %q: %v", payload.Signed.Expires, err)
		return result
	}

	result.ExpiresAt = expiresAt.UTC()
	result.DurationLeft = result.ExpiresAt.Sub(now.UTC())
	result.DaysLeft = result.DurationLeft.Hours() / 24
	result.OK = result.DurationLeft >= (time.Duration(minTimestampDays) * 24 * time.Hour)
	return result
}
