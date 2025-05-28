package scep

import (
	"context"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestClient_FetchAndSaveCert(t *testing.T) {
	// Start a test SCEP server
	scepServer := StartTestSCEPServer(t)
	defer scepServer.Close()

	// Create a temporary directory for storing certificates
	certDir := t.TempDir()

	// Create a logger for testing
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create a SCEP client
	client, err := NewClient(
		WithURL(scepServer.URL+"/scep"),
		WithChallenge("test-challenge"),
		WithCertDestDir(certDir),
		WithLogger(logger),
		WithTimeout(5*time.Second),
	)
	require.Error(t, err, "NewClient should fail without commonName")

	// Create a SCEP client with all required parameters
	client, err = NewClient(
		WithURL(scepServer.URL+"/scep"),
		WithChallenge("test-challenge"),
		WithCertDestDir(certDir),
		WithLogger(logger),
		WithTimeout(5*time.Second),
		WithCommonName("test-device"),
	)
	require.NoError(t, err, "NewClient should succeed with all required parameters")

	// Fetch and save the certificate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.FetchAndSaveCert(ctx)
	require.NoError(t, err, "FetchAndSaveCert should succeed")

	// Verify that the certificate and key files were created
	certPath := filepath.Join(certDir, constant.FleetTLSClientCertificateFileName)
	keyPath := filepath.Join(certDir, constant.FleetTLSClientKeyFileName)

	// Check if files exist
	_, err = os.Stat(certPath)
	require.NoError(t, err, "Certificate file should exist")
	_, err = os.Stat(keyPath)
	require.NoError(t, err, "Key file should exist")

	// Verify certificate content
	certData, err := os.ReadFile(certPath)
	require.NoError(t, err, "Should be able to read certificate file")
	certBlock, _ := pem.Decode(certData)
	require.NotNil(t, certBlock, "Certificate should be in PEM format")
	require.Equal(t, "CERTIFICATE", certBlock.Type, "Certificate block type should be CERTIFICATE")

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	require.NoError(t, err, "Should be able to parse certificate")
	require.Equal(t, "test-device", cert.Subject.CommonName, "Certificate should have the correct common name")

	// Verify key content
	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err, "Should be able to read key file")
	keyBlock, _ := pem.Decode(keyData)
	require.NotNil(t, keyBlock, "Key should be in PEM format")
	require.Equal(t, "RSA PRIVATE KEY", keyBlock.Type, "Key block type should be RSA PRIVATE KEY")

	// Test with missing parameters
	_, err = NewClient(
		WithURL(""),
		WithChallenge("test-challenge"),
		WithCertDestDir(certDir),
		WithCommonName("test-device"),
	)
	require.Error(t, err, "NewClient should fail with empty URL")

	_, err = NewClient(
		WithURL(scepServer.URL+"/scep"),
		WithChallenge("test-challenge"),
		WithCertDestDir(""),
		WithCommonName("test-device"),
	)
	require.Error(t, err, "NewClient should fail with empty certDestDir")
}

//go:embed testdata/ca.crt
var caCert []byte

//go:embed testdata/ca.key
var caKey []byte

//go:embed testdata/ca.pem
var caPem []byte

func StartTestSCEPServer(t *testing.T) *httptest.Server {

	caDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(caDir, "ca.crt"), caCert, 0644); err != nil {
		t.Fatalf("failed to write ca.crt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.key"), caKey, 0644); err != nil {
		t.Fatalf("failed to write ca.key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.pem"), caPem, 0644); err != nil {
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

		svc, err := scepserver.NewService(crt[0], key, scepserver.SignCSRAdapter(depot.NewSigner(certDepot)))
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
