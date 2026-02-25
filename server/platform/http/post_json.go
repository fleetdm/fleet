package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// errWithStatus is an error with a particular status code.
type errWithStatus struct {
	err        string
	statusCode int
}

// Error implements the error interface.
func (e *errWithStatus) Error() string {
	return e.err
}

// StatusCode implements the StatusCoder interface for returning custom status codes.
func (e *errWithStatus) StatusCode() int {
	return e.statusCode
}

// PostJSONWithTimeout marshals v as JSON and POSTs it to the given URL with a 30-second timeout.
func PostJSONWithTimeout(ctx context.Context, url string, v any, logger *slog.Logger) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to %s: %s, request-size=%d", MaskSecretURLParams(url), MaskURLError(err), len(jsonBytes))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 512 {
			bodyStr = bodyStr[:512]
		}
		logger.DebugContext(ctx, "non-success response from POST",
			"url", MaskSecretURLParams(url),
			"status_code", resp.StatusCode,
			"body", bodyStr,
		)
		return &errWithStatus{err: fmt.Sprintf("error posting to %s", MaskSecretURLParams(url)), statusCode: resp.StatusCode}
	}

	return nil
}
