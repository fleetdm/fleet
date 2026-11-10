// Command google-conditional-access-mock stands in for fleetdm.com's Google conditional
// access relay, Google's Cloud Identity API, or both, so the Google conditional access
// flow can run without a Workspace tenant or an allowlisted OAuth client. See README.md.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google/mockserver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	listen := flag.String("listen", "localhost:9090", "address to listen on")
	publicURL := flag.String("public-url", "", "URL devices and Google reach the mock at (default http://<listen>)")
	mock := flag.String("mock", "fleetdm,google", "surfaces to serve: fleetdm (the fleetdm.com relay), google (Cloud Identity API), or both, comma-separated")
	fleetURL := flag.String("fleet-url", "", "Fleet server URL to call back (default: the Origin the server registered with)")
	fleetInsecure := flag.Bool("fleet-insecure", false, "don't verify the Fleet server's TLS certificate on the callback")
	clientID := flag.String("google-client-id", os.Getenv("GOOGLE_CA_MOCK_CLIENT_ID"), "Fleet's Web OAuth client ID: sign in with real Google instead of the fake sign-in (env GOOGLE_CA_MOCK_CLIENT_ID)")
	clientSecret := flag.String("google-client-secret", os.Getenv("GOOGLE_CA_MOCK_CLIENT_SECRET"), "the Web OAuth client's secret (env GOOGLE_CA_MOCK_CLIENT_SECRET)")
	partnerID := flag.String("partner-id", os.Getenv("GOOGLE_CA_MOCK_PARTNER_ID"), "Fleet's Google partner ID for the iOS lookup (env GOOGLE_CA_MOCK_PARTNER_ID)")
	customers := flag.String("customer-ids", "C01mock00", "Google customer IDs the Cloud Identity surface knows, comma-separated; any other fails the delegation check")
	email := flag.String("email", "user@example.com", "signed-in user of the fake sign-in")
	records := flag.Int("device-records", 1, "device records a fake iOS sign-in finds for the host (2 adds a leftover record)")
	autoContinue := flag.Duration("auto-continue", 2*time.Second, "delay before the fake sign-in page continues as signed in by itself; 0 waits for a click, to pick another outcome")
	latency := flag.Duration("latency", 0, "mean latency added to every API call")
	errorRate := flag.Float64("error-rate", 0, "share of API calls that fail with a 429 or a 500 (0 to 1)")
	debug := flag.Bool("debug", false, "log every request")
	newKey := flag.Bool("new-key", false, "print a new service account key JSON to upload in Fleet, and exit")
	flag.Parse()

	if *newKey {
		return printNewKey()
	}

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	cfg := mockserver.Config{
		PublicURL:         *publicURL,
		FleetURL:          *fleetURL,
		FakeEmail:         *email,
		FakeDeviceRecords: *records,
		AutoContinue:      *autoContinue,
		Latency:           *latency,
		ErrorRate:         *errorRate,
		Logger:            logger,
	}
	for s := range strings.SplitSeq(*mock, ",") {
		switch strings.TrimSpace(s) {
		case "fleetdm":
			cfg.Relay = true
		case "google":
			cfg.Google = true
		case "":
		default:
			return fmt.Errorf("--mock: unknown surface %q, want fleetdm and/or google", s)
		}
	}
	if !cfg.Relay && !cfg.Google {
		return errors.New("--mock: nothing to serve")
	}
	for c := range strings.SplitSeq(*customers, ",") {
		if c = strings.TrimSpace(c); c != "" {
			cfg.CustomerIDs = append(cfg.CustomerIDs, c)
		}
	}
	switch {
	case *clientID != "" && *clientSecret != "":
		cfg.OAuth = &mockserver.OAuthConfig{ClientID: *clientID, ClientSecret: *clientSecret, PartnerID: *partnerID}
		if *partnerID == "" {
			logger.WarnContext(context.Background(), "no --partner-id: iOS lookups will fail")
		}
	case *clientID != "" || *clientSecret != "":
		return errors.New("--google-client-id and --google-client-secret go together")
	}
	if !cfg.Relay && (cfg.OAuth != nil || *partnerID != "") {
		logger.WarnContext(context.Background(), "the Google OAuth client is only used by the fleetdm.com relay, which is off: ignoring it")
	}
	if cfg.Relay && !cfg.Google && cfg.OAuth == nil {
		logger.WarnContext(context.Background(), "serving fleetdm.com with fake sign-ins against real Google: the fake DeviceUser names don't exist in Google, so marking them fails. Pass --google-client-id and --google-client-secret, or add google to --mock")
	}
	if *errorRate < 0 || *errorRate > 1 {
		return errors.New("--error-rate must be between 0 and 1")
	}
	if cfg.PublicURL == "" {
		cfg.PublicURL = "http://" + *listen
		if host, port, err := net.SplitHostPort(*listen); err == nil {
			// An address that listens on every interface isn't one to connect to.
			if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
				cfg.PublicURL = "http://localhost:" + port
			}
		}
	}
	if *fleetInsecure {
		cfg.FleetClient = fleethttp.NewClient(
			fleethttp.WithFollowRedir(false),
			fleethttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true}), //nolint:gosec // dev mock, opt-in
		)
	}

	printUsage(cfg, *listen)
	srv := &http.Server{Addr: *listen, Handler: mockserver.New(cfg), ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

// printNewKey prints a service account key the mock accepts: Fleet signs the token
// request with it, so it must hold a real RSA key, but the mock doesn't check the signature.
func printNewKey() error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]string{ //nolint:gosec // a throwaway key generated above
		"type":           "service_account",
		"project_id":     "google-conditional-access-mock",
		"private_key_id": "mock",
		"private_key":    string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"client_email":   "fleet@google-conditional-access-mock.iam.gserviceaccount.com",
		"client_id":      "100000000000000000000",
		"token_uri":      "https://oauth2.googleapis.com/token",
	})
}

func printUsage(cfg mockserver.Config, listen string) {
	signIn := "fake, as " + cfg.FakeEmail
	if cfg.OAuth != nil {
		signIn = "real Google with client " + cfg.OAuth.ClientID + ", redirect URI " + cfg.PublicURL + "/api/v1/google-conditional-access/callback"
	}
	fmt.Printf("Google conditional access mock listening on %s (public URL %s)\n", listen, cfg.PublicURL)
	if cfg.Relay {
		fmt.Printf("  fleetdm.com relay: on, sign-in %s\n", signIn)
		fmt.Printf("    FLEET_GOOGLE_CONDITIONAL_ACCESS_PROXY_URI=%s fleet serve --dev\n", cfg.PublicURL)
	} else {
		fmt.Println("  fleetdm.com relay: off (Fleet uses its configured google_conditional_access.proxy_uri)")
	}
	if cfg.Google {
		fmt.Printf("  Cloud Identity API: on, customers %s\n", strings.Join(cfg.CustomerIDs, ", "))
		fmt.Printf("    FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT=%s/ fleet serve --dev\n", cfg.PublicURL)
	} else {
		fmt.Println("  Cloud Identity API: off (Fleet calls real Google)")
	}
	local := "localhost" + listen
	if _, port, err := net.SplitHostPort(listen); err == nil {
		local = "localhost:" + port
	}
	fmt.Printf("  State, from this machine only: curl http://%s/_mock/state\n", local)
}
