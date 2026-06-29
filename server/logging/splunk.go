package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func NewSplunkLogWriter(url, token, index, source, sourceType string, logger *slog.Logger) (*splunkLogWriter, error) {
	if url == "" {
		return nil, errors.New("splunk URL must not be empty")
	}
	if token == "" {
		return nil, errors.New("splunk HEC token must not be empty")
	}

	w := &splunkLogWriter{
		url:        url,
		token:      token,
		index:      index,
		source:     source,
		sourceType: sourceType,
		client:     fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second)),
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HEC health check returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
