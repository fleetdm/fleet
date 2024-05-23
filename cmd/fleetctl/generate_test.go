package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateMDMAppleBM(t *testing.T) {
	// TODO(roberto): update when the new endpoint to get a CSR is ready
	t.Skip()
	outdir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	defer os.Remove(outdir)
	publicKeyPath := filepath.Join(outdir, "public-key.crt")

	out := runAppForTest(t, []string{
		"generate", "mdm-apple-bm",
		"--public-key", publicKeyPath,
	})

	require.Contains(t, out, fmt.Sprintf("Generated your public key at %s", outdir))

	// validate that the certificate is valid
	certPEMBlock, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)

	parsed, err := x509.ParseCertificate(certPEMBlock)
	require.NoError(t, err)
	require.Equal(t, "FleetDM", parsed.Issuer.CommonName)
}

func TestGenerateMDMApple(t *testing.T) {
	t.Run("CSR API call fails", func(t *testing.T) {
		// TODO(roberto): update when the new endpoint to get a CSR is ready
		t.Skip()
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
			},
			`POST /api/latest/fleet/mdm/apple/request_csr received status 422 Validation Failed: this email address is not valid: bad request`,
		)
	})

	t.Run("successful run", func(t *testing.T) {
		// TODO(roberto): update when the new endpoint to get a CSR is ready
		t.Skip()
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
		csrPath := filepath.Join(outdir, "csr.csr")
		out := runAppForTest(t, []string{
			"generate", "mdm-apple",
			"--csr", csrPath,
			"--debug",
			"--context", "default",
		})

		require.Contains(t, out, fmt.Sprintf("Generated your SCEP key at %s", csrPath))

		// validate that the CSR is valid
		csrPEM, err := os.ReadFile(csrPath)
		require.NoError(t, err)

		block, _ := pem.Decode(csrPEM)
		require.NotNil(t, block)
		require.Equal(t, "CERTIFICATE REQUEST", block.Type)
		_, err = x509.ParseCertificateRequest(block.Bytes)
		require.NoError(t, err)
	})
}
