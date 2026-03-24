package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/acme/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelaySmallstepCloud tests the full relay flow against Smallstep's
// hosted Certificate Manager with EAB authentication.
//
// Requires environment variables:
//
//	SMALLSTEP_CA_URL       - e.g., https://victor.fleetdm-nfr.ca.smallstep.com
//	SMALLSTEP_PROVISIONER  - e.g., fleet-relay
//	SMALLSTEP_EAB_KEY_ID   - EAB key ID from `step ca acme eab add`
//	SMALLSTEP_EAB_HMAC_KEY - EAB HMAC key (base64url-encoded)
func TestRelaySmallstepCloud(t *testing.T) {
	caURL := os.Getenv("SMALLSTEP_CA_URL")
	provisioner := os.Getenv("SMALLSTEP_PROVISIONER")
	eabKeyID := os.Getenv("SMALLSTEP_EAB_KEY_ID")
	eabHMACKey := os.Getenv("SMALLSTEP_EAB_HMAC_KEY")

	if caURL == "" || eabKeyID == "" || eabHMACKey == "" {
		t.Skip("set SMALLSTEP_CA_URL, SMALLSTEP_PROVISIONER, SMALLSTEP_EAB_KEY_ID, SMALLSTEP_EAB_HMAC_KEY to run")
	}
	if provisioner == "" {
		provisioner = "fleet-relay"
	}

	directoryURL := caURL + "/acme/" + provisioner + "/directory"
	t.Logf("Directory URL: %s", directoryURL)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	backend := New(logger)

	// Load the Smallstep root CA cert for TLS verification.
	// The step CLI stores it at ~/.step/certs/root_ca.crt after bootstrap.
	var caCert []byte
	home, _ := os.UserHomeDir()
	rootCAPath := home + "/.step/certs/root_ca.crt"
	caCert, err := os.ReadFile(rootCAPath)
	if err != nil {
		t.Logf("Warning: could not read %s: %v (will use system roots)", rootCAPath, err)
	}

	err = backend.AddCA(&api.CAConfig{
		Name:         "smallstep-cloud",
		Type:         "relay",
		DirectoryURL: directoryURL,
		EABKeyID:     eabKeyID,
		EABHMACKey:   eabHMACKey,
		CACert:       caCert,
	})
	require.NoError(t, err)

	// Generate a CSR for a test domain
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	domain := "test.fleet.local"
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}, certKey)
	require.NoError(t, err)

	parsedCSR, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)

	order := &api.Order{
		ID:     "cloud-test-order",
		CAName: "smallstep-cloud",
		Status: "ready",
		Identifiers: []api.Identifier{
			{Type: "dns", Value: domain},
		},
	}

	// Issue certificate through Smallstep cloud
	ctx := context.Background()
	issued, err := backend.IssueCertificate(ctx, parsedCSR, order)
	require.NoError(t, err, "IssueCertificate via Smallstep cloud should succeed")

	require.NotNil(t, issued)
	require.NotEmpty(t, issued.DERChain)
	require.NotNil(t, issued.Leaf)

	t.Logf("Certificate issued via Smallstep cloud:")
	t.Logf("  Subject:      %s", issued.Leaf.Subject.CommonName)
	t.Logf("  Issuer:       %s", issued.Leaf.Issuer.CommonName)
	t.Logf("  Serial:       %s", issued.SerialNumber)
	t.Logf("  NotBefore:    %s", issued.NotBefore)
	t.Logf("  NotAfter:     %s", issued.NotAfter)
	t.Logf("  DNSNames:     %v", issued.Leaf.DNSNames)
	t.Logf("  Chain length: %d", len(issued.DERChain))

	assert.Equal(t, domain, issued.Leaf.Subject.CommonName)
	assert.Contains(t, issued.Leaf.DNSNames, domain)
	assert.GreaterOrEqual(t, len(issued.DERChain), 2, "expected leaf + intermediate")
}
