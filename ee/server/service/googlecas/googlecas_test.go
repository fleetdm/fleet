package googlecas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestValidateServiceAccountKey(t *testing.T) {
	for _, tc := range []struct {
		name    string
		keyJSON string
		wantErr string
	}{
		{"valid", `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----"}`, ""},
		{"not json", "not json", "must be a JSON object"},
		{"json array", `["service_account"]`, "must be a JSON object"},
		{"wrong type", `{"type":"authorized_user","client_email":"a","private_key":"b"}`, `"type" set to "service_account"`},
		{"missing private key", `{"type":"service_account","client_email":"a"}`, `"client_email" and "private_key"`},
		{"blank client email", `{"type":"service_account","client_email":" ","private_key":"b"}`, `"client_email" and "private_key"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateServiceAccountKey(tc.keyJSON)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// testServiceAccountKey returns a service account key file with a freshly generated private key.
func testServiceAccountKey(t *testing.T) fleet.GoogleCASKeyJSON {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	b, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "my-project",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"client_email": "fleet@my-project.iam.gserviceaccount.com",
	})
	require.NoError(t, err)
	return fleet.GoogleCASKeyJSON(b)
}

func TestVerifyCAPool(t *testing.T) {
	const poolPath = "/v1/projects/my-project/locations/us-central1/caPools/my-pool"

	// newGoogle stands in for both the token endpoint and the CAS API.
	newGoogle := func(t *testing.T, tokenStatus, poolStatus int) (*httptest.Server, *[]string) {
		var calls []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/token" {
				w.WriteHeader(tokenStatus)
				if tokenStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant", "error_description": "from google"})
				return
			}
			w.WriteHeader(poolStatus)
			if poolStatus == http.StatusOK {
				_ = json.NewEncoder(w).Encode(map[string]any{"name": poolPath[len("/v1/"):], "tier": "DEVOPS"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": poolStatus, "message": "from google"}})
		}))
		t.Cleanup(srv.Close)
		return srv, &calls
	}
	newCA := func(t *testing.T) fleet.GoogleCASCA {
		return fleet.GoogleCASCA{Name: "WIFI", Project: "my-project", Location: "us-central1", CAPool: "my-pool", APIKeyJSON: testServiceAccountKey(t)}
	}
	newSvc := func(srv *httptest.Server) *Service {
		return NewService(WithEndpoint(srv.URL), WithTokenURL(srv.URL+"/token"))
	}

	t.Run("pool found", func(t *testing.T) {
		srv, calls := newGoogle(t, http.StatusOK, http.StatusOK)
		require.NoError(t, newSvc(srv).VerifyCAPool(t.Context(), newCA(t)))
		require.Equal(t, []string{"POST /token ", "GET " + poolPath + " Bearer test-token"}, *calls)
	})

	t.Run("token_uri in the key is not used", func(t *testing.T) {
		srv, calls := newGoogle(t, http.StatusOK, http.StatusOK)
		elsewhere, elsewhereCalls := newGoogle(t, http.StatusOK, http.StatusOK)
		for _, field := range []string{"token_uri", "TOKEN_URI"} {
			ca := newCA(t)
			var key map[string]any
			require.NoError(t, json.Unmarshal([]byte(ca.APIKeyJSON), &key))
			key[field] = elsewhere.URL + "/token"
			b, err := json.Marshal(key)
			require.NoError(t, err)
			ca.APIKeyJSON = fleet.GoogleCASKeyJSON(b)
			require.NoError(t, newSvc(srv).VerifyCAPool(t.Context(), ca), field)
		}
		require.Empty(t, *elsewhereCalls)
		require.Len(t, *calls, 4)
	})

	t.Run("requests are subject to fleethttp network blocking", func(t *testing.T) {
		srv, calls := newGoogle(t, http.StatusOK, http.StatusOK)
		fleethttp.SetNetworkBlockingMode(fleethttp.BlockingFull)
		t.Cleanup(func() { fleethttp.SetNetworkBlockingMode(fleethttp.BlockingDisabled) })
		require.Error(t, newSvc(srv).VerifyCAPool(t.Context(), newCA(t)))
		require.Empty(t, *calls)
	})

	for _, tc := range []struct {
		name        string
		tokenStatus int
		poolStatus  int
		wantErr     string
	}{
		{"no permission", http.StatusOK, http.StatusForbidden, "The service account can't access the CA pool"},
		{"bad credentials", http.StatusOK, http.StatusUnauthorized, "The service account can't access the CA pool"},
		{"pool missing", http.StatusOK, http.StatusNotFound, `CA pool "my-pool" wasn't found in project "my-project", location "us-central1"`},
		{"google outage", http.StatusOK, http.StatusServiceUnavailable, "Google CAS returned an error (HTTP 503)"},
		{"key revoked", http.StatusBadRequest, http.StatusOK, "Couldn't authenticate with Google CAS"},
		{"token endpoint outage", http.StatusServiceUnavailable, http.StatusOK, "Google couldn't issue an access token (HTTP 503). Try again later."},
		{"token endpoint rate limit", http.StatusTooManyRequests, http.StatusOK, "Google couldn't issue an access token (HTTP 429). Try again later."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newGoogle(t, tc.tokenStatus, tc.poolStatus)
			err := newSvc(srv).VerifyCAPool(t.Context(), newCA(t))
			require.ErrorContains(t, err, tc.wantErr)
			// The Google message is logged, not shown.
			require.NotContains(t, err.Error(), "from google")
		})
	}

	t.Run("Google unreachable", func(t *testing.T) {
		srv, _ := newGoogle(t, http.StatusOK, http.StatusOK)
		srv.Close()
		err := newSvc(srv).VerifyCAPool(t.Context(), newCA(t))
		require.ErrorContains(t, err, "Couldn't connect to Google CAS")
	})

	t.Run("untrusted TLS certificate", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		t.Cleanup(srv.Close)
		err := newSvc(srv).VerifyCAPool(t.Context(), newCA(t))
		require.ErrorContains(t, err, "Couldn't connect to Google CAS")
	})

	t.Run("connection dropped without a response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600})
				return
			}
			if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
				_ = conn.Close()
			}
		}))
		t.Cleanup(srv.Close)
		err := newSvc(srv).VerifyCAPool(t.Context(), newCA(t))
		require.ErrorContains(t, err, "Couldn't connect to Google CAS")
	})

	t.Run("redirects are not followed", func(t *testing.T) {
		elsewhere, elsewhereCalls := newGoogle(t, http.StatusOK, http.StatusOK)
		for _, redirected := range []string{"/token", poolPath} {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == redirected {
					w.Header().Set("Location", elsewhere.URL+redirected)
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600})
			}))
			t.Cleanup(srv.Close)
			require.Error(t, newSvc(srv).VerifyCAPool(t.Context(), newCA(t)), redirected)
		}
		require.Empty(t, *elsewhereCalls)
	})

	t.Run("key that isn't a service account is refused before any request", func(t *testing.T) {
		srv, calls := newGoogle(t, http.StatusOK, http.StatusOK)
		ca := newCA(t)
		ca.APIKeyJSON = `{"type":"authorized_user","client_id":"a","client_secret":"b","refresh_token":"c"}`
		err := newSvc(srv).VerifyCAPool(t.Context(), ca)
		require.ErrorContains(t, err, `"type" set to "service_account"`)
		require.Empty(t, *calls)
	})

	t.Run("unparseable private key", func(t *testing.T) {
		srv, _ := newGoogle(t, http.StatusOK, http.StatusOK)
		ca := newCA(t)
		ca.APIKeyJSON = `{"type":"service_account","client_email":"a@b","private_key":"not a key"}`
		err := newSvc(srv).VerifyCAPool(t.Context(), ca)
		require.ErrorContains(t, err, "Couldn't authenticate with Google CAS")
	})
}
