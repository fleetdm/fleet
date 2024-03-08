package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateMDMAppleBM(t *testing.T) {
	outdir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	defer os.Remove(outdir)
	publicKeyPath := filepath.Join(outdir, "public-key.crt")
	privateKeyPath := filepath.Join(outdir, "private-key.key")
	out := runAppForTest(t, []string{
		"generate", "mdm-apple-bm",
		"--public-key", publicKeyPath,
		"--private-key", privateKeyPath,
	})

	require.Contains(t, out, fmt.Sprintf("Generated your public key at %s", outdir))
	require.Contains(t, out, fmt.Sprintf("Generated your private key at %s", outdir))

	// validate that the keypair is valid
	cert, err := tls.LoadX509KeyPair(publicKeyPath, privateKeyPath)
	require.NoError(t, err)

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	require.Equal(t, "FleetDM", parsed.Issuer.CommonName)
}

func TestGenerateMDMApple(t *testing.T) {
	t.Run("missing input", func(t *testing.T) {
		runAppCheckErr(t, []string{"generate", "mdm-apple"}, `Required flags "email, org" not set`)
		runAppCheckErr(t, []string{"generate", "mdm-apple", "--email", "user@example.com"}, `Required flag "org" not set`)
		runAppCheckErr(t, []string{"generate", "mdm-apple", "--org", "Acme"}, `Required flag "email" not set`)
	})

	t.Run("CSR API call fails", func(t *testing.T) {
		_, _ = runServerWithMockedDS(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fail this call
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		}))
		t.Setenv("TEST_FLEETDM_API_URL", srv.URL)
		t.Cleanup(srv.Close)
		runAppCheckErr(
			t,
			[]string{
				"generate", "mdm-apple",
				"--email", "user@example.com",
				"--org", "Acme",
			},
			`POST /api/latest/fleet/mdm/apple/request_csr received status 422 Validation Failed: this email address is not valid: bad request`,
		)
	})

	t.Run("successful run", func(t *testing.T) {
		_, _ = runServerWithMockedDS(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		t.Setenv("TEST_FLEETDM_API_URL", srv.URL)
		t.Cleanup(srv.Close)

		outdir, err := os.MkdirTemp("", "TestGenerateMDMApple")
		require.NoError(t, err)
		defer os.Remove(outdir)
		apnsKeyPath := filepath.Join(outdir, "apns.key")
		scepCertPath := filepath.Join(outdir, "scep.crt")
		scepKeyPath := filepath.Join(outdir, "scep.key")
		out := runAppForTest(t, []string{
			"generate", "mdm-apple",
			"--email", "user@example.com",
			"--org", "Acme",
			"--apns-key", apnsKeyPath,
			"--scep-cert", scepCertPath,
			"--scep-key", scepKeyPath,
			"--debug",
			"--context", "default",
		})

		require.Contains(t, out, fmt.Sprintf("Generated your APNs key at %s", apnsKeyPath))
		require.Contains(t, out, fmt.Sprintf("Generated your SCEP certificate at %s", scepCertPath))
		require.Contains(t, out, fmt.Sprintf("Generated your SCEP key at %s", scepKeyPath))

		// validate that the keypair is valid
		scepCrt, err := tls.LoadX509KeyPair(scepCertPath, scepKeyPath)
		require.NoError(t, err)
		parsed, err := x509.ParseCertificate(scepCrt.Certificate[0])
		require.NoError(t, err)
		require.Equal(t, "FleetDM", parsed.Issuer.CommonName)
	})
}
