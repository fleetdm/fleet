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
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	platformhttp "github.com/fleetdm/fleet/v4/server/platform/http"
	"google.golang.org/api/idtoken"
)

const (
	// Cloud Run HTTP/1 requests are limited to 32 MiB. HTTP/2 can support
	// larger requests, but keeping this limit makes oversized log behavior
	// predictable for default Cloud Run services.
	cloudRunServiceMaxSizeOfPayload = 32 * 1024 * 1024
	cloudRunServiceTimeout          = 30 * time.Second
)

type cloudRunServiceLogWriter struct {
	client *http.Client
	url    string
	logger *slog.Logger
}

func NewCloudRunServiceLogWriter(ctx context.Context, serviceURL, audience string, logger *slog.Logger) (*cloudRunServiceLogWriter, error) {
	if serviceURL == "" {
		return nil, errors.New("Cloud Run service URL missing")
	}
	parsedServiceURL, err := url.Parse(serviceURL)
	if err != nil || parsedServiceURL.Scheme == "" || parsedServiceURL.Host == "" {
		return nil, fmt.Errorf("invalid Cloud Run service URL: %q", serviceURL)
	}

	var client *http.Client
	if audience != "" {
		idTokenClient, err := idtoken.NewClient(ctx, audience)
		if err != nil {
			return nil, fmt.Errorf("create Cloud Run service ID token client: %w", err)
		}
		idTokenClient.Timeout = cloudRunServiceTimeout
		client = idTokenClient
	} else {
		client = fleethttp.NewClient(fleethttp.WithTimeout(cloudRunServiceTimeout))
	}

	return &cloudRunServiceLogWriter{
		client: client,
		url:    serviceURL,
		logger: logger,
	}, nil
}

func (w *cloudRunServiceLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		if len(log) > cloudRunServiceMaxSizeOfPayload {
			return fmt.Errorf("cloudrun_service POST to %s: log size %d exceeds 32 MiB request limit", platformhttp.MaskSecretURLParams(w.url), len(log))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(log))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		w.logger.DebugContext(ctx, "sending Cloud Run service request",
			"url", platformhttp.MaskSecretURLParams(w.url),
		)

		resp, err := w.client.Do(req)
		if err != nil {
			return fmt.Errorf("cloudrun_service POST to %s failed: %w", platformhttp.MaskSecretURLParams(w.url), platformhttp.MaskURLError(err))
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 513))
			resp.Body.Close()
			bodyStr := string(body)
			if len(bodyStr) > 512 {
				bodyStr = bodyStr[:512]
			}
			w.logger.DebugContext(ctx, "non-success response from Cloud Run service",
				"url", platformhttp.MaskSecretURLParams(w.url),
				"status_code", resp.StatusCode,
				"body", bodyStr,
			)
			return fmt.Errorf("cloudrun_service POST to %s returned status %d", platformhttp.MaskSecretURLParams(w.url), resp.StatusCode)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	return nil
}
