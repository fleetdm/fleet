package connectivity

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"golang.org/x/sync/errgroup"
)

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
			results[i] = runOne(gCtx, client, base, c)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return results, err
	}
	return results, nil
}

func runOne(ctx context.Context, client *http.Client, base *url.URL, c Check) Result {
	target := *base
	target.Path = strings.TrimRight(base.Path, "/") + c.Path

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, c.Method, target.String(), http.NoBody)
	if err != nil {
		return Result{Check: c, Status: StatusBlocked, Latency: time.Since(start), Error: err.Error()}
	}
	req.Header.Set("User-Agent", "fleet-connectivity-probe")

	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Result{Check: c, Status: StatusBlocked, Latency: latency, Error: classifyNetErr(err)}
	}
	defer resp.Body.Close()
	// Drain a small amount so the connection can be reused.
	_, _ = io.CopyN(io.Discard, resp.Body, 4096)

	status := StatusReachable
	if resp.StatusCode == http.StatusNotFound {
		status = StatusNotFound
	}
	return Result{Check: c, Status: status, HTTPStatus: resp.StatusCode, Latency: latency}
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
