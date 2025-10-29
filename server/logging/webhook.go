package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
)

type webhookLogWriter struct {
	url    string
	logger log.Logger
}

func NewWebhookLogWriter(webhookURL string, logger log.Logger) (*webhookLogWriter, error) {
	if webhookURL == "" {
		return nil, errors.New("webhook URL missing")
	}

	return &webhookLogWriter{
		url:    webhookURL,
		logger: logger,
	}, nil
}

type webhookPayload struct {
	Timestamp time.Time         `json:"timestamp"`
	Details   []json.RawMessage `json:"details"`
}

func (w *webhookLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {

	payload := webhookPayload{
		Timestamp: time.Now(),
		Details:   logs,
	}

	level.Debug(w.logger).Log(
		"msg", "sending webhook request",
		"url", server.MaskSecretURLParams(w.url),
	)

	if err := server.PostJSONWithTimeout(ctx, w.url, payload); err != nil {
		level.Error(w.logger).Log(
			"msg", fmt.Sprintf("failed to send automation webhook to %s", server.MaskSecretURLParams(w.url)),
			"err", server.MaskURLError(err).Error(),
		)
	}

	return nil
}
