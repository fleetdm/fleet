package testinfra

import (
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/testinfra/acmeclient"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testinfra/stepca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/acme"
)

// TestStepCASmoke verifies that the step-ca test server starts and serves
// a valid ACME directory.
func TestStepCASmoke(t *testing.T) {
	ca := stepca.New(t)
	dir := ca.VerifyDirectory(t)

	t.Logf("ACME directory: %v", dir)
	assert.Contains(t, dir, "newNonce")
	assert.Contains(t, dir, "newAccount")
	assert.Contains(t, dir, "newOrder")
}

// TestACMEClientDiscover verifies the test ACME client can discover the
// directory from a running step-ca.
func TestACMEClientDiscover(t *testing.T) {
	ca := stepca.New(t)

	client, err := acmeclient.New(ca.DirectoryURL(), acmeclient.WithTLSConfig(ca.TLSConfig()))
	require.NoError(t, err)

	dir, err := client.Discover()
	require.NoError(t, err)

	assert.NotEmpty(t, dir.NonceURL)
	assert.NotEmpty(t, dir.RegURL)
	assert.NotEmpty(t, dir.OrderURL)
}

// TestACMEEndToEnd exercises the complete ACME certificate issuance flow:
// client -> step-ca (discover, register, order, challenge, finalize, download).
func TestACMEEndToEnd(t *testing.T) {
	ca := stepca.New(t)

	client, err := acmeclient.New(ca.DirectoryURL(), acmeclient.WithTLSConfig(ca.TLSConfig()))
	require.NoError(t, err)

	_, err = client.RegisterAccount("mailto:test@example.com")
	require.NoError(t, err)

	challengeResponses := make(map[string]string)
	challengeServer := startChallengeServer(t, challengeResponses)
	defer challengeServer.Close()

	domain := "localhost"

	certs, err := client.OrderCertificate(domain, func(challenge *acme.Challenge) error {
		path := client.HTTP01ChallengePath(challenge)
		resp, err := client.HTTP01ChallengeResponse(challenge)
		if err != nil {
			return err
		}
		challengeResponses[path] = resp
		return nil
	})
	require.NoError(t, err, "full ACME flow failed")

	require.NotEmpty(t, certs)
	leaf := certs[0]
	assert.Equal(t, domain, leaf.Subject.CommonName)
	assert.Contains(t, leaf.DNSNames, domain)
	assert.GreaterOrEqual(t, len(certs), 2)

	pemData := acmeclient.CertificateToPEM(certs)
	assert.Contains(t, string(pemData), "BEGIN CERTIFICATE")
	t.Logf("Certificate issued: Subject=%s Serial=%s (%d certs, %d bytes PEM)",
		leaf.Subject.CommonName, leaf.SerialNumber, len(certs), len(pemData))
}

func startChallengeServer(t *testing.T, responses map[string]string) *http.Server {
	t.Helper()

	listener, ok := ListenPort80(t)
	if !ok {
		t.Skip("cannot bind port 80 (run: sudo sysctl net.ipv4.ip_unprivileged_port_start=80)")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		if resp, ok := responses[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte(resp))
		} else {
			http.NotFound(w, r)
		}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	return srv
}
