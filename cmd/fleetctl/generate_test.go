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
	outdir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	defer os.Remove(outdir)
	publicKeyPath := filepath.Join(outdir, "public-key.crt")
	_, _ = runServerWithMockedDS(t)

	out := runAppForTest(t, []string{
		"generate", "mdm-apple-bm",
		"--public-key", publicKeyPath,
	})

	require.Contains(t, out, fmt.Sprintf("Generated your public key at %s", outdir))

	// validate that the certificate is valid
	certPEM, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	require.NotNil(t, parsed)
}

func TestGenerateMDMApple(t *testing.T) {
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
			},
			ErrGeneric.Error(),
		)
	})

	t.Run("successful run", func(t *testing.T) {
		_, _ = runServerWithMockedDS(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"csr": "dGVzdAo="}`))
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

		require.Contains(t, out, fmt.Sprintf("Generated your certificate signing request (CSR) at %s", csrPath))

		// validate that the CSR is valid
		csrPEM, err := os.ReadFile(csrPath)
		require.NoError(t, err)
		require.Equal(t, "test\n", string(csrPEM))
	})
}
