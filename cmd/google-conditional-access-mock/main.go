// Command google-conditional-access-mock stands in for fleetdm.com's Google conditional access
// API and Google's Cloud Identity API, for developing and testing Google conditional access
// without a Workspace tenant. See README.md and the mock package.
//
// Usage:
//
//	google-conditional-access-mock --listen :9998
//	google-conditional-access-mock --listen :9998 --google-client-id <id> --google-client-secret <secret> --google-partner-id <id>
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/google-conditional-access-mock/mock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "exiting", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	listen := flag.String("listen", ":9998", "host:port to listen on")
	fleetServerURL := flag.String("fleet-server-url", "", "Fleet server URL for the inbound register call (default: the Origin the Fleet server registered with)")
	publicURL := flag.String("public-url", "", "the mock's externally reachable URL, for the OAuth redirect URI and generated keys (default: derived from each request)")
	googleClientID := flag.String("google-client-id", "", "real Google Web OAuth client ID; with --google-client-secret, turns on forwarding mode")
	googleClientSecret := flag.String("google-client-secret", "", "real Google Web OAuth client secret (or GOOGLE_CLIENT_SECRET)")
	partnerID := flag.String("google-partner-id", "", "Fleet's partner ID, sent on the iOS lookup in forwarding mode")
	customers := flag.String("customers", "", "comma-separated Google customer IDs devices.list accepts besides registered ones (default: accept any)")
	latency := flag.Duration("latency", 0, "mean latency added to server-to-server responses")
	errorRate := flag.Float64("error-rate", 0, "fraction of server-to-server requests that fail with 429 or 500 [0, 1]")
	insecure := flag.Bool("insecure", false, "skip TLS verification when calling the Fleet server (self-signed dev certificates)")
	flag.Parse()

	secret := *googleClientSecret
	if secret == "" {
		secret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	if (*googleClientID == "") != (secret == "") {
		return errors.New("--google-client-id and --google-client-secret must be set together")
	}
	if *errorRate < 0 || *errorRate > 1 {
		return errors.New("--error-rate must be between 0 and 1")
	}
	if *googleClientID != "" && *partnerID == "" {
		logger.WarnContext(ctx, "forwarding mode without --google-partner-id: iOS lookups will find nothing")
	}

	var customerIDs []string
	for c := range strings.SplitSeq(*customers, ",") {
		if c = strings.TrimSpace(c); c != "" {
			customerIDs = append(customerIDs, c)
		}
	}

	clientOpts := []fleethttp.ClientOpt{fleethttp.WithTimeout(30 * time.Second)}
	if *insecure {
		clientOpts = append(clientOpts, fleethttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true})) //nolint:gosec // opt-in for local dev certificates
	}

	handler := mock.New(mock.Config{
		FleetServerURL:     *fleetServerURL,
		GoogleClientID:     *googleClientID,
		GoogleClientSecret: secret,
		PartnerID:          *partnerID,
		PublicURL:          *publicURL,
		Customers:          customerIDs,
		Latency:            *latency,
		ErrorRate:          *errorRate,
		HTTPClient:         fleethttp.NewClient(clientOpts...),
		Logger:             logger,
	})

	mode := "fake"
	if *googleClientID != "" {
		mode = "forwarding"
	}
	logger.InfoContext(ctx, "Google conditional access mock listening", "addr", *listen, "mode", mode)
	srv := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	return srv.ListenAndServe()
}
