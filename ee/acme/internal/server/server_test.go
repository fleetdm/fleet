package server

import (
	"bytes"
	gocontext "context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/issuer/relay"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testinfra"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testinfra/stepca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerDirectory verifies the server returns a valid ACME directory.
func TestServerDirectory(t *testing.T) {
	srv, client, baseURL := startTestServer(t, nil)
	_ = srv

	resp, err := client.Get(baseURL + "/testca/directory")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var dir map[string]interface{}
	require.NoError(t, readJSON(resp, &dir))

	assert.Contains(t, dir["newNonce"], "/testca/new-nonce")
	assert.Contains(t, dir["newAccount"], "/testca/new-account")
	assert.Contains(t, dir["newOrder"], "/testca/new-order")
	assert.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
}

// TestServerNewNonce verifies nonce generation.
func TestServerNewNonce(t *testing.T) {
	_, client, baseURL := startTestServer(t, nil)

	req, _ := http.NewRequest("HEAD", baseURL+"/testca/new-nonce", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	nonce1 := resp.Header.Get("Replay-Nonce")
	assert.NotEmpty(t, nonce1)

	// Second request should return a different nonce
	resp2, err := client.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	nonce2 := resp2.Header.Get("Replay-Nonce")
	assert.NotEqual(t, nonce1, nonce2)
}

// TestServerNewAccount verifies account creation.
func TestServerNewAccount(t *testing.T) {
	_, client, baseURL := startTestServer(t, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"contact": []string{"mailto:test@example.com"},
	})
	resp, err := client.Post(baseURL+"/testca/new-account", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var acct map[string]interface{}
	require.NoError(t, readJSON(resp, &acct))
	assert.Equal(t, "valid", acct["status"])
	assert.NotEmpty(t, resp.Header.Get("Location"))
}

// TestServerFullFlow tests the complete ACME flow through the server with
// a real relay backend and step-ca upstream.
//
// ACME client → Fleet ACME Server → RelayBackend → step-ca
func TestServerFullFlow(t *testing.T) {
	// Start upstream step-ca
	ca := stepca.New(t)

	// Create relay backend
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	backend := relay.New(logger)
	require.NoError(t, backend.AddCA(&api.CAConfig{
		Name:         "testca",
		Type:         "relay",
		DirectoryURL: ca.DirectoryURL(),
		CACert:       ca.RootCACert(),
	}))

	// Start challenge server for http-01 (relay needs this for upstream validation)
	challengeServer, available := startChallengeServerForRelay(t, backend)
	if !available {
		t.Skip("cannot bind port 80 for http-01 challenges")
	}
	defer challengeServer.Close()

	// Start ACME server with relay backend
	_, client, baseURL := startTestServer(t, backend)

	// 1. Get directory
	resp, err := client.Get(baseURL + "/testca/directory")
	require.NoError(t, err)
	var dir map[string]interface{}
	require.NoError(t, readJSON(resp, &dir))
	resp.Body.Close()
	t.Logf("Directory: %v", dir)

	// 2. Create account
	resp, err = client.Post(baseURL+"/testca/new-account", "application/json",
		jsonBody(map[string]interface{}{"contact": []string{"mailto:test@fleet.local"}}))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	t.Log("Account created")

	// 3. Create order
	resp, err = client.Post(baseURL+"/testca/new-order", "application/json",
		jsonBody(map[string]interface{}{
			"identifiers": []map[string]string{{"type": "dns", "value": "localhost"}},
		}))
	require.NoError(t, err)
	var order map[string]interface{}
	require.NoError(t, readJSON(resp, &order))
	resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "pending", order["status"])
	t.Logf("Order created: status=%s", order["status"])

	// 4. Get authorization
	authzURLs := order["authorizations"].([]interface{})
	require.NotEmpty(t, authzURLs)
	resp, err = client.Get(authzURLs[0].(string))
	require.NoError(t, err)
	var authz map[string]interface{}
	require.NoError(t, readJSON(resp, &authz))
	resp.Body.Close()
	assert.Equal(t, "pending", authz["status"])

	challenges := authz["challenges"].([]interface{})
	require.NotEmpty(t, challenges)
	challengeData := challenges[0].(map[string]interface{})
	challengeURL := challengeData["url"].(string)
	t.Logf("Challenge: type=%s url=%s", challengeData["type"], challengeURL)

	// 5. Respond to challenge (POST to challenge URL)
	resp, err = client.Post(challengeURL, "application/json", jsonBody(map[string]interface{}{}))
	require.NoError(t, err)
	var chResp map[string]interface{}
	require.NoError(t, readJSON(resp, &chResp))
	resp.Body.Close()
	assert.Equal(t, "valid", chResp["status"])
	t.Logf("Challenge validated: status=%s", chResp["status"])

	// 6. Poll order until ready
	orderURL := resp.Header.Get("Location")
	if orderURL == "" {
		// Derive from finalize URL
		orderURL = order["finalize"].(string)
		orderURL = orderURL[:len(orderURL)-len("/finalize")]
	}

	var updatedOrder map[string]interface{}
	for i := 0; i < 10; i++ {
		resp, err = client.Get(orderURL)
		require.NoError(t, err)
		require.NoError(t, readJSON(resp, &updatedOrder))
		resp.Body.Close()
		status := updatedOrder["status"].(string)
		t.Logf("Order poll [%d]: status=%s", i, status)
		if status == "ready" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.Equal(t, "ready", updatedOrder["status"], "order should be ready after challenge validation")

	// 7. Finalize with CSR
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "localhost"},
		DNSNames: []string{"localhost"},
	}, certKey)
	require.NoError(t, err)

	finalizeURL := order["finalize"].(string)
	resp, err = client.Post(finalizeURL, "application/json",
		jsonBody(map[string]string{"csr": base64.RawURLEncoding.EncodeToString(csrDER)}))
	require.NoError(t, err)
	var finalOrder map[string]interface{}
	require.NoError(t, readJSON(resp, &finalOrder))
	resp.Body.Close()
	assert.Equal(t, "valid", finalOrder["status"], "order should be valid after finalize")
	certURL, ok := finalOrder["certificate"].(string)
	require.True(t, ok, "order should have certificate URL")
	t.Logf("Order finalized: status=%s cert=%s", finalOrder["status"], certURL)

	// 8. Download certificate
	resp, err = client.Get(certURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pem-certificate-chain", resp.Header.Get("Content-Type"))

	certPEM, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(certPEM), "BEGIN CERTIFICATE")

	// Parse and verify the certificate
	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block, "should have PEM certificate")
	leaf, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	t.Logf("Certificate issued via full stack:")
	t.Logf("  Subject:  %s", leaf.Subject.CommonName)
	t.Logf("  Issuer:   %s", leaf.Issuer.CommonName)
	t.Logf("  Serial:   %s", leaf.SerialNumber)
	t.Logf("  DNSNames: %v", leaf.DNSNames)

	assert.Equal(t, "localhost", leaf.Subject.CommonName)
	assert.Contains(t, leaf.DNSNames, "localhost")
}

// TestServerUnknownCA verifies that requests to unknown CAs are rejected.
func TestServerUnknownCA(t *testing.T) {
	_, client, baseURL := startTestServer(t, nil)

	resp, err := client.Get(baseURL + "/unknownca/directory")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- Helpers ---

// noopIssuer is a CertificateIssuer that does nothing (for unit tests that
// don't need actual certificate issuance).
type noopIssuer struct{}

func (n *noopIssuer) ValidateChallenge(_ gocontext.Context, _ *api.Challenge, _ *api.Order) error {
	return nil
}
func (n *noopIssuer) IssueCertificate(_ gocontext.Context, _ *x509.CertificateRequest, _ *api.Order) (*api.IssuedCertificate, error) {
	return nil, fmt.Errorf("not implemented")
}
func (n *noopIssuer) RevokeCertificate(_ gocontext.Context, _ *x509.Certificate, _ int) error {
	return nil
}

// startTestServer creates an ACME server with TLS, wired to the given issuer.
// If issuer is nil, a noop issuer is used.
func startTestServer(t *testing.T, issuer api.CertificateIssuer) (*Server, *http.Client, string) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	proxyTLS, proxyCert := selfSignedTLSConfig()

	mux := http.NewServeMux()
	httpServer := httptest.NewUnstartedServer(mux)
	httpServer.TLS = &tls.Config{Certificates: []tls.Certificate{proxyCert}}
	httpServer.StartTLS()
	t.Cleanup(httpServer.Close)

	acmeServer := New(httpServer.URL, logger)
	if issuer == nil {
		issuer = &noopIssuer{}
	}
	acmeServer.RegisterIssuer("testca", issuer)
	mux.Handle(PathPrefix, acmeServer)

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: proxyTLS},
	}

	baseURL := httpServer.URL + PathPrefix[:len(PathPrefix)-1]
	return acmeServer, client, baseURL
}

func selfSignedTLSConfig() (*tls.Config, tls.Certificate) {
	tmpSrv := httptest.NewTLSServer(http.NotFoundHandler())
	cert := tmpSrv.TLS.Certificates[0]
	clientTLS := tmpSrv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	tmpSrv.Close()
	return clientTLS, cert
}

func startChallengeServerForRelay(t *testing.T, backend *relay.Backend) (*http.Server, bool) {
	t.Helper()

	listener, ok := testinfra.ListenPort80(t)
	if !ok {
		return nil, false
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Path[len("/.well-known/acme-challenge/"):]
		keyAuth, err := backend.HTTP01ChallengeResponse("testca", token)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte(keyAuth))
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	return srv, true
}

func readJSON(resp *http.Response, v interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func jsonBody(v interface{}) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}
