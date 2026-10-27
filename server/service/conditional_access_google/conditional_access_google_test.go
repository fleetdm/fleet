package conditional_access_google

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func serviceAccountJSON(t *testing.T, tokenURI string) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	b, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"client_email": "sa@example.iam.gserviceaccount.com",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"token_uri":    tokenURI,
	})
	require.NoError(t, err)
	return b
}

func TestCheckDelegation(t *testing.T) {
	var tokenCalls, listCalls atomic.Int32
	var listStatus atomic.Int32
	listStatus.Store(http.StatusOK)
	var (
		mu           sync.Mutex
		assertion    string
		listRequests []*http.Request
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		_ = r.ParseForm()
		mu.Lock()
		assertion = r.PostForm.Get("assertion")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		listCalls.Add(1)
		mu.Lock()
		listRequests = append(listRequests, r)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if code := int(listStatus.Load()); code != http.StatusOK {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"error":{"code":403,"message":"Not Authorized to access this resource/api","status":"PERMISSION_DENIED"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"devices":[]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := NewClient(t.Context(), serviceAccountJSON(t, srv.URL+"/token"), "admin@example.com", "C01234abc", srv.URL+"/")
	require.NoError(t, err)

	require.NoError(t, c.CheckDelegation(t.Context()))
	require.NoError(t, c.CheckDelegation(t.Context()))
	require.EqualValues(t, 1, tokenCalls.Load(), "token should be cached")
	require.EqualValues(t, 2, listCalls.Load())

	// The JWT claims carry the impersonated admin and the single delegated scope.
	parts := strings.Split(assertion, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	require.Equal(t, "admin@example.com", claims["sub"])
	require.Equal(t, Scope, claims["scope"])
	for _, r := range listRequests {
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		require.Equal(t, "customers/C01234abc", r.URL.Query().Get("customer"))
		require.Equal(t, "1", r.URL.Query().Get("pageSize"))
	}

	listStatus.Store(http.StatusForbidden)
	err = c.CheckDelegation(t.Context())
	require.ErrorContains(t, err, "Not Authorized to access this resource/api")
}

func TestNewClientInvalidKey(t *testing.T) {
	_, err := NewClient(t.Context(), []byte(`{"client_email":"x"}`), "admin@example.com", "C01234abc", "")
	require.Error(t, err)

	_, err = NewClient(t.Context(), serviceAccountJSON(t, "https://oauth2.googleapis.com/token"), "admin@example.com", "", "")
	require.Error(t, err)
}
