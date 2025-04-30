package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
)

var defaultTimeout = 15 * time.Second

type webhookLogWriter struct {
	url           string
	basicAuthUser string
	basicAuthPass string
	bearerToken   string
	timeout       time.Duration
	logger        log.Logger
}

func NewWebhookLogWriter(webhookURL, basicAuthUser, basicAuthPass, bearerToken string, timeout time.Duration, logger log.Logger) (*webhookLogWriter, error) {
	if webhookURL == "" {
		return nil, errors.New("webhook URL missing")
	}

	// We can only have basic auth or bearer, not both
	if (basicAuthUser != "" || basicAuthPass != "") && bearerToken != "" {
		return nil, errors.New("basic auth and bearer token cannot be used at the same time")
	}

	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &webhookLogWriter{
		url:           webhookURL,
		basicAuthUser: basicAuthUser,
		basicAuthPass: basicAuthPass,
		bearerToken:   bearerToken,
		timeout:       timeout,
		logger:        logger,
	}, nil
}

type webhookDetails struct {
	Timestamp time.Time       `json:"timestamp"`
	Details   json.RawMessage `json:"details"`
}

func (w *webhookLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	detailLogs := make([]webhookDetails, 0, len(logs))

	for _, log := range logs {
		detailLogs = append(detailLogs, webhookDetails{
			Timestamp: time.Now(),
			Details:   log,
		})
	}

	level.Debug(w.logger).Log(
		"msg", "sending webhook request",
		"url", server.MaskSecretURLParams(w.url),
	)

	if err := w.sendWebhookJson(ctx, detailLogs); err != nil {
		level.Error(w.logger).Log(
			"msg", fmt.Sprintf("failed to send automation webhook to %s", server.MaskSecretURLParams(w.url)),
			"err", server.MaskURLError(err).Error(),
		)
	}

	return nil
}

func (w *webhookLogWriter) sendWebhookJson(ctx context.Context, v any) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(w.timeout))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// We already check that only one auth type is set
	if w.basicAuthUser != "" || w.basicAuthPass != "" {
		req.SetBasicAuth(w.basicAuthUser, w.basicAuthPass)
	}

	if w.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+w.bearerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to %s: %s, request-size=%d", server.MaskSecretURLParams(w.url), server.MaskURLError(err), len(jsonBytes))
	}
	defer resp.Body.Close()

	if !httpSuccessStatus(resp.StatusCode) {
		body, _ := io.ReadAll(resp.Body)
		return &errWithStatus{err: fmt.Sprintf("error posting to %s: %d. %s", server.MaskSecretURLParams(w.url), resp.StatusCode, string(body)), statusCode: resp.StatusCode}
	}

	return nil

}

func httpSuccessStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}

// errWithStatus is an error with a particular status code.
type errWithStatus struct {
	err        string
	statusCode int
}

// Error implements the error interface
func (e *errWithStatus) Error() string {
	return e.err
}

// StatusCode implements the StatusCoder interface for returning custom status codes.
func (e *errWithStatus) StatusCode() int {
	return e.statusCode
}
