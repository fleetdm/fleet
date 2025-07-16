package hostidentity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEnrollmentSecret = "test_secret"

func TestHostIdentitySCEP(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.HostIdentitySCEP")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"GetCert", testGetCert},
		{"GetCertFailures", testGetCertFailures},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.BaseSuite.DS, []string{
				"host_identity_scep_serials", "host_identity_scep_certificates",
			}...)
			c.fn(t, s)
		})
	}
}

func testGetCert(t *testing.T, s *Suite) {
	t.Run("ECC P256", func(t *testing.T) {
		testGetCertWithCurve(t, s, elliptic.P256())
	})

	t.Run("ECC P384", func(t *testing.T) {
		testGetCertWithCurve(t, s, elliptic.P384())
	})
}

func testGetCertWithCurve(t *testing.T, s *Suite, curve elliptic.Curve) {
	ctx := t.Context()
	// Create an enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{
		{
			Secret: testEnrollmentSecret,
		},
	})
	require.NoError(t, err)

	// Create ECC private key with specified curve
	eccPrivateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	// Create SCEP client
	scepURL := fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", s.Server.URL)
	scepClient, err := scepclient.New(scepURL, s.Logger, nil, "", false)
	require.NoError(t, err)

	// Get CA certificate
	resp, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(resp)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts)

	// Create CSR using ECC key
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test-host-identity",
			},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
		},
		ChallengePassword: testEnrollmentSecret,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, eccPrivateKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	tempRSAKey, deviceCert := createTempRSAKeyAndCert(t, "test-host-identity")

	// Create SCEP PKI message
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   tempRSAKey, // Use RSA key for SCEP protocol
		SignerCert:  deviceCert,
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(s.Logger))
	require.NoError(t, err)

	// Send PKI operation request
	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	require.NoError(t, err)

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(s.Logger), scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)

	// Verify successful response
	require.Equal(t, scep.SUCCESS, pkiMsgResp.PKIStatus, "SCEP request should succeed")

	// Decrypt PKI envelope using RSA key
	err = pkiMsgResp.DecryptPKIEnvelope(deviceCert, tempRSAKey)
	require.NoError(t, err)

	// Verify we got a certificate
	require.NotNil(t, pkiMsgResp.CertRepMessage)
	require.NotNil(t, pkiMsgResp.CertRepMessage.Certificate)

	// Verify the certificate was signed by the CA
	cert := pkiMsgResp.CertRepMessage.Certificate
	require.NotNil(t, cert)

	// Verify certificate properties
	assert.Equal(t, "test-host-identity", cert.Subject.CommonName)
	assert.Equal(t, x509.ECDSA, cert.PublicKeyAlgorithm)
	certPubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Certificate should contain ECC public key")
	assert.True(t, eccPrivateKey.PublicKey.Equal(certPubKey), "Certificate public key should match our ECC private key")
	assert.Equal(t, curve, certPubKey.Curve, "Certificate should use the expected elliptic curve")

	// Retrieve the certificate from datastore and verify it matches SCEP response
	storedCert, err := s.DS.GetHostIdentityCertBySerialNumber(ctx, cert.SerialNumber.Uint64())
	require.NoError(t, err)
	require.NotNil(t, storedCert)

	// Verify stored certificate properties match the SCEP response
	assert.Equal(t, cert.SerialNumber.Uint64(), storedCert.SerialNumber)
	assert.Equal(t, cert.Subject.CommonName, storedCert.CommonName)
	assert.Equal(t, cert.NotAfter, storedCert.NotValidAfter)

	// Verify the stored public key matches the certificate public key
	storedPubKey, err := storedCert.UnmarshalPublicKey()
	require.NoError(t, err)
	require.NotNil(t, storedPubKey)
	assert.True(t, certPubKey.Equal(storedPubKey), "Stored public key should match certificate public key")
	assert.Equal(t, curve, storedPubKey.Curve, "Stored public key should use the expected elliptic curve")
}

func createTempRSAKeyAndCert(t *testing.T, commonName string) (*rsa.PrivateKey, *x509.Certificate) {
	// Create temporary RSA key for SCEP envelope (required by SCEP protocol)
	tempRSAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create self-signed certificate for SCEP protocol using RSA key
	deviceCertTemplate := x509.Certificate{
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	deviceCertDerBytes, err := x509.CreateCertificate(
		rand.Reader,
		&deviceCertTemplate,
		&deviceCertTemplate,
		&tempRSAKey.PublicKey,
		tempRSAKey,
	)
	require.NoError(t, err)

	deviceCert, err := x509.ParseCertificate(deviceCertDerBytes)
	require.NoError(t, err)
	return tempRSAKey, deviceCert
}

func testGetCertFailures(t *testing.T, s *Suite) {
	cases := []struct {
		name   string
		config SCEPFailureConfig
	}{
		{
			name: "empty challenge password",
			config: SCEPFailureConfig{
				ChallengePassword: "",
				CommonName:        "test-host-identity",
				UseECC:            true,
			},
		},
		{
			name: "wrong challenge password",
			config: SCEPFailureConfig{
				ChallengePassword: "wrong-secret",
				CommonName:        "test-host-identity",
				UseECC:            true,
			},
		},
		{
			name: "CN longer than 255 characters",
			config: SCEPFailureConfig{
				ChallengePassword: testEnrollmentSecret,
				CommonName:        strings.Repeat("a", 256),
				UseECC:            true,
			},
		},
		{
			name: "non-ECC algorithm used",
			config: SCEPFailureConfig{
				ChallengePassword: testEnrollmentSecret,
				CommonName:        "test-host-identity",
				UseECC:            false,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testSCEPFailure(t, s, c.config)
		})
	}
}

type SCEPFailureConfig struct {
	ChallengePassword string
	CommonName        string
	UseECC            bool
}

func testSCEPFailure(t *testing.T, s *Suite, config SCEPFailureConfig) {
	ctx := t.Context()

	// Create an enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{
		{
			Secret: testEnrollmentSecret,
		},
	})
	require.NoError(t, err)

	// Create SCEP client
	scepURL := fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", s.Server.URL)
	scepClient, err := scepclient.New(scepURL, s.Logger, nil, "", false)
	require.NoError(t, err)

	// Get CA certificate
	resp, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(resp)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts)

	var privateKey interface{}
	var sigAlg x509.SignatureAlgorithm

	if config.UseECC {
		// Create ECC private key
		eccKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		privateKey = eccKey
		sigAlg = x509.ECDSAWithSHA256
	} else {
		// Create RSA private key to test non-ECC algorithm rejection (should fail)
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		privateKey = rsaKey
		sigAlg = x509.SHA256WithRSA
	}

	// Create CSR
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: config.CommonName,
			},
			SignatureAlgorithm: sigAlg,
		},
		ChallengePassword: config.ChallengePassword,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	tempRSAKey, deviceCert := createTempRSAKeyAndCert(t, config.CommonName)

	// Create SCEP PKI message
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   tempRSAKey,
		SignerCert:  deviceCert,
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(s.Logger))
	require.NoError(t, err)

	// Send PKI operation request
	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	require.NoError(t, err)

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(s.Logger), scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)

	// Verify failure response
	assert.Equal(t, scep.FAILURE, pkiMsgResp.PKIStatus, "SCEP request should fail")
}
