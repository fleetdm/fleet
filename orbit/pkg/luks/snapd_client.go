package luks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/rs/zerolog/log"
)

// snapdSocketPath is the privileged snapd REST API socket. Privileged
// operations (managing FDE recovery keys) require this socket and root, both of
// which orbit has. The unprivileged /run/snapd-snap.socket is not used.
const snapdSocketPath = "/run/snapd.socket"

// snapdChangePollInterval is how often we poll a snapd change for completion.
const snapdChangePollInterval = 250 * time.Millisecond

// snapdClient is a thin client for the snapd REST API spoken over its unix
// domain socket. It handles the standard snapd response envelope and the
// synchronous/asynchronous ("change") request patterns; endpoint-specific
// payloads live in snapd_fde.go.
type snapdClient struct {
	httpClient *http.Client
	// baseURL is the scheme+host portion of request URLs. The host is ignored
	// because the transport always dials the unix socket; it exists so tests can
	// point the client at an httptest server.
	baseURL string
}

func newSnapdClient() *snapdClient {
	client := fleethttp.NewClient(fleethttp.WithTimeout(60 * time.Second))
	// Override fleethttp's transport with one that dials the snapd unix socket.
	// The socket is local, so we intentionally skip the otelhttp/HTTP telemetry
	// layer that fleethttp installs by default; this mirrors the pattern used
	// by fleethttp's own tests when a bespoke transport is required.
	client.Transport = &http.Transport{ //nolint:gocritic // unix socket transport override
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", snapdSocketPath)
		},
	}
	return &snapdClient{
		httpClient: client,
		baseURL:    "http://localhost",
	}
}

// snapdResponse is the standard snapd REST API response envelope.
// See https://snapcraft.io/docs/snapd-rest-api.
type snapdResponse struct {
	Type       string          `json:"type"` // "sync" or "async"
	StatusCode int             `json:"status-code"`
	Status     string          `json:"status"`
	Result     json.RawMessage `json:"result"`
	Change     string          `json:"change"` // change id, set on async responses
}

// snapdError is the shape of the "result" field when a request fails.
type snapdError struct {
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// snapdChange is the relevant subset of a snapd change (GET /v2/changes/{id}).
type snapdChange struct {
	Status string          `json:"status"`
	Ready  bool            `json:"ready"`
	Err    string          `json:"err"`
	Data   json.RawMessage `json:"data"`
}

// requestSync performs a request that snapd answers synchronously and unmarshals
// the "result" field into out (which may be nil to discard it).
func (c *snapdClient) requestSync(ctx context.Context, method, path string, body any, out any) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	if out == nil || len(resp.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		return fmt.Errorf("decoding snapd %s result: %w", path, err)
	}
	return nil
}

// requestAsync performs a request that snapd answers asynchronously with a
// change id, waits for the change to complete, and unmarshals the change's
// "data" field into out (which may be nil to discard it).
func (c *snapdClient) requestAsync(ctx context.Context, method, path string, body any, out any) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	if resp.Change == "" {
		return fmt.Errorf("snapd %s did not return a change id", path)
	}

	data, err := c.waitChange(ctx, resp.Change)
	if err != nil {
		return err
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding snapd change data: %w", err)
	}
	return nil
}

// waitChange polls a snapd change until it is ready, returning its data on
// success or an error if the change failed or the context is cancelled.
func (c *snapdClient) waitChange(ctx context.Context, changeID string) (json.RawMessage, error) {
	ticker := time.NewTicker(snapdChangePollInterval)
	defer ticker.Stop()

	log.Debug().Str("change", changeID).Msg("waiting for snapd change to complete")
	for {
		var change snapdChange
		if err := c.requestSync(ctx, http.MethodGet, "/v2/changes/"+changeID, nil, &change); err != nil {
			return nil, fmt.Errorf("polling snapd change %s: %w", changeID, err)
		}
		log.Debug().Str("change", changeID).Str("status", change.Status).Bool("ready", change.Ready).Msg("polled snapd change")
		if change.Ready {
			if change.Status != "Done" {
				if change.Err != "" {
					return nil, fmt.Errorf("snapd change %s failed: %s", changeID, change.Err)
				}
				return nil, fmt.Errorf("snapd change %s ended with status %s", changeID, change.Status)
			}
			log.Debug().Str("change", changeID).Msg("snapd change completed successfully")
			return change.Data, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// do performs a single request and decodes the snapd envelope, returning an
// error for transport failures and for snapd error responses (status-code >=
// 400).
func (c *snapdClient) do(ctx context.Context, method, path string, body any) (*snapdResponse, error) {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding snapd request body: %w", err)
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("building snapd request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// NB: we intentionally do not log request or response bodies here — the
	// generate-recovery-key result contains the plaintext recovery key.
	log.Debug().Str("method", method).Str("path", path).Msg("snapd socket request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling snapd %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading snapd response: %w", err)
	}

	var decoded snapdResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decoding snapd response (status %d): %w", resp.StatusCode, err)
	}

	if decoded.StatusCode >= 400 {
		var snapErr snapdError
		_ = json.Unmarshal(decoded.Result, &snapErr)
		log.Debug().Str("path", path).Int("status_code", decoded.StatusCode).
			Str("kind", snapErr.Kind).Str("message", snapErr.Message).Msg("snapd socket error response")
		if snapErr.Message != "" {
			return nil, fmt.Errorf("snapd %s returned %d: %s", path, decoded.StatusCode, snapErr.Message)
		}
		return nil, fmt.Errorf("snapd %s returned status %d", path, decoded.StatusCode)
	}

	log.Debug().Str("path", path).Str("type", decoded.Type).Int("status_code", decoded.StatusCode).
		Str("status", decoded.Status).Str("change", decoded.Change).Msg("snapd socket response")
	return &decoded, nil
}
