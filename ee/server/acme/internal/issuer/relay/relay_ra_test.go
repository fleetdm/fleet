package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/acme/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelaySmallstepRA tests the full relay flow against a local step-ca RA
// that is linked to Smallstep's hosted Certificate Manager.
//
// The RA handles ACME locally and forwards signing to the cloud.
// Certificates are signed by the Smallstep cloud CA (real PKI chain).
//
// Requires:
//   - step-ca RA running on localhost:8443 (started via Docker with linked token)
//   - SMALLSTEP_RA_TEST=1 environment variable
//   - RA root cert at claude-plans/acme/smallstep/secrets/ra-root.pem
func TestRelaySmallstepRA(t *testing.T) {
	if _, ok := os.LookupEnv("SMALLSTEP_RA_TEST"); !ok {
		t.Skip("set SMALLSTEP_RA_TEST=1 to run (requires step-ca RA on localhost:8443)")
	}

	// Load RA root cert
	rootCertPath := "../../../../claude-plans/acme/smallstep/secrets/ra-root.pem"
	caCert, err := os.ReadFile(rootCertPath)
	if err != nil {
		// Try absolute path
		caCert, err = os.ReadFile("/root/fleet/claude-plans/acme/smallstep/secrets/ra-root.pem")
		require.NoError(t, err, "cannot read RA root cert")
	}

	// The RA provisioner name from the Smallstep UI
	provisioner := "Victor's Registration Authority-acme"
	directoryURL := "https://localhost:8443/acme/" + provisioner + "/directory"

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	backend := New(logger)

	err = backend.AddCA(&api.CAConfig{
		Name:         "smallstep-ra",
		Type:         "relay",
		DirectoryURL: directoryURL,
		CACert:       caCert,
	})
	require.NoError(t, err)

	// Start http-01 challenge server on port 80
	challengeServer, available := startRAChallenge(t, backend)
	if !available {
		t.Skip("cannot bind port 80 for http-01 challenges")
	}
	defer challengeServer.Close()

	// Generate CSR
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	domain := "localhost"
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}, certKey)
	require.NoError(t, err)

	parsedCSR, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)

	order := &api.Order{
		ID:     "ra-test-order",
		CAName: "smallstep-ra",
		Status: "ready",
		Identifiers: []api.Identifier{
			{Type: "dns", Value: domain},
		},
	}

	// Issue certificate: Fleet relay → local RA → Smallstep cloud
	ctx := context.Background()
	issued, err := backend.IssueCertificate(ctx, parsedCSR, order)
	require.NoError(t, err, "IssueCertificate via Smallstep RA should succeed")

	require.NotNil(t, issued)
	require.NotEmpty(t, issued.DERChain)
	require.NotNil(t, issued.Leaf)

	t.Logf("Certificate issued via Smallstep cloud (through RA):")
	t.Logf("  Subject:      %s", issued.Leaf.Subject.CommonName)
	t.Logf("  Issuer:       %s", issued.Leaf.Issuer.CommonName)
	t.Logf("  Serial:       %s", issued.SerialNumber)
	t.Logf("  NotBefore:    %s", issued.NotBefore)
	t.Logf("  NotAfter:     %s", issued.NotAfter)
	t.Logf("  DNSNames:     %v", issued.Leaf.DNSNames)
	t.Logf("  Chain length: %d", len(issued.DERChain))

	assert.Equal(t, domain, issued.Leaf.Subject.CommonName)
	assert.Contains(t, issued.Leaf.DNSNames, domain)
}

func startRAChallenge(t *testing.T, backend *Backend) (*http.Server, bool) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:80")
	if err != nil {
		return nil, false
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Path[len("/.well-known/acme-challenge/"):]
		keyAuth, err := backend.HTTP01ChallengeResponse("smallstep-ra", token)
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
