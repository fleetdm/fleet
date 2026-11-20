package mockserver_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	cag "github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google/mockserver"
	"github.com/stretchr/testify/require"
)

const customer = "C02mock00"

// fakeFleet is the Fleet server's inbound register route.
type fakeFleet struct {
	*httptest.Server
	mu     sync.Mutex
	calls  []fleetCall
	status int
	result string
}

type fleetCall struct {
	Auth string
	Body struct {
		Token    string   `json:"token"`
		Names    []string `json:"names"`
		Customer string   `json:"customer"`
		Email    string   `json:"email"`
	}
}

func newFakeFleet(t *testing.T) *fakeFleet {
	f := &fakeFleet{status: http.StatusOK, result: "done"}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/fleet/google_conditional_access/register" {
			http.NotFound(w, r)
			return
		}
		var c fleetCall
		c.Auth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&c.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, c)
		w.WriteHeader(f.status)
		_ = json.NewEncoder(w).Encode(map[string]string{"result": f.result})
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeFleet) answer(status int, result string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status, f.result, f.calls = status, result, nil
}

func (f *fakeFleet) lastCall(t *testing.T) fleetCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotEmpty(t, f.calls)
	return f.calls[len(f.calls)-1]
}

type env struct {
	mock  *mockserver.Server
	srv   *httptest.Server
	relay cag.Relay
	http  *http.Client
}

func newEnv(t *testing.T, cfg mockserver.Config) *env {
	cfg.Logger = slog.New(slog.DiscardHandler)
	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	t.Cleanup(srv.Close)
	cfg.PublicURL = srv.URL
	m := mockserver.New(cfg)
	handler = m
	return &env{
		mock:  m,
		srv:   srv,
		relay: cag.NewRelay(srv.URL, cag.AllowHTTP()),
		http:  fleethttp.NewClient(fleethttp.WithFollowRedir(false)),
	}
}

func (e *env) get(t *testing.T, rawURL string) (*http.Response, string) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	res, err := e.http.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res, string(b)
}

func relayStatus(t *testing.T, err error) int {
	se, ok := errors.AsType[*cag.RelayStatusError](err)
	require.True(t, ok, "%v", err)
	return se.Code
}

func TestRelayRegistration(t *testing.T) {
	e := newEnv(t, mockserver.Config{Relay: true})
	ctx := t.Context()

	reg, err := e.relay.Register(ctx, "https://fleet-a.example.com", customer)
	require.NoError(t, err)
	require.NotEmpty(t, reg.FleetServerSecret)

	_, err = e.relay.Register(ctx, "https://fleet-b.example.com", customer)
	require.Equal(t, http.StatusConflict, relayStatus(t, err))

	again, err := e.relay.Register(ctx, "https://fleet-a.example.com", customer)
	require.NoError(t, err)
	require.NotEqual(t, reg.FleetServerSecret, again.FleetServerSecret)

	err = e.relay.Deregister(ctx, "https://fleet-a.example.com", customer, reg.FleetServerSecret)
	require.True(t, fleet.IsNotFound(err), "%v", err)
	require.NoError(t, e.relay.Deregister(ctx, "https://fleet-a.example.com", customer, again.FleetServerSecret))
	require.Empty(t, e.mock.State().Registrations)
}

var continueLink = regexp.MustCompile(`id="continue" href="([^"]+)"`)

// signIn opens the sign-in URL and returns the fake sign-in page's link for code.
func (e *env) signIn(t *testing.T, hostUUID, token, code string) string {
	signInURL, err := e.relay.SignInURL(customer, hostUUID, token)
	require.NoError(t, err)
	res, body := e.get(t, signInURL)
	require.Equal(t, http.StatusOK, res.StatusCode, body)
	m := continueLink.FindStringSubmatch(body)
	require.Len(t, m, 2, body)
	u, err := url.Parse(html.UnescapeString(m[1]))
	require.NoError(t, err)
	q := u.Query()
	q.Set("code", code)
	u.RawQuery = q.Encode()
	return u.String()
}

func TestIOSSignInFake(t *testing.T) {
	fl := newFakeFleet(t)
	e := newEnv(t, mockserver.Config{Relay: true, Google: true, CustomerIDs: []string{customer}, FakeEmail: "ann@example.com"})

	signInURL, err := e.relay.SignInURL(customer, "host-1", "tok")
	require.NoError(t, err)
	res, _ := e.get(t, signInURL)
	require.Equal(t, http.StatusNotFound, res.StatusCode, "no registration yet")

	reg, err := e.relay.Register(t.Context(), fl.URL, customer)
	require.NoError(t, err)

	cases := []struct {
		name       string
		status     int
		result     string
		code       string
		wantPage   string
		wantNames  int
		wantCalled bool
	}{
		{"done", http.StatusOK, "done", mockserver.CodeSignedIn, "/google-conditional-access/done", 1, true},
		{"not allowed", http.StatusOK, "not_allowed", mockserver.CodeSignedIn, "/google-conditional-access/not-allowed", 1, true},
		{"not found", http.StatusOK, "not_found", mockserver.CodeNotFound, "/google-conditional-access/not-found", 0, true},
		{"error", http.StatusOK, "error", mockserver.CodeSignedIn, "/google-conditional-access/error", 1, true},
		{"bad secret", http.StatusUnauthorized, "", mockserver.CodeSignedIn, "/google-conditional-access/error", 1, true},
		{"refused", http.StatusOK, "done", mockserver.CodeRefused, "/google-conditional-access/error", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fl.answer(c.status, c.result)
			callback := e.signIn(t, "host-1", "tok-"+c.name, c.code)
			res, body := e.get(t, callback)
			require.Equal(t, http.StatusFound, res.StatusCode, body)
			loc, err := url.Parse(res.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, c.wantPage, loc.Path)

			res, body = e.get(t, e.srv.URL+loc.String())
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Contains(t, body, "<h1>")

			if !c.wantCalled {
				fl.mu.Lock()
				require.Empty(t, fl.calls)
				fl.mu.Unlock()
				return
			}
			call := fl.lastCall(t)
			require.Equal(t, "Bearer "+reg.FleetServerSecret, call.Auth)
			require.Equal(t, "tok-"+c.name, call.Body.Token)
			require.Equal(t, customer, call.Body.Customer)
			require.Equal(t, "ann@example.com", call.Body.Email)
			require.Len(t, call.Body.Names, c.wantNames)
			for _, n := range call.Body.Names {
				require.Regexp(t, `^devices/[^/]+/deviceUsers/[^/]+$`, n)
			}

			// The state is single use.
			res, _ = e.get(t, callback)
			require.Equal(t, "/google-conditional-access/error", strings.SplitN(res.Header.Get("Location"), "?", 2)[0])
		})
	}

	// The same host gets the same names, and they exist on the Google surface.
	fl.answer(http.StatusOK, "done")
	res, _ = e.get(t, e.signIn(t, "host-1", "tok-a", mockserver.CodeSignedIn))
	require.Equal(t, http.StatusFound, res.StatusCode)
	first := fl.lastCall(t).Body.Names
	res, _ = e.get(t, e.signIn(t, "host-1", "tok-b", mockserver.CodeSignedIn))
	require.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, first, fl.lastCall(t).Body.Names)
	st := e.mock.State()
	require.Len(t, st.Devices, 1)
	require.Equal(t, first[0], st.Devices[0].Users[0].Name)
}

func TestIOSSignInTwoRecords(t *testing.T) {
	fl := newFakeFleet(t)
	e := newEnv(t, mockserver.Config{Relay: true, Google: true, CustomerIDs: []string{customer}, FakeDeviceRecords: 2})
	_, err := e.relay.Register(t.Context(), fl.URL, customer)
	require.NoError(t, err)
	res, _ := e.get(t, e.signIn(t, "host-2", "tok", mockserver.CodeSignedIn))
	require.Equal(t, http.StatusFound, res.StatusCode)
	names := fl.lastCall(t).Body.Names
	require.Len(t, names, 2)
	require.NotEqual(t, strings.Split(names[0], "/")[1], strings.Split(names[1], "/")[1], "names on two device records")
}

func newKey(t *testing.T) []byte {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	b, err := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "sa@example.iam.gserviceaccount.com",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	})
	require.NoError(t, err)
	return b
}

func (e *env) cloudIdentity(t *testing.T, customerID string) cag.CloudIdentity {
	dev_mode.SetOverride(cag.EndpointOverrideEnv, e.srv.URL+"/", t)
	ci, err := cag.NewCloudIdentityFactory(slog.New(slog.DiscardHandler))(t.Context(), cag.Credentials{
		ServiceAccountKey: newKey(t), AdminEmail: "admin@example.com", CustomerID: customerID,
	})
	require.NoError(t, err)
	require.NotNil(t, ci)
	return ci
}

func clientState(t *testing.T, m *mockserver.Server, name string) *mockserver.ClientState {
	for _, d := range m.State().Devices {
		for _, u := range d.Users {
			if u.Name == name {
				cs := u.ClientStates[strings.TrimPrefix(customer, "C")+"-fleet"]
				require.NotNil(t, cs, name)
				return cs
			}
		}
	}
	t.Fatalf("no device user %s", name)
	return nil
}

func TestCloudIdentity(t *testing.T) {
	e := newEnv(t, mockserver.Config{Google: true, CustomerIDs: []string{customer}})
	ctx := t.Context()
	ci := e.cloudIdentity(t, customer)

	require.NoError(t, ci.CheckDelegation(ctx))
	err := e.cloudIdentity(t, "C0unknown").CheckDelegation(ctx)
	require.True(t, cag.IsAuthError(err), "%v", err)

	d := e.mock.Seed(mockserver.SeedDevice{CustomerID: customer, DeviceType: "IOS", Emails: []string{"a@example.com", "b@example.com"}})
	deviceID := strings.TrimPrefix(d.Name, "devices/")
	names, err := ci.ListDeviceUserNames(ctx, deviceID)
	require.NoError(t, err)
	require.Equal(t, []string{d.Users[0].Name, d.Users[1].Name}, names)

	require.NoError(t, ci.SetClientState(ctx, names[0], fleet.GoogleConditionalAccessClientState{Managed: new(true), CustomID: new("host-uuid")}))
	cs := clientState(t, e.mock, names[0])
	require.Equal(t, "MANAGED", cs.Managed)
	require.Equal(t, "host-uuid", cs.CustomID)
	require.NoError(t, ci.SetClientState(ctx, names[0], fleet.GoogleConditionalAccessClientState{Managed: new(false)}))
	cs = clientState(t, e.mock, names[0])
	require.Equal(t, "UNMANAGED", cs.Managed)
	require.Equal(t, "host-uuid", cs.CustomID, "a field outside the mask is kept")

	// A name the mock doesn't know, as real Google returned it to a real relay.
	require.NoError(t, ci.SetClientState(ctx, "devices/real1/deviceUsers/u1", fleet.GoogleConditionalAccessClientState{Managed: new(true)}))
	require.Equal(t, "MANAGED", clientState(t, e.mock, "devices/real1/deviceUsers/u1").Managed)

	ok, err := ci.DeviceExists(ctx, deviceID)
	require.NoError(t, err)
	require.True(t, ok)

	require.True(t, e.mock.SetDeviceState(deviceID, mockserver.DeviceDeleted))
	ok, err = ci.DeviceExists(ctx, deviceID)
	require.NoError(t, err)
	require.False(t, ok)
	_, err = ci.ListDeviceUserNames(ctx, deviceID)
	require.True(t, fleet.IsNotFound(err), "%v", err)
	err = ci.SetClientState(ctx, names[0], fleet.GoogleConditionalAccessClientState{Managed: new(true)})
	require.True(t, cag.IsStale(err), "%v", err)

	require.True(t, e.mock.SetDeviceState(deviceID, mockserver.DeviceLeft))
	_, err = ci.DeviceExists(ctx, deviceID)
	require.True(t, cag.IsStale(err), "%v", err)
	require.False(t, cag.IsAuthError(err))

	e.mock.SetTokenError("unauthorized_client")
	err = e.cloudIdentity(t, customer).CheckDelegation(ctx)
	require.True(t, cag.IsAuthError(err), "%v", err)
	require.Contains(t, err.Error(), "unauthorized_client")
}

func TestSurfaces(t *testing.T) {
	relayOnly := newEnv(t, mockserver.Config{Relay: true})
	_, err := relayOnly.cloudIdentity(t, customer).ListDeviceUserNames(t.Context(), "d")
	require.Error(t, err)
	_, err = relayOnly.relay.Register(t.Context(), "https://fleet.example.com", customer)
	require.NoError(t, err)

	googleOnly := newEnv(t, mockserver.Config{Google: true, CustomerIDs: []string{customer}})
	_, err = googleOnly.relay.Register(t.Context(), "https://fleet.example.com", customer)
	require.Equal(t, http.StatusNotFound, relayStatus(t, err))
	require.NoError(t, googleOnly.cloudIdentity(t, customer).CheckDelegation(t.Context()))
}

// fakeGoogle is real Google's OAuth and lookup endpoints, for forwarding mode.
func fakeGoogle(t *testing.T) (*httptest.Server, func() []url.Values) {
	var mu sync.Mutex
	var lookups []url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil || r.PostForm.Get("code") != "real-code" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			claims := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"real@example.com"}`))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "at", "token_type": "Bearer", "expires_in": 3600, "id_token": "h." + claims + ".s",
			})
		case "/v1/devices/-/deviceUsers:lookup":
			if r.Header.Get("Authorization") != "Bearer at" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			mu.Lock()
			lookups = append(lookups, r.URL.Query())
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"names": []string{"devices/g1/deviceUsers/u1"}, "customer": "customers/02mock00"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []url.Values {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(lookups)
	}
}

func TestForwardingMode(t *testing.T) {
	g, lookups := fakeGoogle(t)
	fl := newFakeFleet(t)
	e := newEnv(t, mockserver.Config{Relay: true, OAuth: &mockserver.OAuthConfig{
		ClientID: "web-client", ClientSecret: "shh", PartnerID: "partner-1",
		AuthURL: "https://accounts.example.com/auth", TokenURL: g.URL + "/token", APIEndpoint: g.URL + "/",
	}})
	ctx := t.Context()
	_, err := e.relay.Register(ctx, fl.URL, customer)
	require.NoError(t, err)

	signInURL, err := e.relay.SignInURL(customer, "host-9", "tok")
	require.NoError(t, err)
	res, _ := e.get(t, signInURL)
	require.Equal(t, http.StatusFound, res.StatusCode)
	loc, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "accounts.example.com", loc.Host)
	q := loc.Query()
	require.Equal(t, "web-client", q.Get("client_id"))
	require.Equal(t, e.srv.URL+"/api/v1/google-conditional-access/callback", q.Get("redirect_uri"))
	require.Contains(t, q.Get("scope"), "https://www.googleapis.com/auth/cloud-identity.devices.lookup")

	res, body := e.get(t, q.Get("redirect_uri")+"?"+url.Values{"state": {q.Get("state")}, "code": {"real-code"}}.Encode())
	require.Equal(t, http.StatusFound, res.StatusCode, body)
	require.Equal(t, "/google-conditional-access/done", res.Header.Get("Location"))
	call := fl.lastCall(t)
	require.Equal(t, []string{"devices/g1/deviceUsers/u1"}, call.Body.Names)
	require.Equal(t, customer, call.Body.Customer)
	require.Equal(t, "real@example.com", call.Body.Email)
	require.Len(t, lookups(), 1)
	require.Equal(t, "me", lookups()[0].Get("userId"))
	require.Equal(t, "partner-1", lookups()[0].Get("partner"))
	require.Equal(t, "host-9", lookups()[0].Get("iosDeviceId"))
}

// do sends a request with an optional JSON body and Host header, and returns the status
// and the body.
func (e *env) do(t *testing.T, method, path, host string, headers map[string]string, body any) (int, []byte) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, e.srv.URL+path, r)
	require.NoError(t, err)
	if host != "" {
		req.Host = host
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := e.http.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, b
}

func (e *env) httpState(t *testing.T) mockserver.State {
	status, b := e.do(t, "GET", "/_mock/state", "", nil, nil)
	require.Equal(t, http.StatusOK, status, string(b))
	var st mockserver.State
	require.NoError(t, json.Unmarshal(b, &st))
	return st
}

func TestAdminRoutes(t *testing.T) {
	e := newEnv(t, mockserver.Config{Relay: true, Google: true})
	ctx := t.Context()
	admin := func(path string, body any, want int) []byte {
		status, b := e.do(t, "POST", path, "", nil, body)
		require.Equal(t, want, status, string(b))
		return b
	}

	admin("/_mock/google/customers", map[string]string{"customer_id": customer}, http.StatusNoContent)
	admin("/_mock/google/customers", map[string]string{}, http.StatusBadRequest)
	ci := e.cloudIdentity(t, customer)
	require.NoError(t, ci.CheckDelegation(ctx))

	admin("/_mock/google/devices", map[string]any{"device_id": "x"}, http.StatusBadRequest)
	var seeded mockserver.Device
	require.NoError(t, json.Unmarshal(admin("/_mock/google/devices", mockserver.SeedDevice{
		CustomerID: customer, DeviceID: "gsf1", DeviceType: "android", Emails: []string{"ann@example.com"},
	}, http.StatusOK), &seeded))
	require.Len(t, seeded.Users, 1)
	deviceID := strings.TrimPrefix(seeded.Name, "devices/")

	_, err := e.relay.Register(ctx, "https://fleet.example.com", customer)
	require.NoError(t, err)
	st := e.httpState(t)
	require.Len(t, st.Registrations, 1)
	require.NotEmpty(t, st.Registrations[0].Secret)
	require.Equal(t, []mockserver.Device{seeded}, st.Devices)
	require.Equal(t, []string{"GET /v1/devices"}, st.Calls)

	admin("/_mock/google/devices/"+deviceID+"/state", map[string]string{"state": "gone"}, http.StatusBadRequest)
	admin("/_mock/google/devices/nope/state", map[string]string{"state": mockserver.DeviceDeleted}, http.StatusNotFound)
	admin("/_mock/google/devices/"+deviceID+"/state", map[string]string{"state": mockserver.DeviceDeleted}, http.StatusNoContent)
	ok, err := ci.DeviceExists(ctx, deviceID)
	require.NoError(t, err)
	require.False(t, ok)

	admin("/_mock/google/token-error", map[string]string{"error": "unauthorized_client"}, http.StatusNoContent)
	require.Equal(t, "unauthorized_client", e.httpState(t).TokenError)
	require.True(t, cag.IsAuthError(e.cloudIdentity(t, customer).CheckDelegation(ctx)))
	admin("/_mock/google/token-error", map[string]string{"error": ""}, http.StatusNoContent)
	require.NoError(t, e.cloudIdentity(t, customer).CheckDelegation(ctx))

	// Through a tunnel the request comes from loopback under the tunnel's host name.
	for _, route := range [][2]string{
		{"GET", "/_mock/state"}, {"POST", "/_mock/reset"}, {"POST", "/_mock/google/customers"}, {"POST", "/_mock/google/devices"},
		{"POST", "/_mock/google/devices/" + deviceID + "/state"}, {"POST", "/_mock/google/token-error"},
	} {
		status, _ := e.do(t, route[0], route[1], "mock.tunnel.example.com", nil, map[string]string{})
		require.Equal(t, http.StatusForbidden, status, route[1])
	}
	require.Len(t, e.mock.State().Registrations, 1)

	admin("/_mock/reset", nil, http.StatusNoContent)
	st = e.httpState(t)
	require.Empty(t, st.Registrations)
	require.Empty(t, st.Devices)
	require.Empty(t, st.Customers)
	require.Empty(t, st.Calls)
}

func TestPatchClientStateRefused(t *testing.T) {
	e := newEnv(t, mockserver.Config{Google: true, CustomerIDs: []string{customer}})
	bare := strings.TrimPrefix(customer, "C")
	patch := func(name, mask string, body map[string]string) int {
		q := url.Values{"customer": {"customers/" + bare}, "updateMask": {mask}}
		status, _ := e.do(t, "PATCH", "/v1/"+name+"/clientStates/"+bare+"-fleet?"+q.Encode(), "",
			map[string]string{"Authorization": "Bearer x"}, body)
		return status
	}

	// A refused patch creates nothing.
	require.Equal(t, http.StatusBadRequest, patch("devices/new1/deviceUsers/u1", "customId,managed", map[string]string{"customId": "h", "managed": "MAYBE"}))
	require.Equal(t, http.StatusBadRequest, patch("devices/new1/deviceUsers/u1", "managed,color", map[string]string{"managed": "MANAGED"}))
	require.Equal(t, http.StatusBadRequest, patch("devices/new1/deviceUsers/u1", "", map[string]string{"managed": "MANAGED"}))
	require.Empty(t, e.mock.State().Devices)

	// Nor does it change a state already there.
	require.Equal(t, http.StatusOK, patch("devices/new1/deviceUsers/u1", "managed,customId", map[string]string{"customId": "h", "managed": "MANAGED"}))
	require.Equal(t, http.StatusBadRequest, patch("devices/new1/deviceUsers/u1", "customId,managed", map[string]string{"customId": "other", "managed": "MAYBE"}))
	cs := clientState(t, e.mock, "devices/new1/deviceUsers/u1")
	require.Equal(t, "MANAGED", cs.Managed)
	require.Equal(t, "h", cs.CustomID)
}

func TestListDevicesFilter(t *testing.T) {
	e := newEnv(t, mockserver.Config{Google: true})
	for _, id := range []string{"a1", "a2", "a2", "a20"} {
		e.mock.Seed(mockserver.SeedDevice{CustomerID: customer, DeviceID: id, DeviceType: "android", Emails: []string{"ann@example.com"}})
	}
	list := func(filter string) (int, []string) {
		q := url.Values{"customer": {"customers/" + strings.TrimPrefix(customer, "C")}, "filter": {filter}}
		status, b := e.do(t, "GET", "/v1/devices?"+q.Encode(), "", map[string]string{"Authorization": "Bearer x"}, nil)
		var body struct {
			Devices []struct {
				DeviceID string `json:"deviceId"`
			} `json:"devices"`
		}
		if status == http.StatusOK {
			require.NoError(t, json.Unmarshal(b, &body))
		}
		var ids []string
		for _, d := range body.Devices {
			ids = append(ids, d.DeviceID)
		}
		return status, ids
	}

	// A device ID isn't unique: a re-enrolled device can have two records.
	status, ids := list("id:a2")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, []string{"a2", "a2"}, ids)
	status, ids = list("id:nope")
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, ids)
	status, _ = list("serial:a2")
	require.Equal(t, http.StatusBadRequest, status)

	// The real client finds the records the cron lists the DeviceUsers of.
	found, err := e.cloudIdentity(t, customer).FindDeviceIDs(t.Context(), "a2")
	require.NoError(t, err)
	require.Len(t, found, 2)
	for _, id := range found {
		names, err := e.cloudIdentity(t, customer).ListDeviceUserNames(t.Context(), id)
		require.NoError(t, err)
		require.Len(t, names, 1)
	}
}

func TestListDevicesPages(t *testing.T) {
	e := newEnv(t, mockserver.Config{Google: true})
	for _, id := range []string{"a1", "a2", "a3"} {
		e.mock.Seed(mockserver.SeedDevice{CustomerID: customer, DeviceID: id, DeviceType: "android"})
	}
	var got []string
	token := ""
	for page := 0; ; page++ {
		require.Less(t, page, 3)
		q := url.Values{"customer": {"customers/" + strings.TrimPrefix(customer, "C")}, "pageSize": {"2"}, "pageToken": {token}}
		status, b := e.do(t, "GET", "/v1/devices?"+q.Encode(), "", map[string]string{"Authorization": "Bearer x"}, nil)
		require.Equal(t, http.StatusOK, status, string(b))
		var body struct {
			Devices []struct {
				DeviceID string `json:"deviceId"`
			} `json:"devices"`
			NextPageToken string `json:"nextPageToken"`
		}
		require.NoError(t, json.Unmarshal(b, &body))
		require.LessOrEqual(t, len(body.Devices), 2)
		for _, d := range body.Devices {
			got = append(got, d.DeviceID)
		}
		if token = body.NextPageToken; token == "" {
			break
		}
	}
	require.ElementsMatch(t, []string{"a1", "a2", "a3"}, got)

	status, _ := e.do(t, "GET", "/v1/devices?customer=customers/0unknown", "", map[string]string{"Authorization": "Bearer x"}, nil)
	require.Equal(t, http.StatusForbidden, status)
	status, _ = e.do(t, "GET", "/v1/devices?customer=customers/"+strings.TrimPrefix(customer, "C"), "", nil, nil)
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestSimulatedFailures(t *testing.T) {
	e := newEnv(t, mockserver.Config{Relay: true, Google: true, CustomerIDs: []string{customer}, ErrorRate: 1, Latency: time.Millisecond})
	ctx := t.Context()

	// A failed token exchange is an outage, not a broken delegation.
	err := e.cloudIdentity(t, customer).CheckDelegation(ctx)
	require.Error(t, err)
	require.False(t, cag.IsAuthError(err), "%v", err)

	status, b := e.do(t, "GET", "/v1/devices?customer=customers/"+strings.TrimPrefix(customer, "C"), "", map[string]string{"Authorization": "Bearer x"}, nil)
	require.Contains(t, []int{http.StatusTooManyRequests, http.StatusInternalServerError}, status)
	var gErr struct {
		Error struct {
			Status string `json:"status"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(b, &gErr))
	require.NotEmpty(t, gErr.Error.Status)
	require.Empty(t, e.mock.State().Calls)

	// The relay fails with fleetdm.com's plain text, not Google's JSON.
	_, err = e.relay.Register(ctx, "https://fleet.example.com", customer)
	se, ok := errors.AsType[*cag.RelayStatusError](err)
	require.True(t, ok, "%v", err)
	require.Contains(t, []int{http.StatusTooManyRequests, http.StatusInternalServerError}, se.Code)
	require.Contains(t, se.RespErr, "simulated")
	require.NotContains(t, se.RespErr, "{")
	require.Empty(t, e.mock.State().Registrations)

	// The admin routes are spared.
	require.Empty(t, e.httpState(t).Registrations)
}

func TestFakeSignInPage(t *testing.T) {
	for _, c := range []struct {
		delay time.Duration
		want  string
	}{
		{0, ""},
		{7 * time.Second, `<meta http-equiv="refresh" content="7;url=`},
	} {
		e := newEnv(t, mockserver.Config{Relay: true, AutoContinue: c.delay})
		_, err := e.relay.Register(t.Context(), "https://fleet.example.com", customer)
		require.NoError(t, err)
		signInURL, err := e.relay.SignInURL(customer, "host-1", "tok")
		require.NoError(t, err)
		res, body := e.get(t, signInURL)
		require.Equal(t, http.StatusOK, res.StatusCode, body)
		require.Contains(t, body, `id="not-found"`)
		require.Contains(t, body, `id="refused"`)
		if c.want == "" {
			require.NotContains(t, body, "http-equiv")
		} else {
			require.Contains(t, body, c.want)
		}
	}
}

func TestSameNamesWithAndWithoutGoogle(t *testing.T) {
	signInNames := func(google bool) []string {
		fl := newFakeFleet(t)
		e := newEnv(t, mockserver.Config{Relay: true, Google: google, CustomerIDs: []string{customer}})
		_, err := e.relay.Register(t.Context(), fl.URL, customer)
		require.NoError(t, err)
		res, _ := e.get(t, e.signIn(t, "host-1", "tok", mockserver.CodeSignedIn))
		require.Equal(t, http.StatusFound, res.StatusCode)
		return fl.lastCall(t).Body.Names
	}
	relayOnly := signInNames(false)
	require.NotEmpty(t, relayOnly)
	require.Equal(t, relayOnly, signInNames(true))
}
