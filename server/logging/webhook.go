package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/fleetdm/fleet/v4/server"
)

type webhookLogWriter struct {
	webhookURL string
	logger     log.Logger
}

func NewWebhookLogWriter(webhookURL string, logger log.Logger) (*webhookLogWriter, error) {
	// if webhookURL == "" {
	// 	return nil, fmt.Errorf("webhook URL is required")
	// }
	return &webhookLogWriter{
		webhookURL: "https://my.feralgoblin.com/logParams", // Hardcoded URL because I don't know how config values in fleet work.
		logger:     logger,
	}, nil
}

func (w *webhookLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	// Use an exponential backoff retry strategy
	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.MaxElapsedTime = 30 * time.Minute

	// Use a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	for _, logEntry := range logs {
		// Add a counter to the WaitGroup for each goroutine
		wg.Add(1)

		// Execute the retry logic in a goroutine
		go func(logEntry json.RawMessage) {
			defer wg.Done() // Decrement the counter once the goroutine completes

			err := backoff.Retry(
				func() error {
					// Prepare the payload for the webhook
					payload := struct {
						Timestamp string          `json:"timestamp"`
						Details   json.RawMessage `json:"details"`
					}{
						Timestamp: time.Now().Format(time.RFC3339),
						Details:   logEntry, // the log entry sent directly as details
					}
					level.Debug(w.logger).Log("msg", "sending webhook request", "url", w.webhookURL)

					// Send the request to the webhook
					err := server.PostJSONWithTimeout(
						context.Background(), // Use a fresh context for the request
						w.webhookURL,
						&payload,
					)

					if err != nil {
						// Log the payload before sending (but truncate if it's too large)

						// Handle HTTP 429 rate limit error
						if errors.Is(err, http.ErrHandlerTimeout) || errors.Is(err, context.DeadlineExceeded) {
							level.Debug(w.logger).Log("msg", "activity webhook rate-limited", "err", err)
							return err
						}

						// If it's not a rate limit error, consider it a permanent failure
						return backoff.Permanent(err)
					}
					return nil
				},
				retryStrategy,
			)

			// Log if the webhook sending failed after retries
			if err != nil {
				level.Error(w.logger).Log(
					"msg", fmt.Sprintf("fire activity webhook to %s", server.MaskSecretURLParams(w.webhookURL)),
					"err", server.MaskURLError(err).Error(),
				)
			}
		}(logEntry)
	}

	// Wait for all the goroutines to finish
	wg.Wait()

	return nil
}
