package connectivity

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/sync/errgroup"
)

// fingerprintBodyLimit caps how much body we buffer when fingerprinting. The
// Fleet JSON error shape is small (a few hundred bytes); anything larger is
// almost certainly not what we're looking for.
const fingerprintBodyLimit = 8 << 10 // 8 KiB

// defaultConcurrency caps in-flight probes. Chosen to stay well under typical
// connection-per-host limits while finishing a full scan in a few seconds.
const defaultConcurrency = 8

// Options controls how Probe issues requests to the target server.
type Options struct {
	// BaseURL is the Fleet server URL (e.g. https://fleet.example.com). It
	// may include a port but must not include a path.
	BaseURL string
	// RootCAs, if set, overrides the system cert pool. Use this to trust a
	// private Fleet CA, matching orbit's --fleet-certificate.
	RootCAs *x509.CertPool
	// Insecure skips TLS verification. Matches orbit's --insecure.
	Insecure bool
	// Timeout bounds each individual request. Zero means 10 seconds.
	Timeout time.Duration
	// Concurrency caps parallel probes. Zero means 8.
	Concurrency int
	// OrbitNodeKey, if non-empty, is sent as the request body for checks
	// with Auth == AuthOrbitNodeKey. When empty, those checks fall back to
	// unauthenticated probes.
	OrbitNodeKey string
}

// Probe runs the given checks against Options.BaseURL and returns results in
// the same order as the input slice. The context cancels outstanding probes.
func Probe(ctx context.Context, opts Options, checks []Check) ([]Result, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("connectivity: BaseURL is required")
	}
	base, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("base url %q is missing scheme or host", opts.BaseURL)
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	tlsConf := &tls.Config{
		RootCAs:            opts.RootCAs,
		InsecureSkipVerify: opts.Insecure, //nolint:gosec // opt-in per flag
		MinVersion:         tls.VersionTLS12,
	}
	client := fleethttp.NewClient(
		fleethttp.WithTimeout(timeout),
		fleethttp.WithTLSClientConfig(tlsConf),
		fleethttp.WithFollowRedir(false),
	)

	results := make([]Result, len(checks))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, c := range checks {
		g.Go(func() error {
			results[i] = runOne(gCtx, client, base, c, opts.OrbitNodeKey)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return results, err
	}
	return results, nil
}

func runOne(ctx context.Context, client *http.Client, base *url.URL, c Check, orbitNodeKey string) Result {
	target := *base
	target.Path = strings.TrimRight(base.Path, "/") + c.Path

	method, body, contentType, authUsed := prepareRequest(c, orbitNodeKey)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return Result{Check: c, Status: StatusBlocked, Latency: time.Since(start), Error: err.Error()}
	}
	req.Header.Set("User-Agent", "fleet-connectivity-probe")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Result{Check: c, Status: StatusBlocked, Latency: latency, Error: classifyNetErr(err)}
	}
	defer resp.Body.Close()

	// Buffer up to fingerprintBodyLimit so we can inspect for Fleet's JSON
	// error shape. The remainder is drained so the connection can be reused.
	bodyPeek, _ := io.ReadAll(io.LimitReader(resp.Body, fingerprintBodyLimit))
	_, _ = io.Copy(io.Discard, resp.Body)

	return classifyResponse(c, resp, bodyPeek, latency, authUsed)
}

// prepareRequest returns the method, request body, content type, and whether
// auth was attached for a check. AuthOrbitNodeKey downgrades to AuthNone when
// no key is available.
func prepareRequest(c Check, orbitNodeKey string) (method string, body io.Reader, contentType string, authUsed AuthMode) {
	method = c.Method
	if c.Auth == AuthOrbitNodeKey && orbitNodeKey != "" {
		payload, _ := json.Marshal(map[string]string{"orbit_node_key": orbitNodeKey})
		return method, bytes.NewReader(payload), "application/json", AuthOrbitNodeKey
	}
	return method, http.NoBody, "", AuthNone
}

func classifyResponse(c Check, resp *http.Response, bodyPeek []byte, latency time.Duration, authUsed AuthMode) Result {
	result := Result{Check: c, HTTPStatus: resp.StatusCode, Latency: latency}

	switch resp.StatusCode {
	case http.StatusForbidden:
		result.Status = StatusForbidden
		return result
	case http.StatusNotFound:
		result.Status = StatusNotFound
		return result
	}

	fleetish := c.Fingerprint == FingerprintNone || fingerprintMatches(c.Fingerprint, resp, bodyPeek)

	// An authenticated orbit probe that doesn't get 200 back means the
	// server responded but rejected our credentials. If the response still
	// looks like Fleet, point the user at enrollment (stale/revoked node
	// key) rather than at a captive portal.
	if authUsed == AuthOrbitNodeKey && resp.StatusCode != http.StatusOK {
		if fleetish {
			result.Status = StatusReachable
			result.Error = fmt.Sprintf("authenticated probe rejected with HTTP %d (stale orbit node key?)", resp.StatusCode)
			return result
		}
		result.Status = StatusNotFleet
		result.Error = fmt.Sprintf("authenticated probe rejected with HTTP %d", resp.StatusCode)
		return result
	}

	if !fleetish {
		result.Status = StatusNotFleet
		result.Error = "response did not match Fleet fingerprint"
		return result
	}

	result.Status = StatusReachable
	return result
}

func fingerprintMatches(mode FingerprintMode, resp *http.Response, bodyPeek []byte) bool {
	if mode&FingerprintCapabilitiesHeader != 0 {
		if resp.Header.Get(fleet.CapabilitiesHeader) != "" {
			return true
		}
	}
	if mode&FingerprintFleetJSONError != 0 {
		if looksLikeFleetJSONError(bodyPeek) {
			return true
		}
	}
	if mode&FingerprintFleetHTMLTitle != 0 {
		if looksLikeFleetHTMLTitle(bodyPeek) {
			return true
		}
	}
	return false
}

// looksLikeFleetJSONError reports whether body decodes as an object with
// Fleet's characteristic "message" + "errors" shape.
func looksLikeFleetJSONError(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	_, hasMessage := parsed["message"]
	_, hasErrors := parsed["errors"]
	return hasMessage && hasErrors
}

// looksLikeFleetHTMLTitle reports whether body contains Fleet's HTML title
// tag. Fleet's templates emit <title>Fleet</title> verbatim; we also accept
// <title>Fleet ...</title> to cover any future page-suffixed variants. Match
// is case-insensitive but not whitespace-tolerant — Fleet does not emit
// whitespace or attributes inside the title tag.
func looksLikeFleetHTMLTitle(body []byte) bool {
	lower := bytes.ToLower(body)
	return bytes.Contains(lower, []byte("<title>fleet</title>")) ||
		bytes.Contains(lower, []byte("<title>fleet "))
}

// classifyNetErr returns a short, human-readable reason for a transport error.
// We unwrap url.Error wrappers so the message is the underlying cause.
func classifyNetErr(err error) string {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		if urlErr.Timeout() {
			return "timeout"
		}
		if urlErr.Err != nil {
			return urlErr.Err.Error()
		}
	}
	return err.Error()
}
