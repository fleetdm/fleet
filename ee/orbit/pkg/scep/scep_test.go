package scep

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	_ "embed"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	challengePassword = "test-challenge"
)

// TestNewClientValidation tests the validation of parameters in the NewClient function
func TestNewClientValidation(t *testing.T) {
	signingKey, err := newSigningKey()
	require.NoError(t, err, "Failed to create test signing key")

	// Define test cases
	testCases := []struct {
		name        string
		options     []Option
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing common name",
			options: []Option{
				WithSigningKey(signingKey),
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithTimeout(ptr.Duration(5 * time.Second)),
			},
			expectError: true,
			errorMsg:    "should fail without commonName",
		},
		{
			name: "missing URL",
			options: []Option{
				WithSigningKey(signingKey),
				WithURL(""),
				WithChallenge("test-challenge"),
				WithCommonName("test-device"),
			},
			expectError: true,
			errorMsg:    "should fail with empty URL",
		},
		{
			name: "missing SecureHW",
			options: []Option{
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithCommonName("test-device"),
				WithTimeout(ptr.Duration(5 * time.Second)),
			},
			expectError: true,
			errorMsg:    "should fail without SecureHW",
		},
		{
			name: "all required parameters",
			options: []Option{
				WithSigningKey(signingKey),
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithCommonName("test-device"),
				WithTimeout(ptr.Duration(5 * time.Second)),
			},
			expectError: false,
			errorMsg:    "should succeed with all required parameters",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.options...)
			if tc.expectError {
				require.Error(t, err, "NewClient "+tc.errorMsg)
				require.Nil(t, client, "Client should be nil when error occurs")
			} else {
				require.NoError(t, err, "NewClient "+tc.errorMsg)
				require.NotNil(t, client, "Client should not be nil when no error occurs")
			}
		})
	}
}

// TestClient_FetchCert tests the successful retrieval of a certificate using SCEP.
func TestClient_FetchCert(t *testing.T) {
	// Start a test SCEP server
	scepServer := StartTestSCEPServer(t)
	defer scepServer.Close()

	// Create a logger for testing
	logger := zerolog.New(zerolog.NewTestWriter(t))

	t.Run("successful fetch", func(t *testing.T) {
		signingKey, err := newSigningKey()
		require.NoError(t, err, "Failed to create test SecureHW")

		// Create a SCEP client with all required parameters
		client, err := NewClient(
			WithSigningKey(signingKey),
			WithURL(scepServer.URL+"/scep"),
			WithChallenge(challengePassword),
			WithLogger(logger),
			WithTimeout(ptr.Duration(5*time.Second)),
			WithCommonName("test-device"),
		)
		require.NoError(t, err, "NewClient should succeed with all required parameters")

		// Fetch and save the certificate
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		clientCert, err := client.FetchCert(ctx)
		require.NoError(t, err, "FetchCert should succeed")
		require.NotNil(t, clientCert)

		//
		// Verify certificate content
		//

		require.Equal(t, "test-device", clientCert.Subject.CommonName, "Certificate should have the correct common name")
		// Verify the certificate has the correct ExtKeyUsage
		assert.Contains(t, clientCert.ExtKeyUsage, x509.ExtKeyUsageClientAuth, "Certificate should have ExtKeyUsageClientAuth")
		// Verify the certificate was signed by the CA (the CA in our test uses RSA)
		// The CSR was signed with our ECC key, but the final certificate is signed by the CA
		assert.Equal(t, x509.SHA256WithRSA, clientCert.SignatureAlgorithm, "Certificate should be signed by CA with RSA")
	})

	t.Run("bad challenge password", func(t *testing.T) {
		signingKey, err := newSigningKey()
		require.NoError(t, err, "Failed to create test SecureHW")

		// Create a SCEP client with all required parameters
		client, err := NewClient(
			WithSigningKey(signingKey),
			WithURL(scepServer.URL+"/scep"),
			WithChallenge("BAD"),
			WithLogger(logger),
			WithTimeout(ptr.Duration(5*time.Second)),
			WithCommonName("test-device"),
		)
		require.NoError(t, err, "NewClient should succeed with all required parameters")
		_, err = client.FetchCert(t.Context())
		assert.ErrorContains(t, err, "PKIMessage CSR request failed", "FetchAndSaveCert should fail with bad challenge password")
	})
}

//go:embed testdata/ca.crt
var caCert []byte

//go:embed testdata/ca.key
var caKey []byte

//go:embed testdata/ca.pem
var caPem []byte

func StartTestSCEPServer(t *testing.T) *httptest.Server {
	caDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(caDir, "ca.crt"), caCert, 0o644); err != nil {
		t.Fatalf("failed to write ca.crt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.key"), caKey, 0o644); err != nil {
		t.Fatalf("failed to write ca.key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.pem"), caPem, 0o644); err != nil {
		t.Fatalf("failed to write ca.pem: %v", err)
	}

	newSCEPServer := func(t *testing.T) *httptest.Server {
		var server *httptest.Server
		t.Cleanup(func() {
			if server != nil {
				server.Close()
			}
		})

		certDepot, err := filedepot.NewFileDepot(caDir)
		if err != nil {
			t.Fatal(err)
		}
		crt, key, err := certDepot.CA([]byte{})
		if err != nil {
			t.Fatal(err)
		}

		signer := scepserver.StaticChallengeMiddleware(challengePassword, scepserver.SignCSRAdapter(depot.NewSigner(certDepot)))
		svc, err := scepserver.NewService(crt[0], key, signer)
		if err != nil {
			t.Fatal(err)
		}
		logger := kitlog.NewNopLogger()
		e := scepserver.MakeServerEndpoints(svc)
		scepHandler := scepserver.MakeHTTPHandler(e, svc, logger)
		r := mux.NewRouter()
		r.Handle("/scep", scepHandler)
		server = httptest.NewServer(r)
		return server
	}
	scepServer := newSCEPServer(t)
	return scepServer
}

// testKey implements the SigningKey interface for testing.
type testSigningKey struct {
	key *ecdsa.PrivateKey
}

// testSigner implements crypto.Signer for testing
type testSigner struct {
	key *ecdsa.PrivateKey
}

// newSigningKey creates a new test signing key implementation with an ECC P-384 key
func newSigningKey() (*testSigningKey, error) {
	// Create ECC P-384 key in memory
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &testSigningKey{key: key}, nil
}

// Signer implements securehw.Key.Signer
func (k *testSigningKey) Signer() (crypto.Signer, error) {
	return &testSigner{key: k.key}, nil
}

// Public implements crypto.Signer.Public
func (s *testSigner) Public() crypto.PublicKey {
	return &s.key.PublicKey
}

// Sign implements crypto.Signer.Sign
func (s *testSigner) Sign(rand io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return ecdsa.SignASN1(rand, s.key, digest)
}
