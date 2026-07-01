package logging

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const (
	// splunkHECPath is the Splunk HTTP Event Collector endpoint.
	splunkHECPath = "/services/collector/event"
	// splunkHealthPath is the HEC health check endpoint.
	splunkHealthPath = "/services/collector/health"
	// splunkMaxBatchSize is the default max content length for HEC (1 MB).
	splunkMaxBatchSize = 1_000_000
	// splunkMaxSizeOfRecord is the max size of a single HEC event (1 MB).
	splunkMaxSizeOfRecord = 1_000_000
	// splunkMaxRetries is the maximum number of retries on transient errors.
	splunkMaxRetries = 8
)

// splunkEvent wraps a log entry in the Splunk HEC event format.
type splunkEvent struct {
	Event json.RawMessage `json:"event"`
	// Time is the event timestamp in epoch seconds.
	Time float64 `json:"time,omitempty"`
	// Index is the Splunk index to send events to.
	Index string `json:"index,omitempty"`
	// Source overrides the default source.
	Source string `json:"source,omitempty"`
	// SourceType overrides the default sourcetype.
	SourceType string `json:"sourcetype,omitempty"`
}

type splunkLogWriter struct {
	url        string
	token      string
	index      string
	source     string
	sourceType string
	client     *http.Client
	logger     *slog.Logger
}

func NewSplunkLogWriter(url, token, index, source, sourceType string, insecureSkipVerify bool, logger *slog.Logger) (*splunkLogWriter, error) {
	clientOpts := []fleethttp.ClientOpt{fleethttp.WithTimeout(30 * time.Second)}
	if insecureSkipVerify {
		clientOpts = append(clientOpts, fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // user-configured option for self-signed certs
		}))
	}

	w := &splunkLogWriter{
		url:        url,
		token:      token,
		index:      index,
		source:     source,
		sourceType: sourceType,
		client:     fleethttp.NewClient(clientOpts...),
		logger:     logger,
	}

	if err := w.checkHealth(); err != nil {
		return nil, fmt.Errorf("splunk health check: %w", err)
	}

	return w, nil
}

func (w *splunkLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	if len(logs) == 0 {
		return nil
	}

	now := float64(time.Now().UnixNano()) / float64(time.Second)

	var buf bytes.Buffer
	for _, l := range logs {
		evt := splunkEvent{
			Event:      l,
			Time:       now,
			Index:      w.index,
			Source:     w.source,
			SourceType: w.sourceType,
		}
		b, err := json.Marshal(evt)
		if err != nil {
			w.logger.ErrorContext(ctx, "failed to marshal splunk event", "err", err)
			continue
		}

		if len(b) > splunkMaxSizeOfRecord {
			w.logger.InfoContext(ctx, "dropping splunk event over 1MB limit",
				"size", len(b),
			)
			continue
		}

		// If adding this event would exceed the batch size, flush first.
		if buf.Len() > 0 && buf.Len()+len(b) > splunkMaxBatchSize {
			if err := w.send(ctx, buf.Bytes()); err != nil {
				return err
			}
			buf.Reset()
		}

		buf.Write(b)
	}

	if buf.Len() > 0 {
		return w.send(ctx, buf.Bytes())
	}

	return nil
}

func (w *splunkLogWriter) send(ctx context.Context, payload []byte) error {
	return w.sendWithRetry(ctx, payload, 0)
}

// splunkRetryDelay calculates the backoff duration for a given retry attempt.
// Exported as a var so tests can override it to avoid waiting.
var splunkRetryDelay = func(try int) time.Duration {
	return 100 * time.Millisecond * time.Duration(1<<try)
}

func (w *splunkLogWriter) sendWithRetry(ctx context.Context, payload []byte, try int) error {
	if try > 0 {
		timer := time.NewTimer(splunkRetryDelay(try))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctxerr.Wrap(ctx, ctx.Err(), "splunk retry canceled")
		case <-timer.C:
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url+splunkHECPath, bytes.NewReader(payload))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "splunk create request")
	}
	req.Header.Set("Authorization", "Splunk "+w.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "splunk send")
	}
	defer resp.Body.Close()

	if (resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests) && try < splunkMaxRetries {
		io.Copy(io.Discard, resp.Body) //nolint:errcheck // best-effort drain for connection reuse
		resp.Body.Close()
		return w.sendWithRetry(ctx, payload, try+1)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ctxerr.Errorf(ctx, "splunk HEC returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *splunkLogWriter) checkHealth() error {
	req, err := http.NewRequest(http.MethodGet, w.url+splunkHealthPath, nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}
	req.Header.Set("Authorization", "Splunk "+w.token)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("health request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HEC health check returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
