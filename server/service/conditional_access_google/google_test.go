package conditional_access_google

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
)

func TestTokenURL(t *testing.T) {
	require.Equal(t, google.JWTTokenURL, tokenURL(""))
	// The override also serves the token endpoint.
	require.Equal(t, "http://mock/token", tokenURL("http://mock/"))
	require.Equal(t, "http://mock/token", tokenURL("http://mock"))
}

func TestTokenExchangeIgnoresKeyTokenURI(t *testing.T) {
	elsewhereCalled := false
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { elsewhereCalled = true }))
	t.Cleanup(elsewhere.Close)

	fake, creds := newTestGoogle(t)
	key := keyFields(t, creds)
	key["token_uri"] = elsewhere.URL + "/token"
	creds.ServiceAccountKey = marshalKey(t, key)

	client, err := NewCloudIdentityFactory(discardLogger)(t.Context(), creds)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.CheckDelegation(t.Context()))
	require.Len(t, fake.assertions, 1)
	require.False(t, elsewhereCalled)
}

type fakeGoogle struct {
	mu         sync.Mutex
	headers    []map[string]any
	assertions []map[string]any
	requests   []*http.Request
	bodies     []string

	tokenStatus int
	tokenBody   string
	// tokenDrop closes the connection instead of answering a token request.
	tokenDrop bool
	apiStatus int
	apiBody   string
	// apiDrop closes the connection instead of answering an API call.
	apiDrop bool
	// blockAPI holds API calls until closed.
	blockAPI chan struct{}
	// pages answers successive API calls in order, before apiBody.
	pages []string
}

func decodeJWTPart(part string) map[string]any {
	b, _ := base64.RawURLEncoding.DecodeString(part)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func (f *fakeGoogle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/token" {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = r.ParseForm()
		if parts := strings.Split(r.Form.Get("assertion"), "."); len(parts) == 3 {
			f.headers = append(f.headers, decodeJWTPart(parts[0]))
			f.assertions = append(f.assertions, decodeJWTPart(parts[1]))
		}
		if f.tokenDrop {
			dropConnection(w)
			return
		}
		if f.tokenStatus != 0 {
			w.WriteHeader(f.tokenStatus)
			_, _ = w.Write([]byte(f.tokenBody))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`))
		return
	}
	if f.blockAPI != nil {
		<-f.blockAPI
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	f.requests = append(f.requests, r)
	f.bodies = append(f.bodies, string(body))
	if f.apiDrop {
		dropConnection(w)
		return
	}
	if len(f.pages) > 0 {
		page := f.pages[0]
		f.pages = f.pages[1:]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(page))
		return
	}
	status := f.apiStatus
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if f.apiBody != "" {
		_, _ = w.Write([]byte(f.apiBody))
		return
	}
	_, _ = w.Write([]byte(`{}`))
}

func dropConnection(w http.ResponseWriter) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err == nil {
		_ = conn.Close()
	}
}

func testKey(t *testing.T) map[string]string {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	return map[string]string{
		"type":           "service_account",
		"client_email":   "sa@project.iam.gserviceaccount.com",
		"private_key_id": "key-id-1",
		"private_key":    string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"token_uri":      google.JWTTokenURL,
	}
}

func keyFields(t *testing.T, creds Credentials) map[string]string {
	key := map[string]string{}
	require.NoError(t, json.Unmarshal(creds.ServiceAccountKey, &key))
	return key
}

func marshalKey(t *testing.T, key map[string]string) []byte {
	b, err := json.Marshal(key)
	require.NoError(t, err)
	return b
}

var discardLogger = slog.New(slog.DiscardHandler)

func newTestGoogle(t *testing.T) (*fakeGoogle, Credentials) {
	fake := &fakeGoogle{}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	dev_mode.SetOverride(EndpointOverrideEnv, srv.URL+"/", t)
	key := marshalKey(t, testKey(t))
	return fake, Credentials{ServiceAccountKey: key, AdminEmail: "admin@example.com", CustomerID: "C02cajzvg"}
}

func newTestCloudIdentity(t *testing.T) (CloudIdentity, *fakeGoogle) {
	fake, creds := newTestGoogle(t)
	client, err := NewCloudIdentityFactory(discardLogger)(t.Context(), creds)
	require.NoError(t, err)
	require.NotNil(t, client)
	return client, fake
}

func TestCheckDelegation(t *testing.T) {
	client, fake := newTestCloudIdentity(t)

	require.NoError(t, client.CheckDelegation(t.Context()))

	// Minted as the admin with the devices scope only, signed with the key ID.
	require.Len(t, fake.assertions, 1)
	require.Equal(t, "admin@example.com", fake.assertions[0]["sub"])
	require.Equal(t, "https://www.googleapis.com/auth/cloud-identity.devices", fake.assertions[0]["scope"])
	require.Equal(t, "key-id-1", fake.headers[0]["kid"])

	require.Len(t, fake.requests, 1)
	req := fake.requests[0]
	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, "/v1/devices", req.URL.Path)
	require.Equal(t, "customers/02cajzvg", req.URL.Query().Get("customer"))
	require.Equal(t, "1", req.URL.Query().Get("pageSize"))
	// No device data is requested.
	require.Equal(t, "nextPageToken", req.URL.Query().Get("fields"))
	require.Equal(t, "Bearer tok", req.Header.Get("Authorization"))
}

func TestCheckDelegationReturnsGoogleReason(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiStatus = http.StatusForbidden
	fake.apiBody = `{"error":{"code":403,"message":"Not Authorized to access this resource/api","status":"PERMISSION_DENIED"}}`

	err := client.CheckDelegation(t.Context())
	require.EqualError(t, err, "Not Authorized to access this resource/api")
	require.False(t, fleet.IsNotFound(err))
	// A 403 on the delegation check isn't about a resource.
	require.False(t, IsStale(err))
	require.True(t, IsAuthError(err))
}

func TestCheckDelegationNotFoundIsNotAMissingResource(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiStatus = http.StatusNotFound
	fake.apiBody = `{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND"}}`

	err := client.CheckDelegation(t.Context())
	require.EqualError(t, err, "Requested entity was not found.")
	require.False(t, fleet.IsNotFound(err))
	require.False(t, IsStale(err))
	// The check names no device, so any refusal is about the delegation.
	require.True(t, IsAuthError(err))
}

func TestGoogleErrorMessages(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fakeGoogle)
		want  string
	}{
		{
			name:  "API error without a JSON body",
			setup: func(f *fakeGoogle) { f.apiStatus, f.apiBody = http.StatusBadGateway, "<html>Bad Gateway</html>" },
			want:  "Google returned HTTP 502",
		},
		{
			name:  "token error without a JSON body",
			setup: func(f *fakeGoogle) { f.tokenStatus, f.tokenBody = http.StatusBadGateway, "<html>Bad Gateway</html>" },
			want:  "Google token request failed: HTTP 502",
		},
		{
			name: "token error with Google's reason",
			setup: func(f *fakeGoogle) {
				f.tokenStatus = http.StatusUnauthorized
				f.tokenBody = `{"error":"unauthorized_client","error_description":"Client is unauthorized to retrieve access tokens using this method."}`
			},
			want: "unauthorized_client: Client is unauthorized to retrieve access tokens using this method.",
		},
		{
			name: "response larger than the cap",
			setup: func(f *fakeGoogle) {
				f.apiBody = `{"nextPageToken":"` + strings.Repeat("x", maxGoogleResponseBytes) + `"}`
			},
			want: "Google's response is too large",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client, fake := newTestCloudIdentity(t)
			c.setup(fake)
			require.EqualError(t, client.CheckDelegation(t.Context()), c.want)
		})
	}

	t.Run("transport error hides the request URL", func(t *testing.T) {
		client, fake := newTestCloudIdentity(t)
		fake.apiDrop = true
		err := client.CheckDelegation(t.Context())
		require.ErrorContains(t, err, "request to Google failed: ")
		serverHost := strings.TrimSuffix(strings.TrimPrefix(dev_mode.Env(EndpointOverrideEnv), "http://"), "/")
		require.NotContains(t, err.Error(), serverHost)
		require.NotContains(t, err.Error(), "customers")
		require.NotContains(t, err.Error(), "/v1/devices")
	})

	t.Run("long reason is truncated", func(t *testing.T) {
		client, fake := newTestCloudIdentity(t)
		fake.apiStatus = http.StatusForbidden
		fake.apiBody = `{"error":{"code":403,"message":"` + strings.Repeat("x", str.MaxErrorResponseBytes+1) + `"}}`
		err := client.CheckDelegation(t.Context())
		require.LessOrEqual(t, len(err.Error()), str.MaxErrorResponseBytes+len(" [truncated]"))
		require.True(t, strings.HasSuffix(err.Error(), " [truncated]"))
	})
}

func TestServiceAccountKeyNotJSON(t *testing.T) {
	_, creds := newTestGoogle(t)
	creds.ServiceAccountKey = []byte("nope")
	_, err := NewCloudIdentityFactory(discardLogger)(t.Context(), creds)
	require.EqualError(t, err, "read the service account key: not a JSON object")
}

// Keys aren't validated on upload, so an unusable one must fail before reaching Google.
func TestUnusableServiceAccountKey(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ecDER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	ecPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecDER}))

	for name, edit := range map[string]func(map[string]string){
		"private key not PEM": func(k map[string]string) { k["private_key"] = "not a key" },
		"no private key":      func(k map[string]string) { delete(k, "private_key") },
		"private key not RSA": func(k map[string]string) { k["private_key"] = ecPEM },
	} {
		t.Run(name, func(t *testing.T) {
			fake, creds := newTestGoogle(t)
			key := keyFields(t, creds)
			edit(key)
			creds.ServiceAccountKey = marshalKey(t, key)
			_, err := NewCloudIdentityFactory(discardLogger)(t.Context(), creds)
			require.ErrorContains(t, err, "read the service account key: ")
			require.False(t, IsUnavailable(err))
			require.Empty(t, fake.assertions)
			require.Empty(t, fake.requests)
		})
	}
}

func TestCloudIdentityCallStopsOnCancel(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	fake.blockAPI = release

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := client.CheckDelegation(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestSetClientState(t *testing.T) {
	cases := []struct {
		name  string
		state fleet.GoogleConditionalAccessClientState
		mask  string
		body  string
	}{
		{"managed", fleet.GoogleConditionalAccessClientState{Managed: new(true)}, "managed", `{"managed":"MANAGED"}`},
		{"unmanaged", fleet.GoogleConditionalAccessClientState{Managed: new(false)}, "managed", `{"managed":"UNMANAGED"}`},
		{"managed with custom ID", fleet.GoogleConditionalAccessClientState{Managed: new(true), CustomID: new("host-uuid-1")},
			"managed,customId", `{"managed":"MANAGED","customId":"host-uuid-1"}`},
		{"custom ID only", fleet.GoogleConditionalAccessClientState{CustomID: new("host-uuid-1")}, "customId", `{"customId":"host-uuid-1"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client, fake := newTestCloudIdentity(t)

			require.NoError(t, client.SetClientState(t.Context(), "devices/dev1/deviceUsers/user1", c.state))

			require.Len(t, fake.requests, 1)
			req := fake.requests[0]
			require.Equal(t, http.MethodPatch, req.Method)
			require.Equal(t, "/v1/devices/dev1/deviceUsers/user1/clientStates/02cajzvg-fleet", req.URL.Path)
			require.Equal(t, "customers/02cajzvg", req.URL.Query().Get("customer"))
			require.Equal(t, c.mask, req.URL.Query().Get("updateMask"))
			require.JSONEq(t, c.body, fake.bodies[0])
		})
	}

	t.Run("nothing to write", func(t *testing.T) {
		client, fake := newTestCloudIdentity(t)
		err := client.SetClientState(t.Context(), "devices/dev1/deviceUsers/user1", fleet.GoogleConditionalAccessClientState{})
		require.Error(t, err)
		require.Empty(t, fake.requests)
	})
}

func TestSetClientStateNotFound(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiStatus = http.StatusNotFound
	fake.apiBody = `{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND"}}`

	err := client.SetClientState(t.Context(), "devices/gone/deviceUsers/user1", fleet.GoogleConditionalAccessClientState{Managed: new(false)})
	require.True(t, fleet.IsNotFound(err))
	require.True(t, IsStale(err))
	require.False(t, IsAuthError(err))
}

func TestSetClientStateForbiddenIsStale(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiStatus = http.StatusForbidden
	fake.apiBody = `{"error":{"code":403,"message":"nope"}}`

	err := client.SetClientState(t.Context(), "devices/left/deviceUsers/user1", fleet.GoogleConditionalAccessClientState{Managed: new(true), CustomID: new("host-uuid-1")})
	require.False(t, fleet.IsNotFound(err))
	require.True(t, IsStale(err))
	require.False(t, IsAuthError(err))
}

func TestCloudIdentityTokenReusedPerClient(t *testing.T) {
	fake, creds := newTestGoogle(t)

	client, err := NewCloudIdentityFactory(discardLogger)(t.Context(), creds)
	require.NoError(t, err)
	require.NotNil(t, client)
	// Building a client doesn't call Google.
	require.Empty(t, fake.assertions)

	require.NoError(t, client.CheckDelegation(t.Context()))
	require.NoError(t, client.SetClientState(t.Context(), "devices/dev1/deviceUsers/user1", fleet.GoogleConditionalAccessClientState{Managed: new(false)}))
	// One token for both calls of the same client.
	require.Len(t, fake.assertions, 1)
}

// A failure on Google's side or on the way there is told apart from settings Google
// rejects, so the caller doesn't blame the admin's input for an outage.
func TestGoogleErrorUnavailable(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(f *fakeGoogle)
		unavailable bool
	}{
		{"API 5xx", func(f *fakeGoogle) { f.apiStatus, f.apiBody = http.StatusServiceUnavailable, "<html>down</html>" }, true},
		{"API 429", func(f *fakeGoogle) {
			f.apiStatus, f.apiBody = http.StatusTooManyRequests, `{"error":{"code":429,"message":"Quota exceeded","status":"RESOURCE_EXHAUSTED"}}`
		}, true},
		{"token endpoint 5xx", func(f *fakeGoogle) { f.tokenStatus, f.tokenBody = http.StatusBadGateway, "<html>Bad Gateway</html>" }, true},
		{"token endpoint 5xx with an OAuth error", func(f *fakeGoogle) {
			f.tokenStatus, f.tokenBody = http.StatusInternalServerError, `{"error":"internal_failure","error_description":"Backend Error"}`
		}, true},
		{"token endpoint 429", func(f *fakeGoogle) {
			f.tokenStatus, f.tokenBody = http.StatusTooManyRequests, `{"error":"rate_limit_exceeded"}`
		}, true},
		{"token endpoint answers a proxy page", func(f *fakeGoogle) { f.tokenStatus, f.tokenBody = http.StatusOK, "<html>Sign in</html>" }, true},
		{"token request dropped", func(f *fakeGoogle) { f.tokenDrop = true }, true},
		{"connection dropped", func(f *fakeGoogle) { f.apiDrop = true }, true},
		{"response larger than the cap", func(f *fakeGoogle) {
			f.apiBody = `{"nextPageToken":"` + strings.Repeat("x", maxGoogleResponseBytes) + `"}`
		}, true},
		{"API 403", func(f *fakeGoogle) {
			f.apiStatus, f.apiBody = http.StatusForbidden, `{"error":{"code":403,"message":"Not Authorized to access this resource/api"}}`
		}, false},
		{"API 404", func(f *fakeGoogle) {
			f.apiStatus, f.apiBody = http.StatusNotFound, `{"error":{"code":404,"message":"Requested entity was not found."}}`
		}, false},
		{"token refused", func(f *fakeGoogle) {
			f.tokenStatus, f.tokenBody = http.StatusUnauthorized, `{"error":"unauthorized_client","error_description":"Client is unauthorized."}`
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client, fake := newTestCloudIdentity(t)
			c.setup(fake)
			err := client.CheckDelegation(t.Context())
			require.Error(t, err)
			require.Equal(t, c.unavailable, IsUnavailable(err))
		})
	}

	t.Run("request canceled", func(t *testing.T) {
		client, _ := newTestCloudIdentity(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := client.CheckDelegation(ctx)
		require.ErrorIs(t, err, context.Canceled)
		require.True(t, IsUnavailable(err))
	})
}

func TestAuthError(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.tokenStatus = http.StatusBadRequest
	fake.tokenBody = `{"error":"unauthorized_client","error_description":"Client is unauthorized"}`

	err := client.SetClientState(t.Context(), "devices/d/deviceUsers/u", fleet.GoogleConditionalAccessClientState{Managed: new(true), CustomID: new("host-uuid-1")})
	require.True(t, IsAuthError(err))
	require.False(t, IsStale(err))

	client, fake = newTestCloudIdentity(t)
	fake.apiStatus = http.StatusUnauthorized
	fake.apiBody = `{"error":{"code":401,"message":"bad token"}}`
	require.True(t, IsAuthError(client.CheckDelegation(t.Context())))
	// A refused token on a device call isn't about the device.
	err = client.SetClientState(t.Context(), "devices/d/deviceUsers/u", fleet.GoogleConditionalAccessClientState{Managed: new(true)})
	require.True(t, IsAuthError(err))
	require.False(t, IsStale(err))

	// A token endpoint outage or rate limit isn't a delegation problem.
	client, fake = newTestCloudIdentity(t)
	fake.tokenStatus = http.StatusBadGateway
	fake.tokenBody = "<html>Bad Gateway</html>"
	require.False(t, IsAuthError(client.CheckDelegation(t.Context())))
	client, fake = newTestCloudIdentity(t)
	fake.tokenStatus = http.StatusTooManyRequests
	fake.tokenBody = `{"error":"rate_limit_exceeded"}`
	require.False(t, IsAuthError(client.CheckDelegation(t.Context())))

	// Network failures aren't auth errors.
	client, fake = newTestCloudIdentity(t)
	fake.apiDrop = true
	require.False(t, IsAuthError(client.CheckDelegation(t.Context())))
	require.False(t, IsAuthError(errors.New("other")))
}

func TestListDeviceUserNames(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.pages = []string{
		`{"deviceUsers":[{"name":"devices/dev1/deviceUsers/a"}],"nextPageToken":"p2"}`,
		`{"deviceUsers":[{"name":"devices/dev1/deviceUsers/b"}]}`,
	}

	names, err := client.ListDeviceUserNames(t.Context(), "dev1")
	require.NoError(t, err)
	require.Equal(t, []string{"devices/dev1/deviceUsers/a", "devices/dev1/deviceUsers/b"}, names)
	require.Len(t, fake.requests, 2)
	req := fake.requests[0]
	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, "/v1/devices/dev1/deviceUsers", req.URL.Path)
	require.Equal(t, "customers/02cajzvg", req.URL.Query().Get("customer"))
	require.Equal(t, "20", req.URL.Query().Get("pageSize"))
	require.Equal(t, "p2", fake.requests[1].URL.Query().Get("pageToken"))

	fake.apiStatus = http.StatusNotFound
	fake.apiBody = `{"error":{"code":404,"message":"Requested entity was not found."}}`
	_, err = client.ListDeviceUserNames(t.Context(), "gone")
	require.True(t, fleet.IsNotFound(err))
	require.True(t, IsStale(err))

	// A device that left the Workspace.
	fake.apiStatus = http.StatusForbidden
	fake.apiBody = `{"error":{"code":403,"message":"nope"}}`
	_, err = client.ListDeviceUserNames(t.Context(), "left")
	require.False(t, fleet.IsNotFound(err))
	require.True(t, IsStale(err))
	require.False(t, IsAuthError(err))

	// A page that fails leaves no partial listing.
	fake.pages = []string{`{"deviceUsers":[{"name":"devices/dev1/deviceUsers/a"}],"nextPageToken":"p2"}`}
	fake.apiStatus = http.StatusBadGateway
	fake.apiBody = "<html>Bad Gateway</html>"
	names, err = client.ListDeviceUserNames(t.Context(), "dev1")
	require.True(t, IsUnavailable(err))
	require.Empty(t, names)
}

func TestListDeviceUserNamesKeepsOnlyTheDeviceNames(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiBody = `{"deviceUsers":[{"name":"devices/DEV1/deviceUsers/a"},{"name":"devices/dev2/deviceUsers/b"},{"name":"devices/dev1x/deviceUsers/c"}]}`

	names, err := client.ListDeviceUserNames(t.Context(), "dev1")
	require.NoError(t, err)
	require.Equal(t, []string{"devices/DEV1/deviceUsers/a"}, names)
}

func TestDeviceIDThatCannotNameADevice(t *testing.T) {
	client, fake := newTestCloudIdentity(t)

	// "-" stands for every device, and the others would change the request path.
	for _, id := range []string{"", "-", "dev1/deviceUsers/u", "dev1?x", "dev1#x"} {
		_, err := client.ListDeviceUserNames(t.Context(), id)
		require.True(t, fleet.IsNotFound(err), id)
		require.True(t, IsStale(err), id)
		require.False(t, IsAuthError(err), id)

		exists, err := client.DeviceExists(t.Context(), id)
		require.NoError(t, err, id)
		require.False(t, exists, id)
	}
	require.Empty(t, fake.requests)
}

func TestListingStopsAfterTooManyPages(t *testing.T) {
	endless := func(page string, n int) []string {
		pages := make([]string, n)
		for i := range pages {
			pages[i] = page
		}
		return pages
	}

	client, fake := newTestCloudIdentity(t)
	fake.pages = endless(`{"deviceUsers":[{"name":"devices/dev1/deviceUsers/a"}],"nextPageToken":"more"}`, maxDeviceUserPages+1)
	names, err := client.ListDeviceUserNames(t.Context(), "dev1")
	require.ErrorIs(t, err, errTooManyPages)
	require.Empty(t, names)
	require.Len(t, fake.requests, maxDeviceUserPages)
	// Google answered, so this is no outage, nor about the device or the delegation.
	require.False(t, IsUnavailable(err))
	require.False(t, IsStale(err))
	require.False(t, IsAuthError(err))

	client, fake = newTestCloudIdentity(t)
	fake.pages = endless(`{"devices":[{"name":"devices/dev1","deviceId":"other"}],"nextPageToken":"more"}`, maxDevicePages+1)
	ids, err := client.FindDeviceIDs(t.Context(), "358304e85c824471")
	require.ErrorIs(t, err, errTooManyPages)
	require.Empty(t, ids)
	require.Len(t, fake.requests, maxDevicePages)
	require.False(t, IsUnavailable(err))
	require.False(t, IsAuthError(err))
}

func TestDeviceExists(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiBody = `{"name":"devices/dev1"}`

	exists, err := client.DeviceExists(t.Context(), "dev1")
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, fake.requests, 1)
	require.Equal(t, http.MethodGet, fake.requests[0].Method)
	require.Equal(t, "/v1/devices/dev1", fake.requests[0].URL.Path)
	require.Equal(t, "customers/02cajzvg", fake.requests[0].URL.Query().Get("customer"))

	fake.apiStatus = http.StatusNotFound
	fake.apiBody = `{"error":{"code":404,"message":"Requested entity was not found."}}`
	exists, err = client.DeviceExists(t.Context(), "dev1")
	require.NoError(t, err)
	require.False(t, exists)

	// A device that left the Workspace is for the caller to decide on.
	fake.apiStatus = http.StatusForbidden
	fake.apiBody = `{"error":{"code":403,"message":"nope"}}`
	_, err = client.DeviceExists(t.Context(), "dev1")
	require.True(t, IsStale(err))
	require.False(t, IsAuthError(err))

	fake.apiStatus = http.StatusBadRequest
	fake.apiBody = `{"error":{"code":400,"message":"bad"}}`
	_, err = client.DeviceExists(t.Context(), "dev1")
	require.Error(t, err)
	require.False(t, IsStale(err))
}

func TestFindDeviceIDs(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.pages = []string{
		`{"devices":[{"name":"devices/dev1","deviceId":"358304e85c824471"},{"name":"devices/other","deviceId":"358304e85c8244710"}],"nextPageToken":"p2"}`,
		`{"devices":[{"name":"devices/dev2","deviceId":"358304E85C824471"}]}`,
	}

	// Only exact matches are kept, whatever Google's filter lets through.
	ids, err := client.FindDeviceIDs(t.Context(), "358304e85c824471")
	require.NoError(t, err)
	require.Equal(t, []string{"dev1", "dev2"}, ids)
	require.Len(t, fake.requests, 2)
	req := fake.requests[0]
	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, "/v1/devices", req.URL.Path)
	require.Equal(t, "customers/02cajzvg", req.URL.Query().Get("customer"))
	require.Equal(t, "id:358304e85c824471", req.URL.Query().Get("filter"))
	require.Equal(t, "100", req.URL.Query().Get("pageSize"))
	require.Equal(t, "p2", fake.requests[1].URL.Query().Get("pageToken"))

	ids, err = client.FindDeviceIDs(t.Context(), "none")
	require.NoError(t, err)
	require.Empty(t, ids)

	// An empty ID isn't searched: it would match the records that have no deviceId.
	fake.apiBody = `{"devices":[{"name":"devices/other"}]}`
	ids, err = client.FindDeviceIDs(t.Context(), "")
	require.NoError(t, err)
	require.Empty(t, ids)
	require.Len(t, fake.requests, 3)

	fake.apiStatus = http.StatusForbidden
	fake.apiBody = `{"error":{"code":403,"message":"The caller does not have permission"}}`
	_, err = client.FindDeviceIDs(t.Context(), "358304e85c824471")
	require.Error(t, err)
	require.False(t, IsStale(err))
}
