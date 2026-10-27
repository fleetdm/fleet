// Package mock stands in for fleetdm.com's Google conditional access API and for the parts
// of Google's Cloud Identity API the Fleet server calls, so the whole flow runs without a
// Google Workspace tenant or Fleet's allowlisted OAuth client.
//
// fleetdm.com surface (/api/v1/google-conditional-access/...): registration, the iOS sign-in
// and callback pages, and the Android device register route. In fake mode the mock signs the
// end user in itself and derives the Google device from the host UUID. In forwarding mode
// (Config.GoogleClientID and GoogleClientSecret set) it runs the real Google OAuth flow and
// calls the real deviceUsers.lookup and devices.get.
//
// Cloud Identity surface (/v1/...): devices.list, devices.get, and
// devices.deviceUsers.clientStates.patch, always fake. Point the Fleet server's Cloud Identity
// endpoint at the mock's base URL. The service account token request goes to the key's
// token_uri, not the API endpoint, so the mock also serves POST /token: either use a key from
// GET /mock/service-account-key, whose token_uri is the mock, or have the client set the JWT
// config's TokenURL to <endpoint>/token. JWT signatures are not verified.
//
// Mock-only routes live under /mock/: a state dump, a key generator, and switches that make a
// device stale or an Android lookup find nothing.
package mock

import (
	"cmp"
	"crypto/rand"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

const (
	// defaultOAuthClientID is returned on registration in fake mode, where there is no real client.
	defaultOAuthClientID = "mock-client-id"
	// fakeEmail is the end user's email in fake mode when the sign-in page doesn't send one.
	fakeEmail = "user@example.com"
	// signInTTL bounds how long a sign-in can sit on Google's page before the callback.
	signInTTL = 10 * time.Minute
)

// Config configures the mock. The zero value runs fake mode with every customer accepted.
type Config struct {
	// FleetServerURL is where the callback sends the inbound register call. Empty uses the
	// Origin the Fleet server registered with, as fleetdm.com does.
	FleetServerURL string
	// GoogleClientID and GoogleClientSecret are a real Google Web OAuth client. Setting both
	// turns on forwarding mode.
	GoogleClientID     string
	GoogleClientSecret string
	// PartnerID is sent as the partner parameter of the iOS lookup in forwarding mode.
	PartnerID string
	// PublicURL is the mock's externally reachable base URL, used for the OAuth redirect URI
	// and the generated key's token_uri. Empty derives it from each request.
	PublicURL string
	// Customers are the Google customer IDs devices.list accepts besides registered ones.
	// Empty accepts every customer, since the Fleet server checks delegation before it
	// registers.
	Customers []string
	// Latency and ErrorRate simulate a slow or flaky upstream on server-to-server routes.
	Latency   time.Duration
	ErrorRate float64
	// HTTPClient is used to call the Fleet server and Google.
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// Server is the mock's HTTP handler. All state is in memory.
type Server struct {
	cfg        Config
	mux        *http.ServeMux
	logger     *slog.Logger
	httpClient *http.Client

	mu              sync.Mutex
	registrations   map[string]*Registration // by customer ID
	signIns         map[string]*signIn       // by OAuth state
	devices         map[string]*Device       // by device ID
	clientStates    map[string]*ClientState  // by resource name
	tokens          map[string]string        // access token -> subject
	androidNoRecord map[string]struct{}
}

type signIn struct {
	customerID  string
	token       string
	hostUUID    string
	redirectURL string
	expiresAt   time.Time
}

// State is the dump served at GET /mock/state.
type State struct {
	Registrations []Registration `json:"registrations"`
	Devices       []Device       `json:"devices"`
	ClientStates  []ClientState  `json:"client_states"`
}

// Registration is a Fleet server registered with the fleetdm.com surface. There is no admin
// consent step, so a registration is complete as soon as it is created.
type Registration struct {
	CustomerID        string `json:"customer_id"`
	Origin            string `json:"origin"`
	FleetServerSecret string `json:"fleet_server_secret"`
	OAuthClientID     string `json:"oauth_client_id"`
}

// Device is a device in the mock's Google inventory, added when an end user signs in.
type Device struct {
	Name           string `json:"name"`
	DeviceID       string `json:"device_id"`
	DeviceUserName string `json:"device_user_name"`
	CustomerID     string `json:"customer_id"`
	Email          string `json:"email"`
	HostUUID       string `json:"host_uuid"`
	AndroidID      string `json:"android_id,omitempty"`
	// Stale devices answer 404, as Google does after a factory reset or admin deletion.
	Stale bool `json:"stale"`
}

// ClientState is a partner client state written through clientStates.patch.
type ClientState struct {
	Name      string    `json:"name"`
	Managed   string    `json:"managed"`
	CustomID  string    `json:"custom_id"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// New returns a mock with empty state.
func New(cfg Config) *Server {
	s := &Server{
		cfg:             cfg,
		logger:          cmp.Or(cfg.Logger, slog.New(slog.DiscardHandler)),
		httpClient:      cfg.HTTPClient,
		registrations:   make(map[string]*Registration),
		signIns:         make(map[string]*signIn),
		devices:         make(map[string]*Device),
		clientStates:    make(map[string]*ClientState),
		tokens:          make(map[string]string),
		androidNoRecord: make(map[string]struct{}),
	}
	if s.httpClient == nil {
		s.httpClient = fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second))
	}
	s.mux = s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) forwarding() bool {
	return s.cfg.GoogleClientID != "" && s.cfg.GoogleClientSecret != ""
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	sim := func(h http.HandlerFunc) http.HandlerFunc {
		return simulateLatencyAndErrors(s.cfg.Latency, s.cfg.ErrorRate, h)
	}

	mux.HandleFunc("GET /mock/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /mock/state", s.handleState)
	mux.HandleFunc("GET /mock/service-account-key", s.handleServiceAccountKey)
	mux.HandleFunc("POST /mock/devices/{deviceID}/stale", s.handleStale)
	mux.HandleFunc("POST /mock/android/{androidID}/no-record", s.handleAndroidNoRecord)

	// fleetdm.com
	mux.HandleFunc("POST /api/v1/google-conditional-access", sim(s.handleCreate))
	mux.HandleFunc("DELETE /api/v1/google-conditional-access", sim(s.handleDelete))
	mux.HandleFunc("GET /api/v1/google-conditional-access/sign-in", s.handleSignIn)
	mux.HandleFunc("GET /api/v1/google-conditional-access/callback", s.handleCallback)
	mux.HandleFunc("POST /api/v1/google-conditional-access/device/register", sim(s.handleAndroidRegister))
	mux.HandleFunc("GET /google-conditional-access/{page}", s.handleResultPage)

	// Google
	mux.HandleFunc("POST /token", s.handleToken)
	mux.HandleFunc("GET /v1/devices", sim(s.authed(s.handleDevicesList)))
	mux.HandleFunc("GET /v1/devices/{device}", sim(s.authed(s.handleDevicesGet)))
	mux.HandleFunc("PATCH /v1/devices/{device}/deviceUsers/{user}/clientStates/{state}", sim(s.authed(s.handleClientStatePatch)))

	return mux
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	st := State{
		Registrations: derefSorted(s.registrations, func(r Registration) string { return r.CustomerID }),
		Devices:       derefSorted(s.devices, func(d Device) string { return d.DeviceID }),
		ClientStates:  derefSorted(s.clientStates, func(c ClientState) string { return c.Name }),
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, st)
}

// derefSorted copies the map's values out from under the lock in a stable order.
func derefSorted[T any](m map[string]*T, key func(T) string) []T {
	out := make([]T, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	slices.SortFunc(out, func(a, b T) int { return strings.Compare(key(a), key(b)) })
	return out
}

func (s *Server) handleStale(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[r.PathValue("deviceID")]
	if !ok {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	d.Stale = true
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAndroidNoRecord(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.androidNoRecord[r.PathValue("androidID")] = struct{}{}
	w.WriteHeader(http.StatusOK)
}

// knownCustomer reports whether the Cloud Identity surface answers for customerID.
// Callers hold s.mu.
func (s *Server) knownCustomer(customerID string) bool {
	if len(s.cfg.Customers) == 0 {
		return true
	}
	if _, ok := s.registrations[customerID]; ok {
		return true
	}
	return slices.Contains(s.cfg.Customers, customerID)
}

// upsertDevice adds a signed-in device to the inventory, reviving it if it was stale.
func (s *Server) upsertDevice(d Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[d.DeviceID] = &d
}

// fakeDevice derives the Google device for a host in fake mode.
func fakeDevice(hostUUID string) (deviceID, deviceUserName string) {
	deviceID = "mock-" + hostUUID[:min(12, len(hostUUID))]
	return deviceID, "devices/" + deviceID + "/deviceUsers/" + deviceID
}

// baseURL is the mock's own externally reachable URL.
func (s *Server) baseURL(r *http.Request) string {
	if s.cfg.PublicURL != "" {
		return strings.TrimSuffix(s.cfg.PublicURL, "/")
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

const secretAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomSecret(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = secretAlphabet[int(b[i])%len(secretAlphabet)]
	}
	return string(b)
}
