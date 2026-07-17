//go:build linux

package luks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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

// snapdChangeMaxWait bounds the total time waitChange will poll a change. If a
// snapd change never reaches "Ready" (stuck task, deadlock in snapd, etc.),
// the caller's context could otherwise let us poll forever. Two minutes is
// well above any realistic recovery-key enrollment on healthy snapd, but
// still tight enough that a hung snapd surfaces as a failure before the config
// tick tries again.
const snapdChangeMaxWait = 2 * time.Minute

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

// snapdSystemInfo is the relevant subset of GET /v2/system-info. Only "version"
// is consumed today, to preflight the recovery-key management API.
type snapdSystemInfo struct {
	Version string `json:"version"`
}

// systemInfo fetches GET /v2/system-info and returns the parsed system info.
func (c *snapdClient) systemInfo(ctx context.Context) (snapdSystemInfo, error) {
	var info snapdSystemInfo
	if err := c.requestSync(ctx, http.MethodGet, "/v2/system-info", nil, &info); err != nil {
		return snapdSystemInfo{}, err
	}
	return info, nil
}

// snapdError is the shape of the "result" field when a request fails.
type snapdError struct {
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// snapdAPIError is the typed error returned by snapdClient.do when snapd
// answers a request with a non-2xx envelope. It preserves the HTTP status
// code, snapd's "kind", and the message so callers can distinguish specific
// failure modes (e.g. a "resource already exists" conflict on
// add-recovery-key) from generic auth or transport failures without parsing
// error strings.
type snapdAPIError struct {
	StatusCode int
	Kind       string
	Message    string
	Path       string
}

func (e *snapdAPIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("snapd %s returned %d: %s", e.Path, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("snapd %s returned status %d", e.Path, e.StatusCode)
}

// isConflict reports whether the error looks like a snapd "resource already
// exists" response, which is the only case in which the add-recovery-key
// caller should fall back to replace-recovery-key.
func (e *snapdAPIError) isConflict() bool {
	if e == nil {
		return false
	}
	if e.StatusCode == http.StatusConflict {
		return true
	}
	// snapd emits kinds like "resource-already-exists" for these cases; a
	// substring match on either the kind or the message keeps us resilient to
	// small wording drift across snapd versions while still not matching
	// unrelated auth or transport errors.
	//
	// The message check uses "already exist" (no trailing s) as the prefix so
	// it matches both the singular ("key slot X already exists") and the
	// plural ("key slots [...] already exist") wording — snapd 2.75 returns
	// the plural form when a name-only keyslotRef expands to both the
	// system-data and system-save containers, which is exactly our case.
	kind := strings.ToLower(e.Kind)
	msg := strings.ToLower(e.Message)
	return strings.Contains(kind, "already-exists") ||
		strings.Contains(kind, "conflict") ||
		strings.Contains(msg, "already exist")
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
// success or an error if the change failed, the caller's context is cancelled,
// or snapdChangeMaxWait elapses without the change reaching Ready.
func (c *snapdClient) waitChange(ctx context.Context, changeID string) (json.RawMessage, error) {
	waitCtx, cancel := context.WithTimeout(ctx, snapdChangeMaxWait)
	defer cancel()

	ticker := time.NewTicker(snapdChangePollInterval)
	defer ticker.Stop()

	log.Debug().Str("change", changeID).Dur("max_wait", snapdChangeMaxWait).Msg("waiting for snapd change to complete")
	for {
		var change snapdChange
		if err := c.requestSync(waitCtx, http.MethodGet, "/v2/changes/"+changeID, nil, &change); err != nil {
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
		case <-waitCtx.Done():
			// Distinguish our poll deadline from a caller-initiated
			// cancellation so operators can tell which one fired.
			if ctx.Err() == nil {
				return nil, fmt.Errorf("timed out after %s waiting for snapd change %s", snapdChangeMaxWait, changeID)
			}
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
		return nil, &snapdAPIError{
			StatusCode: decoded.StatusCode,
			Kind:       snapErr.Kind,
			Message:    snapErr.Message,
			Path:       path,
		}
	}

	log.Debug().Str("path", path).Str("type", decoded.Type).Int("status_code", decoded.StatusCode).
		Str("status", decoded.Status).Str("change", decoded.Change).Msg("snapd socket response")
	return &decoded, nil
}
