package hostidentity

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/remitly-oss/httpsig-go"
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
		cert, eccPrivateKey := testGetCertWithCurve(t, s, elliptic.P256())
		testOrbitEnrollment(t, s, cert, eccPrivateKey)
	})

	t.Run("ECC P384", func(t *testing.T) {
		cert, eccPrivateKey := testGetCertWithCurve(t, s, elliptic.P384())
		testOrbitEnrollment(t, s, cert, eccPrivateKey)
	})
}

func testGetCertWithCurve(t *testing.T, s *Suite, curve elliptic.Curve) (cert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey) {
	ctx := t.Context()
	// Create an enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{
		{
			Secret: testEnrollmentSecret,
		},
	})
	require.NoError(t, err)

	// Create ECC private key with specified curve
	eccPrivateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	// Create SCEP client
	scepURL := fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", s.Server.URL)
	scepClient, err := scepclient.New(scepURL, s.Logger, nil)
	require.NoError(t, err)

	// Get CA certificate
	resp, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(resp)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts)

	// Create CSR using ECC key
	hostIdentifier := "test-host-identity"
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: hostIdentifier,
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
	cert = pkiMsgResp.CertRepMessage.Certificate
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

	return cert, eccPrivateKey
}

func testOrbitEnrollment(t *testing.T, s *Suite, cert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey) {
	// Test orbit enrollment with the certificate
	type EnrollOrbitResponse struct {
		OrbitNodeKey string `json:"orbit_node_key,omitempty"`
		Err          error  `json:"error,omitempty"`
	}

	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + cert.Subject.CommonName,
		HardwareSerial:    "test-serial-" + cert.Subject.CommonName,
		Hostname:          "test-hostname-" + cert.Subject.CommonName,
		OsqueryIdentifier: cert.Subject.CommonName,
	}

	// This request is sent without an HTTP signature, so it should fail.
	var enrollResp EnrollOrbitResponse
	s.DoJSON(t, "POST", "/api/fleet/orbit/enroll", enrollRequest, http.StatusUnauthorized, &enrollResp)

	// Now send the same request with an HTTP signature
	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Determine the algorithm based on the curve
	var algo httpsig.Algorithm
	switch eccPrivateKey.Curve {
	case elliptic.P256():
		algo = httpsig.Algo_ECDSA_P256_SHA256
	case elliptic.P384():
		algo = httpsig.Algo_ECDSA_P384_SHA384
	default:
		t.Fatalf("Unsupported curve: %v", eccPrivateKey.Curve)
	}

	// Create signer
	signer, err := httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: algo,
		// We are not using @target-uri in the signature so that we don't run into issues with HTTPS forwarding and proxies (http vs https).
		Fields:   httpsig.Fields("@method", "@authority", "@path", "@query", "content-digest"),
		Metadata: []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
	}, httpsig.SigningKey{
		Key:       eccPrivateKey,
		MetaKeyID: fmt.Sprintf("%d", cert.SerialNumber.Uint64()),
	})
	require.NoError(t, err)

	// Sign the request
	err = signer.Sign(req)
	require.NoError(t, err)

	// Send the signed request
	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// The request with a valid HTTP signature should succeed
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Request with HTTP signature should succeed")

	// Parse the response
	var signedEnrollResp EnrollOrbitResponse
	err = json.NewDecoder(httpResp.Body).Decode(&signedEnrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, signedEnrollResp.OrbitNodeKey, "Should receive orbit node key")
	require.NoError(t, signedEnrollResp.Err)

	// Send the same request again. We don't have replay protection, so it should succeed.
	httpResp, err = client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Same request with HTTP signature should succeed")
	// Parse the response
	signedEnrollResp = EnrollOrbitResponse{}
	err = json.NewDecoder(httpResp.Body).Decode(&signedEnrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, signedEnrollResp.OrbitNodeKey, "Should receive orbit node key")
	require.NoError(t, signedEnrollResp.Err)

	// Test /api/fleet/orbit/config endpoint with different signature scenarios
	t.Run("config endpoint signature tests", func(t *testing.T) {
		type configRequest struct {
			OrbitNodeKey string `json:"orbit_node_key"`
		}

		testCases := []struct {
			name           string
			setupRequest   func() (*http.Request, error)
			expectedStatus int
		}{
			{
				name: "without signature",
				setupRequest: func() (*http.Request, error) {
					configReq := configRequest{OrbitNodeKey: signedEnrollResp.OrbitNodeKey}
					reqBody, err := json.Marshal(configReq)
					if err != nil {
						return nil, err
					}
					req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
					if err != nil {
						return nil, err
					}
					req.Header.Set("Content-Type", "application/json")
					return req, nil
				},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name: "with valid signature",
				setupRequest: func() (*http.Request, error) {
					configReq := configRequest{OrbitNodeKey: signedEnrollResp.OrbitNodeKey}
					reqBody, err := json.Marshal(configReq)
					if err != nil {
						return nil, err
					}
					req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
					if err != nil {
						return nil, err
					}
					req.Header.Set("Content-Type", "application/json")

					err = signer.Sign(req)
					if err != nil {
						return nil, err
					}
					return req, nil
				},
				expectedStatus: http.StatusOK,
			},
			{
				name: "with corrupted signature",
				setupRequest: func() (*http.Request, error) {
					configReq := configRequest{OrbitNodeKey: signedEnrollResp.OrbitNodeKey}
					reqBody, err := json.Marshal(configReq)
					if err != nil {
						return nil, err
					}
					req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
					if err != nil {
						return nil, err
					}
					req.Header.Set("Content-Type", "application/json")

					// Sign with the correct signer first
					err = signer.Sign(req)
					if err != nil {
						return nil, err
					}

					// Then corrupt the signature by modifying the signature header
					sigHeader := req.Header.Get("Signature")
					if sigHeader != "" {
						// Corrupt the signature by changing the last character
						corrupted := sigHeader[:len(sigHeader)-1] + "X"
						req.Header.Set("Signature", corrupted)
					}
					return req, nil
				},
				expectedStatus: http.StatusUnauthorized,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := tc.setupRequest()
				require.NoError(t, err)

				httpResp, err := client.Do(req)
				require.NoError(t, err)
				defer httpResp.Body.Close()

				require.Equal(t, tc.expectedStatus, httpResp.StatusCode)
			})
		}
	})
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
	scepClient, err := scepclient.New(scepURL, s.Logger, nil)
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
