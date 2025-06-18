package scep

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/tee"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
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
	certDir := t.TempDir()
	testTEEDevice, err := newTestTEE()
	require.NoError(t, err, "Failed to create test TEE")

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
				WithTEE(testTEEDevice),
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithCertDestDir(certDir),
				WithTimeout(5 * time.Second),
			},
			expectError: true,
			errorMsg:    "should fail without commonName",
		},
		{
			name: "missing URL",
			options: []Option{
				WithTEE(testTEEDevice),
				WithURL(""),
				WithChallenge("test-challenge"),
				WithCertDestDir(certDir),
				WithCommonName("test-device"),
			},
			expectError: true,
			errorMsg:    "should fail with empty URL",
		},
		{
			name: "missing cert destination directory",
			options: []Option{
				WithTEE(testTEEDevice),
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithCertDestDir(""),
				WithCommonName("test-device"),
			},
			expectError: true,
			errorMsg:    "should fail with empty certDestDir",
		},
		{
			name: "missing TEE",
			options: []Option{
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithCertDestDir(certDir),
				WithCommonName("test-device"),
				WithTimeout(5 * time.Second),
			},
			expectError: true,
			errorMsg:    "should fail without TEE",
		},
		{
			name: "all required parameters",
			options: []Option{
				WithTEE(testTEEDevice),
				WithURL("https://example.com/scep"),
				WithChallenge("test-challenge"),
				WithCertDestDir(certDir),
				WithCommonName("test-device"),
				WithTimeout(5 * time.Second),
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

// TestClient_FetchAndSaveCert tests the successful retrieval and saving of a certificate
func TestClient_FetchAndSaveCert(t *testing.T) {
	// Start a test SCEP server
	scepServer := StartTestSCEPServer(t)
	defer scepServer.Close()

	// Create a logger for testing
	logger := zerolog.New(zerolog.NewTestWriter(t))

	t.Run("successful fetch and save", func(t *testing.T) {
		// Create a temporary directory for storing certificates
		certDir := t.TempDir()

		// Create test TEE implementation
		testTEEDevice, err := newTestTEE()
		require.NoError(t, err, "Failed to create test TEE")

		// Create a SCEP client with all required parameters
		client, err := NewClient(
			WithTEE(testTEEDevice),
			WithURL(scepServer.URL+"/scep"),
			WithChallenge(challengePassword),
			WithCertDestDir(certDir),
			WithLogger(logger),
			WithTimeout(5*time.Second),
			WithCommonName("test-device"),
		)
		require.NoError(t, err, "NewClient should succeed with all required parameters")

		// Fetch and save the certificate
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		err = client.FetchAndSaveCert(ctx)
		require.NoError(t, err, "FetchAndSaveCert should succeed")

		// Verify that the certificate file was created
		certPath := filepath.Join(certDir, constant.FleetTLSClientCertificateFileName)

		// Check if the certificate file exists
		_, err = os.Stat(certPath)
		require.NoError(t, err, "Certificate file should exist")

		// Verify certificate content
		certData, err := os.ReadFile(certPath)
		require.NoError(t, err, "Should be able to read certificate file")
		certBlock, _ := pem.Decode(certData)
		require.NotNil(t, certBlock, "Certificate should be in PEM format")
		require.Equal(t, "CERTIFICATE", certBlock.Type, "Certificate block type should be CERTIFICATE")

		cert, err := x509.ParseCertificate(certBlock.Bytes)
		require.NoError(t, err, "Should be able to parse certificate")
		require.Equal(t, "test-device", cert.Subject.CommonName, "Certificate should have the correct common name")

		// Verify the certificate has the correct ExtKeyUsage
		assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth, "Certificate should have ExtKeyUsageClientAuth")

		// Verify the certificate was signed by the CA (the CA in our test uses RSA)
		// The CSR was signed with our ECC key, but the final certificate is signed by the CA
		assert.Equal(t, x509.SHA256WithRSA, cert.SignatureAlgorithm, "Certificate should be signed by CA with RSA")
	})

	t.Run("bad challenge password", func(t *testing.T) {
		// Create a temporary directory for storing certificates
		certDir := t.TempDir()

		// Create test TEE implementation
		testTEEDevice, err := newTestTEE()
		require.NoError(t, err, "Failed to create test TEE")

		// Create a SCEP client with all required parameters
		client, err := NewClient(
			WithTEE(testTEEDevice),
			WithURL(scepServer.URL+"/scep"),
			WithChallenge("BAD"),
			WithCertDestDir(certDir),
			WithLogger(logger),
			WithTimeout(5*time.Second),
			WithCommonName("test-device"),
		)
		require.NoError(t, err, "NewClient should succeed with all required parameters")
		err = client.FetchAndSaveCert(t.Context())
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

// testTEE implements the TEE interface for testing purposes using in-memory ECC P-384 keys
type testTEE struct {
	key *ecdsa.PrivateKey
}

// testKey implements the Key interface for testing
type testKey struct {
	key *ecdsa.PrivateKey
}

// testSigner implements crypto.Signer for testing
type testSigner struct {
	key *ecdsa.PrivateKey
}

// newTestTEE creates a new test TEE implementation with an ECC P-384 key
func newTestTEE() (*testTEE, error) {
	// Create ECC P-384 key in memory
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &testTEE{key: key}, nil
}

// CreateKey implements TEE.CreateKey
func (t *testTEE) CreateKey(_ context.Context) (tee.Key, error) {
	return &testKey{key: t.key}, nil
}

// LoadKey implements TEE.LoadKey
func (t *testTEE) LoadKey(_ context.Context) (tee.Key, error) {
	return &testKey{key: t.key}, nil
}

// Close implements TEE.Close
func (t *testTEE) Close() error {
	return nil
}

// Signer implements Key.Signer
func (k *testKey) Signer() (crypto.Signer, error) {
	return &testSigner{key: k.key}, nil
}

// Public implements Key.Public
func (k *testKey) Public() (crypto.PublicKey, error) {
	return &k.key.PublicKey, nil
}

// Close implements Key.Close
func (k *testKey) Close() error {
	return nil
}

// Public implements crypto.Signer.Public
func (s *testSigner) Public() crypto.PublicKey {
	return &s.key.PublicKey
}

// Sign implements crypto.Signer.Sign
func (s *testSigner) Sign(rand io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return ecdsa.SignASN1(rand, s.key, digest)
}
