package googlecas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
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

func TestIssueCertificate(t *testing.T) {
	const certificatesPath = "/v1/projects/my-project/locations/us-central1/caPools/my-pool/certificates"

	csrKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "fleetie"}}, csrKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)
	issuedDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "fleetie"}, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour),
	}, &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test CA"}}, &csrKey.PublicKey, csrKey)
	require.NoError(t, err)
	issuedPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issuedDER}))

	type google struct {
		srv         *httptest.Server
		mu          sync.Mutex
		tokenCalls  int
		tokenHang   chan struct{}
		issueStatus int
		issuedPEM   string
		issueBody   map[string]any
		issueQuery  url.Values
	}
	newGoogle := func(t *testing.T, issueStatus int) *google {
		g := &google{issueStatus: issueStatus, issuedPEM: issuedPEM}
		g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.mu.Lock()
			hang := g.tokenHang
			g.mu.Unlock()
			if hang != nil && r.URL.Path == "/token" {
				<-hang
				return
			}
			g.mu.Lock()
			defer g.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/token" {
				g.tokenCalls++
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600})
				return
			}
			if r.Method != http.MethodPost || r.URL.Path != certificatesPath || r.Header.Get("Authorization") != "Bearer test-token" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			g.issueQuery = r.URL.Query()
			_ = json.NewDecoder(r.Body).Decode(&g.issueBody)
			w.WriteHeader(g.issueStatus)
			if g.issueStatus == http.StatusOK {
				_ = json.NewEncoder(w).Encode(map[string]any{"name": "issued", "pemCertificate": g.issuedPEM, "pemCertificateChain": []string{g.issuedPEM}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": g.issueStatus, "message": "from google"}})
		}))
		t.Cleanup(g.srv.Close)
		return g
	}
	newCA := func(t *testing.T) fleet.GoogleCASCA {
		return fleet.GoogleCASCA{Name: "WIFI", Project: "my-project", Location: "us-central1", CAPool: "my-pool", APIKeyJSON: testServiceAccountKey(t)}
	}

	t.Run("issues the certificate and reuses the token", func(t *testing.T) {
		g := newGoogle(t, http.StatusOK)
		svc, ca := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")), newCA(t)

		cert, err := svc.IssueCertificate(t.Context(), ca, csr)
		require.NoError(t, err)
		require.Equal(t, issuedDER, cert.Raw)
		require.Equal(t, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})), g.issueBody["pemCsr"])
		require.Equal(t, "31536000s", g.issueBody["lifetime"])
		require.Regexp(t, `^fleet-[0-9a-f-]{36}$`, g.issueQuery.Get("certificateId"))
		firstID := g.issueQuery.Get("certificateId")

		_, err = svc.IssueCertificate(t.Context(), ca, csr)
		require.NoError(t, err)
		require.Equal(t, 1, g.tokenCalls, "the second request reuses the cached token")
		require.NotEqual(t, firstID, g.issueQuery.Get("certificateId"), "every certificate gets its own ID")
	})

	for _, tc := range []struct {
		name          string
		status        int
		wantErr       string
		wantTransient bool
	}{
		{"rejected by policy", http.StatusBadRequest, "Google CAS rejected the certificate request", false},
		{"no permission", http.StatusForbidden, "can't issue certificates from the CA pool", false},
		{"pool missing", http.StatusNotFound, "The certificate authority's CA pool wasn't found.", false},
		{"rate limited", http.StatusTooManyRequests, "Google CAS couldn't issue the certificate (HTTP 429). Try again later.", true},
		{"google outage", http.StatusServiceUnavailable, "Google CAS couldn't issue the certificate (HTTP 503). Try again later.", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGoogle(t, tc.status)
			_, err := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")).IssueCertificate(t.Context(), newCA(t), csr)
			require.ErrorContains(t, err, tc.wantErr)
			require.NotContains(t, err.Error(), "from google")
			transient, ok := errors.AsType[fleet.CertificateAuthorityTransientError](err)
			require.Equal(t, tc.wantTransient, ok)
			if ok {
				require.Equal(t, 30, transient.RetryAfter())
			}
		})
	}

	t.Run("Google unreachable is transient", func(t *testing.T) {
		g := newGoogle(t, http.StatusOK)
		ca := newCA(t)
		g.srv.Close()
		_, err := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")).IssueCertificate(t.Context(), ca, csr)
		transient, ok := errors.AsType[fleet.CertificateAuthorityTransientError](err)
		require.True(t, ok, err)
		require.Equal(t, "Couldn't connect to Google CAS. Try again later.", transient.Message)
	})

	t.Run("token endpoint outage is transient", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(srv.Close)
		ca := fleet.GoogleCASCA{Name: "WIFI", Project: "my-project", Location: "us-central1", CAPool: "my-pool", APIKeyJSON: testServiceAccountKey(t)}
		_, err := NewService(WithEndpoint(srv.URL), WithTokenURL(srv.URL+"/token")).IssueCertificate(t.Context(), ca, csr)
		transient, ok := errors.AsType[fleet.CertificateAuthorityTransientError](err)
		require.True(t, ok, err)
		require.Equal(t, "Google couldn't issue an access token (HTTP 503). Try again later.", transient.Message)
	})

	t.Run("a CA's client is replaced when its key changes", func(t *testing.T) {
		g := newGoogle(t, http.StatusOK)
		svc, ca := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")), newCA(t)
		ca.ID = 7
		_, err := svc.IssueCertificate(t.Context(), ca, csr)
		require.NoError(t, err)
		ca.APIKeyJSON = testServiceAccountKey(t)
		_, err = svc.IssueCertificate(t.Context(), ca, csr)
		require.NoError(t, err)
		require.Equal(t, 2, g.tokenCalls, "the rotated key gets its own token")
		require.Len(t, svc.issueClients, 1, "the old key's client is dropped")
	})

	t.Run("concurrent requests share one client", func(t *testing.T) {
		g := newGoogle(t, http.StatusOK)
		svc, ca := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")), newCA(t)
		ca.ID = 7
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				_, err := svc.IssueCertificate(t.Context(), ca, csr)
				assert.NoError(t, err)
			})
		}
		wg.Wait()
		require.Len(t, svc.issueClients, 1)
		g.mu.Lock()
		defer g.mu.Unlock()
		require.Equal(t, 1, g.tokenCalls, "one client, so one token for all the requests")
	})

	t.Run("an unexpected certificate is refused", func(t *testing.T) {
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		otherDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{SerialNumber: big.NewInt(43), NotAfter: time.Now().Add(time.Hour)},
			&x509.Certificate{SerialNumber: big.NewInt(1)}, &otherKey.PublicKey, otherKey)
		require.NoError(t, err)
		issued := func(template *x509.Certificate) []byte {
			template.SerialNumber, template.NotAfter = big.NewInt(44), time.Now().Add(time.Hour)
			der, err := x509.CreateCertificate(rand.Reader, template, &x509.Certificate{SerialNumber: big.NewInt(1)}, &csrKey.PublicKey, csrKey)
			require.NoError(t, err)
			return der
		}
		for name, tc := range map[string]struct {
			der     []byte
			wantErr string
		}{
			"for another public key":       {otherDER, "different public key"},
			"for a CA":                     {issued(&x509.Certificate{IsCA: true, BasicConstraintsValid: true}), "can sign other certificates"},
			"that can sign certificates":   {issued(&x509.Certificate{KeyUsage: x509.KeyUsageCertSign}), "can sign other certificates"},
			"that can sign CRLs":           {issued(&x509.Certificate{KeyUsage: x509.KeyUsageCRLSign}), "can sign other certificates"},
			"that can sign OCSP responses": {issued(&x509.Certificate{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}}), "can sign other certificates"},
		} {
			t.Run(name, func(t *testing.T) {
				g := newGoogle(t, http.StatusOK)
				g.issuedPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tc.der}))
				_, err := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")).IssueCertificate(t.Context(), newCA(t), csr)
				require.ErrorContains(t, err, tc.wantErr)
			})
		}
	})

	t.Run("a hanging token endpoint is bounded by the timeout and is transient", func(t *testing.T) {
		g := newGoogle(t, http.StatusOK)
		g.tokenHang = make(chan struct{})
		t.Cleanup(func() { close(g.tokenHang) }) // runs before the server's cleanup, so it can shut down
		start := time.Now()
		_, err := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token"), WithTimeout(300*time.Millisecond)).IssueCertificate(t.Context(), newCA(t), csr)
		require.Less(t, time.Since(start), 10*time.Second)
		_, ok := errors.AsType[fleet.CertificateAuthorityTransientError](err)
		require.True(t, ok, err)
	})

	t.Run("a key that isn't a service account is refused and not cached", func(t *testing.T) {
		g := newGoogle(t, http.StatusOK)
		svc, ca := NewService(WithEndpoint(g.srv.URL), WithTokenURL(g.srv.URL+"/token")), newCA(t)
		ca.APIKeyJSON = `{"type":"authorized_user","client_id":"a","client_secret":"b","refresh_token":"c"}`
		_, err := svc.IssueCertificate(t.Context(), ca, csr)
		require.ErrorContains(t, err, `"type" set to "service_account"`)
		require.Empty(t, svc.issueClients)
	})
}
