package mock

import (
	"encoding/json"
	"errors"
	"html"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	testCustomerID = "C01234abc"
	testHostUUID   = "35B8F9A0-4C2E-4B1D-9F3A-7E6D5C4B3A21"
	testDeviceID   = "mock-35B8F9A0-4C2"
	testDeviceUser = "devices/" + testDeviceID + "/deviceUsers/" + testDeviceID
)

func newTestMock(t *testing.T, cfg Config) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(New(cfg))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, rawURL string, body io.Reader, header http.Header) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, rawURL, body)
	require.NoError(t, err)
	maps.Copy(req.Header, header)
	// Redirects are returned rather than followed so tests can assert where the mock sends the browser.
	resp, err := fleethttp.NewClient(fleethttp.WithFollowRedir(false)).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(b)
}

func createRegistration(t *testing.T, mockURL, customerID, origin string, wantStatus int) Registration {
	t.Helper()
	header := http.Header{"Content-Type": {"application/json"}}
	if origin != "" {
		header.Set("Origin", origin)
	}
	resp, body := do(t, http.MethodPost, mockURL+"/api/v1/google-conditional-access",
		strings.NewReader(`{"googleCustomerId":"`+customerID+`"}`), header)
	require.Equal(t, wantStatus, resp.StatusCode, body)
	var reg Registration
	if wantStatus == http.StatusOK {
		require.NoError(t, json.Unmarshal([]byte(body), &reg))
	}
	return reg
}

func deleteRegistration(t *testing.T, mockURL, customerID, secret string) int {
	t.Helper()
	q := url.Values{"googleCustomerId": {customerID}, "fleetServerSecret": {secret}}
	resp, _ := do(t, http.MethodDelete, mockURL+"/api/v1/google-conditional-access?"+q.Encode(), nil, nil)
	return resp.StatusCode
}

func getState(t *testing.T, mockURL string) State {
	t.Helper()
	resp, body := do(t, http.MethodGet, mockURL+"/mock/state", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var st State
	require.NoError(t, json.Unmarshal([]byte(body), &st))
	return st
}

func signInURL(mockURL, customerID string) string {
	q := url.Values{"googleCustomerId": {customerID}, "token": {"sign-in-token"}, "hostUuid": {testHostUUID}, "platform": {"ios"}}
	return mockURL + "/api/v1/google-conditional-access/sign-in?" + q.Encode()
}

var stateInput = regexp.MustCompile(`name="state" value="([^"]+)"`)

// fakeSignIn loads the fake sign-in page and returns the state it would auto-submit.
func fakeSignIn(t *testing.T, mockURL, customerID string) string {
	t.Helper()
	resp, body := do(t, http.MethodGet, signInURL(mockURL, customerID), nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	require.Contains(t, body, "Google sign-in (mock)")
	m := stateInput.FindStringSubmatch(body)
	require.Len(t, m, 2, body)
	return m[1]
}

func callback(t *testing.T, mockURL, state string) *http.Response {
	t.Helper()
	q := url.Values{"state": {state}, "code": {"mock-code"}, "email": {"user@example.com"}}
	resp, _ := do(t, http.MethodGet, mockURL+"/api/v1/google-conditional-access/callback?"+q.Encode(), nil, nil)
	return resp
}

// fakeFleet records the inbound register calls the mock makes and answers with status.
type fakeFleet struct {
	mu     sync.Mutex
	status int
	calls  []fleetCall
}

type fleetCall struct {
	path          string
	authorization string
	body          map[string]string
}

func newFakeFleet(t *testing.T) (*fakeFleet, *httptest.Server) {
	t.Helper()
	ff := &fakeFleet{status: http.StatusOK}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		ff.mu.Lock()
		defer ff.mu.Unlock()
		ff.calls = append(ff.calls, fleetCall{path: r.URL.Path, authorization: r.Header.Get("Authorization"), body: body})
		w.WriteHeader(ff.status)
	}))
	t.Cleanup(srv.Close)
	return ff, srv
}

func (ff *fakeFleet) setStatus(s int) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.status = s
}

func (ff *fakeFleet) lastCall(t *testing.T) fleetCall {
	t.Helper()
	ff.mu.Lock()
	defer ff.mu.Unlock()
	require.NotEmpty(t, ff.calls)
	return ff.calls[len(ff.calls)-1]
}

func TestRegistration(t *testing.T) {
	srv := newTestMock(t, Config{})

	createRegistration(t, srv.URL, testCustomerID, "", http.StatusBadRequest)

	reg := createRegistration(t, srv.URL, testCustomerID, "https://fleet.example.com", http.StatusOK)
	require.Len(t, reg.FleetServerSecret, 30)
	require.Equal(t, "mock-client-id", reg.OAuthClientID)

	createRegistration(t, srv.URL, testCustomerID, "https://fleet.example.com", http.StatusConflict)

	require.Equal(t, http.StatusNotFound, deleteRegistration(t, srv.URL, testCustomerID, "wrong"))
	require.Equal(t, http.StatusNotFound, deleteRegistration(t, srv.URL, "C0other", reg.FleetServerSecret))
	require.Equal(t, http.StatusOK, deleteRegistration(t, srv.URL, testCustomerID, reg.FleetServerSecret))
	require.Empty(t, getState(t, srv.URL).Registrations)

	reg2 := createRegistration(t, srv.URL, testCustomerID, "https://fleet.example.com", http.StatusOK)
	require.NotEqual(t, reg.FleetServerSecret, reg2.FleetServerSecret)

	srv2 := newTestMock(t, Config{GoogleClientID: "real-client.apps.googleusercontent.com", GoogleClientSecret: "s"})
	reg3 := createRegistration(t, srv2.URL, testCustomerID, "https://fleet.example.com", http.StatusOK)
	require.Equal(t, "real-client.apps.googleusercontent.com", reg3.OAuthClientID)
}

func TestSignIn(t *testing.T) {
	t.Run("unknown customer", func(t *testing.T) {
		srv := newTestMock(t, Config{})
		resp, _ := do(t, http.MethodGet, signInURL(srv.URL, testCustomerID), nil, nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("invalid platform", func(t *testing.T) {
		srv := newTestMock(t, Config{})
		createRegistration(t, srv.URL, testCustomerID, "https://fleet.example.com", http.StatusOK)
		u := strings.Replace(signInURL(srv.URL, testCustomerID), "platform=ios", "platform=darwin", 1)
		resp, _ := do(t, http.MethodGet, u, nil, nil)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("forwarding mode redirects to Google", func(t *testing.T) {
		srv := newTestMock(t, Config{GoogleClientID: "real-client", GoogleClientSecret: "real-secret", PublicURL: "https://mock.example.com"})
		createRegistration(t, srv.URL, testCustomerID, "https://fleet.example.com", http.StatusOK)
		resp, _ := do(t, http.MethodGet, signInURL(srv.URL, testCustomerID), nil, nil)
		require.Equal(t, http.StatusFound, resp.StatusCode)
		loc, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)
		require.Equal(t, "accounts.google.com", loc.Host)
		q := loc.Query()
		require.Equal(t, "real-client", q.Get("client_id"))
		require.Equal(t, "https://www.googleapis.com/auth/cloud-identity.devices.lookup", q.Get("scope"))
		require.Equal(t, "https://mock.example.com/api/v1/google-conditional-access/callback", q.Get("redirect_uri"))
		require.NotEmpty(t, q.Get("state"))
		require.NotContains(t, q.Get("state"), "sign-in-token")
	})
}

func TestCallbackFakeMode(t *testing.T) {
	cases := []struct {
		fleetStatus int
		wantPage    string
		wantCopy    string
	}{
		{http.StatusOK, "/google-conditional-access/done", "You're done!"},
		{http.StatusForbidden, "/google-conditional-access/not-allowed", "This device isn't allowed to access company data."},
		{http.StatusConflict, "/google-conditional-access/not-allowed", "This device isn't allowed to access company data."},
		{http.StatusInternalServerError, "/google-conditional-access/error", "Something went wrong. Try again."},
		{http.StatusUnauthorized, "/google-conditional-access/error", "Something went wrong. Try again."},
	}

	ff, fleetSrv := newFakeFleet(t)
	srv := newTestMock(t, Config{})
	reg := createRegistration(t, srv.URL, testCustomerID, fleetSrv.URL, http.StatusOK)

	for _, c := range cases {
		t.Run(http.StatusText(c.fleetStatus), func(t *testing.T) {
			ff.setStatus(c.fleetStatus)
			state := fakeSignIn(t, srv.URL, testCustomerID)

			resp := callback(t, srv.URL, state)
			require.Equal(t, http.StatusSeeOther, resp.StatusCode)
			require.Equal(t, c.wantPage, resp.Header.Get("Location"))

			call := ff.lastCall(t)
			require.Equal(t, "/api/v1/fleet/google_conditional_access/register", call.path)
			require.Equal(t, "Bearer "+reg.FleetServerSecret, call.authorization)
			require.Equal(t, map[string]string{
				"token":                   "sign-in-token",
				"google_device_id":        testDeviceID,
				"google_device_user_name": testDeviceUser,
				"google_email":            "user@example.com",
			}, call.body)

			pageResp, page := do(t, http.MethodGet, srv.URL+c.wantPage, nil, nil)
			require.Equal(t, http.StatusOK, pageResp.StatusCode)
			require.Contains(t, html.UnescapeString(page), c.wantCopy)
		})
	}

	t.Run("state is single use", func(t *testing.T) {
		ff.setStatus(http.StatusOK)
		state := fakeSignIn(t, srv.URL, testCustomerID)
		require.Equal(t, "/google-conditional-access/done", callback(t, srv.URL, state).Header.Get("Location"))
		require.Equal(t, "/google-conditional-access/error", callback(t, srv.URL, state).Header.Get("Location"))
	})

	t.Run("fleet server url flag overrides origin", func(t *testing.T) {
		ff2, fleetSrv2 := newFakeFleet(t)
		srv2 := newTestMock(t, Config{FleetServerURL: fleetSrv2.URL + "/"})
		createRegistration(t, srv2.URL, testCustomerID, "https://unreachable.example.com", http.StatusOK)
		state := fakeSignIn(t, srv2.URL, testCustomerID)
		require.Equal(t, "/google-conditional-access/done", callback(t, srv2.URL, state).Header.Get("Location"))
		require.Equal(t, "/api/v1/fleet/google_conditional_access/register", ff2.lastCall(t).path)
	})

	st := getState(t, srv.URL)
	require.Len(t, st.Devices, 1)
	require.Equal(t, Device{
		Name:           "devices/" + testDeviceID,
		DeviceID:       testDeviceID,
		DeviceUserName: testDeviceUser,
		CustomerID:     testCustomerID,
		Email:          "user@example.com",
		HostUUID:       testHostUUID,
	}, st.Devices[0])
}

func TestAndroidDeviceRegister(t *testing.T) {
	srv := newTestMock(t, Config{})
	reg := createRegistration(t, srv.URL, testCustomerID, "https://fleet.example.com", http.StatusOK)

	register := func(customerID, secret, androidID string) (int, map[string]string) {
		body, err := json.Marshal(map[string]string{
			"googleCustomerId":  customerID,
			"fleetServerSecret": secret,
			"serverAuthCode":    "4/mock-code",
			"androidId":         androidID,
			"hostUuid":          testHostUUID,
		})
		require.NoError(t, err)
		resp, respBody := do(t, http.MethodPost, srv.URL+"/api/v1/google-conditional-access/device/register",
			strings.NewReader(string(body)), http.Header{"Content-Type": {"application/json"}})
		var out map[string]string
		_ = json.Unmarshal([]byte(respBody), &out)
		return resp.StatusCode, out
	}

	status, out := register(testCustomerID, reg.FleetServerSecret, "a1b2c3d4e5f60718")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, map[string]string{
		"google_device_id":        testDeviceID,
		"google_device_user_name": testDeviceUser,
		"google_email":            "user@example.com",
	}, out)

	status, _ = register("C0unknown", reg.FleetServerSecret, "a1b2c3d4e5f60718")
	require.Equal(t, http.StatusNotFound, status)
	status, _ = register(testCustomerID, "wrong", "a1b2c3d4e5f60718")
	require.Equal(t, http.StatusNotFound, status)
	status, _ = register(testCustomerID, reg.FleetServerSecret, "")
	require.Equal(t, http.StatusBadRequest, status)

	resp, _ := do(t, http.MethodPost, srv.URL+"/mock/android/0000000000000000/no-record", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	status, out = register(testCustomerID, reg.FleetServerSecret, "0000000000000000")
	require.Equal(t, http.StatusUnprocessableEntity, status)
	require.NotEmpty(t, out["error"])

	st := getState(t, srv.URL)
	require.Len(t, st.Devices, 1)
	require.Equal(t, "a1b2c3d4e5f60718", st.Devices[0].AndroidID)
}

// newCloudIdentityService builds the real Cloud Identity client the way the Fleet server
// does, delegated as adminEmail, with the key the mock hands out.
func newCloudIdentityService(t *testing.T, mockURL, adminEmail string) *cloudidentity.Service {
	t.Helper()
	resp, keyJSON := do(t, http.MethodGet, mockURL+"/mock/service-account-key", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, keyJSON)

	cfg, err := google.JWTConfigFromJSON([]byte(keyJSON), cloudidentity.CloudIdentityDevicesScope)
	require.NoError(t, err)
	require.Equal(t, mockURL+"/token", cfg.TokenURL)
	cfg.Subject = adminEmail

	svc, err := cloudidentity.NewService(t.Context(),
		option.WithTokenSource(cfg.TokenSource(t.Context())),
		option.WithEndpoint(mockURL+"/"),
	)
	require.NoError(t, err)
	return svc
}

func requireGoogleStatus(t *testing.T, err error, code int) {
	t.Helper()
	var gErr *googleapi.Error
	require.True(t, errors.As(err, &gErr) && gErr != nil, "want a *googleapi.Error, got %v", err)
	require.Equal(t, code, gErr.Code)
}

func TestCloudIdentity(t *testing.T) {
	_, fleetSrv := newFakeFleet(t)
	srv := newTestMock(t, Config{Customers: []string{"C0known"}})
	createRegistration(t, srv.URL, testCustomerID, fleetSrv.URL, http.StatusOK)
	svc := newCloudIdentityService(t, srv.URL, "fleet-admin@example.com")

	t.Run("devices list", func(t *testing.T) {
		for _, customerID := range []string{"C0known", testCustomerID} {
			res, err := svc.Devices.List().Customer("customers/" + customerID).PageSize(1).Context(t.Context()).Do()
			require.NoError(t, err, customerID)
			require.Len(t, res.Devices, 1)
		}
		_, err := svc.Devices.List().Customer("customers/C0unknown").PageSize(1).Context(t.Context()).Do()
		requireGoogleStatus(t, err, http.StatusForbidden)
	})

	t.Run("requires a token from the mock", func(t *testing.T) {
		resp, _ := do(t, http.MethodGet, srv.URL+"/v1/devices?customer=customers/C0known", nil, nil)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// The callback is what puts a device in Google's inventory.
	state := fakeSignIn(t, srv.URL, testCustomerID)
	require.Equal(t, "/google-conditional-access/done", callback(t, srv.URL, state).Header.Get("Location"))
	customer := "customers/" + testCustomerID
	clientStateName := testDeviceUser + "/clientStates/" + testCustomerID + "-fleet"

	t.Run("client state patch", func(t *testing.T) {
		for _, managed := range []string{"MANAGED", "UNMANAGED"} {
			op, err := svc.Devices.DeviceUsers.ClientStates.Patch(clientStateName,
				&cloudidentity.GoogleAppsCloudidentityDevicesV1ClientState{Managed: managed, CustomId: testHostUUID}).
				Customer(customer).UpdateMask("managed,customId").Context(t.Context()).Do()
			require.NoError(t, err)
			require.True(t, op.Done)

			st := getState(t, srv.URL)
			require.Len(t, st.ClientStates, 1)
			require.Equal(t, clientStateName, st.ClientStates[0].Name)
			require.Equal(t, managed, st.ClientStates[0].Managed)
			require.Equal(t, testHostUUID, st.ClientStates[0].CustomID)
			require.Equal(t, "fleet-admin@example.com", st.ClientStates[0].UpdatedBy)
		}

		// Only fields in the mask are written.
		_, err := svc.Devices.DeviceUsers.ClientStates.Patch(clientStateName,
			&cloudidentity.GoogleAppsCloudidentityDevicesV1ClientState{Managed: "MANAGED", CustomId: "other"}).
			Customer(customer).UpdateMask("managed").Context(t.Context()).Do()
		require.NoError(t, err)
		cs := getState(t, srv.URL).ClientStates[0]
		require.Equal(t, "MANAGED", cs.Managed)
		require.Equal(t, testHostUUID, cs.CustomID)

		_, err = svc.Devices.DeviceUsers.ClientStates.Patch("devices/nope/deviceUsers/nope/clientStates/"+testCustomerID+"-fleet",
			&cloudidentity.GoogleAppsCloudidentityDevicesV1ClientState{Managed: "MANAGED"}).
			Customer(customer).UpdateMask("managed").Context(t.Context()).Do()
		requireGoogleStatus(t, err, http.StatusNotFound)
	})

	t.Run("device get and stale flip", func(t *testing.T) {
		dev, err := svc.Devices.Get("devices/" + testDeviceID).Customer(customer).Context(t.Context()).Do()
		require.NoError(t, err)
		require.Equal(t, testDeviceID, dev.DeviceId)
		require.Equal(t, "devices/"+testDeviceID, dev.Name)

		resp, _ := do(t, http.MethodPost, srv.URL+"/mock/devices/"+testDeviceID+"/stale", nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		_, err = svc.Devices.Get("devices/" + testDeviceID).Customer(customer).Context(t.Context()).Do()
		requireGoogleStatus(t, err, http.StatusNotFound)
		_, err = svc.Devices.DeviceUsers.ClientStates.Patch(clientStateName,
			&cloudidentity.GoogleAppsCloudidentityDevicesV1ClientState{Managed: "UNMANAGED"}).
			Customer(customer).UpdateMask("managed").Context(t.Context()).Do()
		requireGoogleStatus(t, err, http.StatusNotFound)

		resp, _ = do(t, http.MethodPost, srv.URL+"/mock/devices/unknown/stale", nil, nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
