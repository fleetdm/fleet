// Package mockserver stands in for fleetdm.com's Google conditional access relay and for
// Google's Cloud Identity API, for development and tests. Either surface can be served
// alone: the Fleet server points its relay at the mock with
// google_conditional_access.proxy_uri and its Cloud Identity calls with
// FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT, independently.
package mockserver

import (
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// Config configures a Server.
type Config struct {
	// Relay serves fleetdm.com's routes.
	Relay bool
	// Google serves the Cloud Identity API and its token endpoint.
	Google bool

	// PublicURL is the URL devices and Google reach the mock at. The sign-in page and
	// Google's redirect URI are built from it.
	PublicURL string
	// FleetURL, when set, is called back instead of the Origin the Fleet server
	// registered with.
	FleetURL string
	// FleetClient calls the Fleet server back. Defaults to a fleethttp client.
	FleetClient *http.Client

	// OAuth, when set, runs the sign-ins against real Google with Fleet's Web client.
	// Otherwise the lookups are fake: the iOS sign-in page offers the outcomes and the
	// DeviceUser names are derived from the host UUID.
	OAuth *OAuthConfig

	// CustomerIDs are the Google customers the Cloud Identity surface knows; devices list
	// is a 403 for any other, so the delegation check can be failed. With or without "C".
	CustomerIDs []string
	// FakeEmail is the signed-in user of a fake lookup.
	FakeEmail string
	// FakeDeviceRecords is how many device records a fake iOS lookup finds for the host,
	// one DeviceUser each: 2 adds a leftover record, as after a wipe and re-enrollment.
	FakeDeviceRecords int
	// AutoContinue, when positive, has the fake sign-in page continue as signed in by
	// itself after that long, rounded down to seconds.
	AutoContinue time.Duration

	// Latency delays every API call by 50% to 150% of it.
	Latency time.Duration
	// ErrorRate is the share of API calls that fail with a 429 or a 500.
	ErrorRate float64

	Logger *slog.Logger
}

// Fake authorization codes that pick the outcome of a fake lookup.
const (
	// CodeSignedIn finds the device: the default.
	CodeSignedIn = "mock-signed-in"
	// CodeNotFound finds no DeviceUser, as when the user hasn't signed in to Gmail.
	CodeNotFound = "mock-not-found"
	// CodeRefused is Google refusing the code exchange.
	CodeRefused = "mock-refused"
)

// maxCalls is how many Cloud Identity calls the state keeps, the latest ones.
const maxCalls = 1000

// Server is the mock. Its state lives in memory.
type Server struct {
	cfg    Config
	logger *slog.Logger
	mux    *http.ServeMux

	mu            sync.Mutex
	registrations map[string]*registration // by customer ID with its "C"
	signIns       map[string]*pendingSignIn
	customers     map[string]struct{} // bare customer IDs
	devices       map[string]*device  // by resource ID
	tokenError    string              // OAuth error code the token endpoint answers with
	calls         []string            // "METHOD path" of the latest Cloud Identity calls
}

func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.DiscardHandler)
	}
	if cfg.FleetClient == nil {
		cfg.FleetClient = fleethttp.NewClient(fleethttp.WithFollowRedir(false))
	}
	if cfg.FakeEmail == "" {
		cfg.FakeEmail = "user@example.com"
	}
	if cfg.FakeDeviceRecords < 1 {
		cfg.FakeDeviceRecords = 1
	}
	cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")
	cfg.FleetURL = strings.TrimRight(cfg.FleetURL, "/")
	s := &Server{cfg: cfg, logger: cfg.Logger, mux: http.NewServeMux()}
	s.resetLocked()
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.DebugContext(r.Context(), "request", "method", r.Method, "path", r.URL.Path)
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	if s.cfg.Relay {
		s.relayRoutes()
	}
	if s.cfg.Google {
		s.googleRoutes()
	}
	s.mux.HandleFunc("GET /_mock/state", s.admin(s.handleState))
	s.mux.HandleFunc("POST /_mock/reset", s.admin(s.handleReset))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.logger.WarnContext(r.Context(), "no such route; is the surface it belongs to enabled?", "method", r.Method, "path", r.URL.Path)
		http.NotFound(w, r)
	})
}

// admin serves a route to local callers only. The state holds the secrets that
// authenticate the calls to the Fleet server, and the mock may be behind a public tunnel:
// a tunnel's requests come from loopback too, but under the tunnel's host name.
func (s *Server) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r.RemoteAddr) || !isLoopback(r.Host) {
			s.logger.WarnContext(r.Context(), "refused an admin route to a non-local caller", "path", r.URL.Path, "host", r.Host)
			http.Error(w, "admin routes are served on localhost only", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func isLoopback(hostPort string) bool {
	host := hostPort
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (s *Server) resetLocked() {
	s.registrations = map[string]*registration{}
	s.signIns = map[string]*pendingSignIn{}
	s.customers = map[string]struct{}{}
	for _, c := range s.cfg.CustomerIDs {
		s.customers[bareCustomerID(c)] = struct{}{}
	}
	s.devices = map[string]*device{}
	s.tokenError = ""
	s.calls = nil
}

func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.resetLocked()
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// State is the mock's state, as GET /_mock/state returns it.
type State struct {
	Registrations []Registration `json:"registrations"`
	Customers     []string       `json:"customers"`
	Devices       []Device       `json:"devices"`
	TokenError    string         `json:"token_error,omitempty"`
	Calls         []string       `json:"calls"`
}

// Registration is a Fleet server registered with the relay.
type Registration struct {
	CustomerID string `json:"customer_id"`
	Origin     string `json:"origin"`
	Secret     string `json:"secret"`
}

// State returns a copy of the mock's state.
func (s *Server) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := State{Calls: append([]string{}, s.calls...), TokenError: s.tokenError}
	for _, r := range s.registrations {
		st.Registrations = append(st.Registrations, Registration{CustomerID: r.customerID, Origin: r.origin, Secret: r.secret})
	}
	for c := range s.customers {
		st.Customers = append(st.Customers, c)
	}
	for _, d := range s.devices {
		st.Devices = append(st.Devices, d.export())
	}
	sortState(&st)
	return st
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.State())
}

// simulate applies the configured latency and error rate to an API call. fail writes the
// simulated failure in the shape of the surface the call belongs to.
func (s *Server) simulate(next http.HandlerFunc, fail func(w http.ResponseWriter, rateLimited bool)) http.HandlerFunc {
	if s.cfg.Latency <= 0 && s.cfg.ErrorRate <= 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Latency > 0 {
			time.Sleep(time.Duration(float64(s.cfg.Latency) * (0.5 + rand.Float64()))) //nolint:gosec // mock
		}
		if rand.Float64() < s.cfg.ErrorRate { //nolint:gosec // mock
			fail(w, rand.Float64() < 0.5) //nolint:gosec // mock
			return
		}
		next(w, r)
	}
}

func failGoogleAPI(w http.ResponseWriter, rateLimited bool) {
	if rateLimited {
		writeGoogleError(w, http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", "simulated rate limit")
		return
	}
	writeGoogleError(w, http.StatusInternalServerError, "INTERNAL", "simulated server error")
}

// failToken is always a 5xx: the Fleet server reads a 4xx on the token exchange as a
// broken delegation, which a simulated outage isn't.
func failToken(w http.ResponseWriter, _ bool) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "temporarily_unavailable", "error_description": "simulated server error"})
}

func failRelay(w http.ResponseWriter, rateLimited bool) {
	if rateLimited {
		http.Error(w, "simulated rate limit", http.StatusTooManyRequests)
		return
	}
	http.Error(w, "simulated server error", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeGoogleError writes Google's JSON error shape, which the Google client libraries parse.
func writeGoogleError(w http.ResponseWriter, code int, status, message string) {
	writeJSON(w, code, map[string]any{"error": map[string]any{"code": code, "message": message, "status": status}})
}

func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v); err != nil {
		return errors.New("invalid JSON body: " + err.Error())
	}
	return nil
}

// jwtClaims decodes a JWT's payload into v without checking the signature. v is left
// as is when raw isn't a JWT.
func jwtClaims(raw string, v any) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return
	}
	if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
		_ = json.Unmarshal(b, v)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
}

func bareCustomerID(id string) string {
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(id), "customers/"), "C")
}
