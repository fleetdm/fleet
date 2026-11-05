package conditional_access_google

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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
	apiStatus   int
	apiBody     string
	// apiDrop closes the connection instead of answering an API call.
	apiDrop bool
	// blockAPI holds API calls until closed.
	blockAPI chan struct{}
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
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
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
}

func TestCheckDelegationNotFoundIsNotAMissingResource(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiStatus = http.StatusNotFound
	fake.apiBody = `{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND"}}`

	err := client.CheckDelegation(t.Context())
	require.EqualError(t, err, "Requested entity was not found.")
	require.False(t, fleet.IsNotFound(err))
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

func TestSetUnmanaged(t *testing.T) {
	client, fake := newTestCloudIdentity(t)

	require.NoError(t, client.SetUnmanaged(t.Context(), "devices/dev1/deviceUsers/user1"))

	require.Len(t, fake.requests, 1)
	req := fake.requests[0]
	require.Equal(t, http.MethodPatch, req.Method)
	require.Equal(t, "/v1/devices/dev1/deviceUsers/user1/clientStates/02cajzvg-fleet", req.URL.Path)
	require.Equal(t, "customers/02cajzvg", req.URL.Query().Get("customer"))
	require.Equal(t, "managed", req.URL.Query().Get("updateMask"))
	require.JSONEq(t, `{"managed":"UNMANAGED"}`, fake.bodies[0])
}

func TestSetUnmanagedNotFound(t *testing.T) {
	client, fake := newTestCloudIdentity(t)
	fake.apiStatus = http.StatusNotFound
	fake.apiBody = `{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND"}}`

	err := client.SetUnmanaged(t.Context(), "devices/gone/deviceUsers/user1")
	require.True(t, fleet.IsNotFound(err))
}

func TestCloudIdentityTokenReusedPerClient(t *testing.T) {
	fake, creds := newTestGoogle(t)

	client, err := NewCloudIdentityFactory(discardLogger)(t.Context(), creds)
	require.NoError(t, err)
	require.NotNil(t, client)
	// Building a client doesn't call Google.
	require.Empty(t, fake.assertions)

	require.NoError(t, client.CheckDelegation(t.Context()))
	require.NoError(t, client.SetUnmanaged(t.Context(), "devices/dev1/deviceUsers/user1"))
	// One token for both calls of the same client.
	require.Len(t, fake.assertions, 1)
}
