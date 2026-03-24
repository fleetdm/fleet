package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testinfra"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testinfra/stepca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelayImplementsInterface verifies that Backend satisfies the CertificateIssuer interface.
func TestRelayImplementsInterface(t *testing.T) {
	var _ api.CertificateIssuer = (*Backend)(nil)
}

// TestRelayIssueCertificate exercises the full certificate issuance flow through
// the relay backend: create upstream order, complete http-01 challenge, finalize
// with CSR, return certificate.
func TestRelayIssueCertificate(t *testing.T) {
	ca := stepca.New(t)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	backend := New(logger)

	err := backend.AddCA(&api.CAConfig{
		Name:         "testca",
		Type:         "relay",
		DirectoryURL: ca.DirectoryURL(),
		CACert:       ca.RootCACert(),
	})
	require.NoError(t, err)

	// Start http-01 challenge server on port 80
	// The relay needs to complete http-01 challenges with step-ca.
	// We serve the key authorization at /.well-known/acme-challenge/<token>
	challengeServer, available := startChallengeServer(t, backend)
	if !available {
		t.Skip("cannot bind port 80 for http-01 challenges")
	}
	defer challengeServer.Close()

	// Generate a CSR
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "localhost"},
		DNSNames: []string{"localhost"},
	}, certKey)
	require.NoError(t, err)

	parsedCSR, err := x509.ParseCertificateRequest(csr)
	require.NoError(t, err)

	// Create an order (simulating what the ACME server would do)
	order := &api.Order{
		ID:     "test-order-1",
		CAName: "testca",
		Status: "ready",
		Identifiers: []api.Identifier{
			{Type: "dns", Value: "localhost"},
		},
	}

	// Issue certificate through the relay
	ctx := context.Background()
	issued, err := backend.IssueCertificate(ctx, parsedCSR, order)
	require.NoError(t, err, "IssueCertificate should succeed")

	// Verify the issued certificate
	require.NotNil(t, issued)
	require.NotEmpty(t, issued.DERChain, "should have at least one cert in chain")
	require.NotNil(t, issued.Leaf, "should have parsed leaf cert")

	t.Logf("Certificate issued via relay:")
	t.Logf("  Subject:  %s", issued.Leaf.Subject.CommonName)
	t.Logf("  Issuer:   %s", issued.Leaf.Issuer.CommonName)
	t.Logf("  Serial:   %s", issued.SerialNumber.String())
	t.Logf("  NotAfter: %s", issued.NotAfter)
	t.Logf("  DNSNames: %v", issued.Leaf.DNSNames)

	assert.Equal(t, "localhost", issued.Leaf.Subject.CommonName)
	assert.Contains(t, issued.Leaf.DNSNames, "localhost")
	assert.GreaterOrEqual(t, len(issued.DERChain), 2, "expected leaf + intermediate")
}

// TestRelayValidateChallenge verifies that challenge validation succeeds (POC: always passes).
func TestRelayValidateChallenge(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	backend := New(logger)

	challenge := &api.Challenge{
		ID:   "ch-1",
		Type: "device-attest-01",
	}
	order := &api.Order{
		ID:     "order-1",
		CAName: "testca",
	}

	err := backend.ValidateChallenge(context.Background(), challenge, order)
	assert.NoError(t, err, "ValidateChallenge should succeed (POC: no-op)")
}

// TestRelayMultiCA verifies the relay can register multiple upstream CAs and
// route operations to the correct one.
func TestRelayMultiCA(t *testing.T) {
	ca1 := stepca.New(t)
	ca2 := stepca.New(t)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	backend := New(logger)

	require.NoError(t, backend.AddCA(&api.CAConfig{
		Name:         "ca1",
		Type:         "relay",
		DirectoryURL: ca1.DirectoryURL(),
		CACert:       ca1.RootCACert(),
	}))
	require.NoError(t, backend.AddCA(&api.CAConfig{
		Name:         "ca2",
		Type:         "relay",
		DirectoryURL: ca2.DirectoryURL(),
		CACert:       ca2.RootCACert(),
	}))

	// Verify both CAs are registered and can generate key auths
	_, err := backend.HTTP01ChallengeResponse("ca1", "test-token")
	require.NoError(t, err, "ca1 should be registered")

	_, err = backend.HTTP01ChallengeResponse("ca2", "test-token")
	require.NoError(t, err, "ca2 should be registered")

	// Key auths should differ (different account keys)
	auth1, _ := backend.HTTP01ChallengeResponse("ca1", "test-token")
	auth2, _ := backend.HTTP01ChallengeResponse("ca2", "test-token")
	assert.NotEqual(t, auth1, auth2, "different CAs should have different key auths")

	// Unknown CA should fail
	_, err = backend.HTTP01ChallengeResponse("unknown", "test-token")
	assert.Error(t, err)

	// ValidateChallenge routes correctly (POC: always succeeds)
	for _, caName := range []string{"ca1", "ca2"} {
		err := backend.ValidateChallenge(context.Background(),
			&api.Challenge{ID: "ch-1", Type: "device-attest-01"},
			&api.Order{ID: "order-1", CAName: caName},
		)
		assert.NoError(t, err, "ValidateChallenge should succeed for %s", caName)
	}

	t.Log("Both CAs registered and routing correctly")
}

// startChallengeServer starts an HTTP server on port 80 that serves http-01
// challenge responses using the relay backend's account key.
func startChallengeServer(t *testing.T, backend *Backend) (*http.Server, bool) {
	t.Helper()

	listener, ok := testinfra.ListenPort80(t)
	if !ok {
		return nil, false
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Path[len("/.well-known/acme-challenge/"):]
		t.Logf("Challenge request: token=%s", token)

		// Try all registered CAs to find the right key authorization
		backend.mu.RLock()
		defer backend.mu.RUnlock()
		for caName, upstream := range backend.upstreams {
			keyAuth, err := upstream.acmeClient.HTTP01ChallengeResponse(token)
			if err == nil {
				t.Logf("Serving key auth from CA %s", caName)
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write([]byte(keyAuth))
				return
			}
		}

		http.NotFound(w, r)
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	return srv, true
}
