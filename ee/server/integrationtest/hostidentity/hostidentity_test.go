//go:build !windows

// Windows is disabled because the TPM simulator requires CGO, which causes lint failures on Windows.

package hostidentity

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	mathrand "math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/hostidentity"
	orbitscep "github.com/fleetdm/fleet/v4/ee/orbit/pkg/scep"
	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/securehw"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/fleethttpsig"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/google/go-tpm/tpm2/transport/simulator"
	"github.com/remitly-oss/httpsig-go"
	"github.com/rs/zerolog"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEnrollmentSecret = "test_secret"

func TestHostIdentity(t *testing.T) {
	s := SetUpSuiteWithConfig(t, "integrationtest.HostIdentity", false, func(cfg *config.FleetConfig) {
		cfg.Osquery.EnrollCooldown = 0 // Disable rate limiting for tests
	})

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"GetCertAndSignReq", testGetCertAndSignReq},
		{"GetCertFailures", testGetCertFailures},
		{"WrongCertAuthentication", testWrongCertAuthentication},
		{"RealSecureHWAndSCEP", testRealSecureHWAndSCEP},
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

func testGetCertAndSignReq(t *testing.T, s *Suite) {
	t.Run("ECC P256, orbit", func(t *testing.T) {
		t.Parallel()
		cert, eccPrivateKey := testGetCertWithCurve(t, s, elliptic.P256())
		nodeKey := testOrbitEnrollment(t, s, cert, eccPrivateKey)
		testCertificateRenewal(t, s, cert, eccPrivateKey, nodeKey, false) // false = orbit
		testDeleteHostAndReenroll(t, s, cert, eccPrivateKey, nodeKey)
	})

	t.Run("ECC P384, orbit", func(t *testing.T) {
		t.Parallel()
		cert, eccPrivateKey := testGetCertWithCurve(t, s, elliptic.P384())
		nodeKey := testOrbitEnrollment(t, s, cert, eccPrivateKey)
		testCertificateRenewal(t, s, cert, eccPrivateKey, nodeKey, false) // false = orbit
		testDeleteHostAndReenroll(t, s, cert, eccPrivateKey, nodeKey)
	})

	t.Run("ECC P384, osquery", func(t *testing.T) {
		t.Parallel()
		cert, eccPrivateKey := testGetCertWithCurve(t, s, elliptic.P384())
		nodeKey := testOsqueryEnrollment(t, s, cert, eccPrivateKey)
		testCertificateRenewal(t, s, cert, eccPrivateKey, nodeKey, true) // true = osquery
		testDeleteHostAndReenrollOsquery(t, s, cert, eccPrivateKey, nodeKey)
	})
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[mathrand.IntN(len(charset))] // nolint:gosec // waive G404 since this is test code
	}
	return string(result)
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
	scepClient, err := scepclient.New(scepURL, s.Logger)
	require.NoError(t, err)

	// Get CA certificate
	resp, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(resp)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts)

	// Create CSR using ECC key
	hostIdentifier := generateRandomString(16)
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

	tempRSAKey, deviceCert := createTempRSAKeyAndCert(t, hostIdentifier)

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
	assert.Equal(t, hostIdentifier, cert.Subject.CommonName)
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

// createHTTPSigner creates an HTTP signature signer for the given ECC private key and certificate
func createHTTPSigner(t *testing.T, eccPrivateKey *ecdsa.PrivateKey, cert *x509.Certificate) *httpsig.Signer {
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
	signer, err := fleethttpsig.Signer(
		fmt.Sprintf("%d", cert.SerialNumber.Uint64()),
		eccPrivateKey,
		algo,
	)
	require.NoError(t, err)
	return signer
}

func testOrbitEnrollment(t *testing.T, s *Suite, cert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey) string {
	ctx := t.Context()
	// Test orbit enrollment with the certificate
	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + cert.Subject.CommonName,
		HardwareSerial:    "test-serial-" + cert.Subject.CommonName,
		Hostname:          "test-hostname-" + cert.Subject.CommonName,
		OsqueryIdentifier: cert.Subject.CommonName,
	}

	// This request is sent without an HTTP signature, so it should fail.
	var enrollResp enrollOrbitResponse
	s.DoJSON(t, "POST", "/api/fleet/orbit/enroll", enrollRequest, http.StatusUnauthorized, &enrollResp)

	// Now send the same request with an HTTP signature
	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Create signer using the shared helper
	signer := createHTTPSigner(t, eccPrivateKey, cert)

	// Sign the request
	err = signer.Sign(req)
	require.NoError(t, err)

	clonedRequest := req.Clone(ctx)

	// Send the signed request
	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// The request with a valid HTTP signature should succeed
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Request with HTTP signature should succeed")

	// Parse the response
	var signedEnrollResp enrollOrbitResponse
	err = json.NewDecoder(httpResp.Body).Decode(&signedEnrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, signedEnrollResp.OrbitNodeKey, "Should receive orbit node key")
	require.NoError(t, signedEnrollResp.Err)

	// Send the same request again. We don't have replay protection, so it should succeed.
	httpResp, err = client.Do(clonedRequest)
	require.NoError(t, err)
	defer httpResp.Body.Close()
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Same request with HTTP signature should succeed")

	// Parse the response
	signedEnrollResp = enrollOrbitResponse{}
	err = json.NewDecoder(httpResp.Body).Decode(&signedEnrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, signedEnrollResp.OrbitNodeKey, "Should receive orbit node key")
	require.NoError(t, signedEnrollResp.Err)

	// Test /api/fleet/orbit/config endpoint with different signature scenarios
	t.Run("config endpoint signature tests", func(t *testing.T) {
		testCases := []struct {
			name           string
			setupRequest   func() (*http.Request, error)
			expectedStatus int
		}{
			{
				name: "without signature",
				setupRequest: func() (*http.Request, error) {
					configReq := orbitConfigRequest{OrbitNodeKey: signedEnrollResp.OrbitNodeKey}
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
					configReq := orbitConfigRequest{OrbitNodeKey: signedEnrollResp.OrbitNodeKey}
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
					configReq := orbitConfigRequest{OrbitNodeKey: signedEnrollResp.OrbitNodeKey}
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

	return signedEnrollResp.OrbitNodeKey
}

func testOsqueryEnrollment(t *testing.T, s *Suite, cert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey) string {
	// Test osquery enrollment with the certificate
	enrollRequest := contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   testEnrollmentSecret,
		HostIdentifier: cert.Subject.CommonName,
		HostDetails: map[string]map[string]string{
			"osquery_info": {
				"version": "5.0.0",
			},
		},
	}

	// This request is sent without an HTTP signature, so it should fail.
	var enrollResp contract.EnrollOsqueryAgentResponse
	s.DoJSON(t, "POST", "/api/v1/osquery/enroll", enrollRequest, http.StatusUnauthorized, &enrollResp)

	// Now send the same request with HTTP message signature
	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/osquery/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Create signer using the shared helper
	signer := createHTTPSigner(t, eccPrivateKey, cert)

	// Sign the request
	err = signer.Sign(req)
	require.NoError(t, err)

	// Send the signed request
	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// The request with a valid HTTP signature should succeed
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Osquery enrollment with HTTP signature should succeed")

	// Parse the response
	enrollResp = contract.EnrollOsqueryAgentResponse{}
	err = json.NewDecoder(httpResp.Body).Decode(&enrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, enrollResp.NodeKey, "Should receive node key")
	require.NoError(t, enrollResp.Err)

	// Test /api/osquery/config endpoint with different signature scenarios
	t.Run("osquery config endpoint signature tests", func(t *testing.T) {
		testCases := []struct {
			name           string
			setupRequest   func() (*http.Request, error)
			expectedStatus int
		}{
			{
				name: "without signature",
				setupRequest: func() (*http.Request, error) {
					configReq := osqueryConfigRequest{NodeKey: enrollResp.NodeKey}
					reqBody, err := json.Marshal(configReq)
					if err != nil {
						return nil, err
					}
					req, err := http.NewRequest("POST", s.Server.URL+"/api/osquery/config", bytes.NewReader(reqBody))
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
					configReq := osqueryConfigRequest{NodeKey: enrollResp.NodeKey}
					reqBody, err := json.Marshal(configReq)
					if err != nil {
						return nil, err
					}
					req, err := http.NewRequest("POST", s.Server.URL+"/api/osquery/config", bytes.NewReader(reqBody))
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
					configReq := osqueryConfigRequest{NodeKey: enrollResp.NodeKey}
					reqBody, err := json.Marshal(configReq)
					if err != nil {
						return nil, err
					}
					req, err := http.NewRequest("POST", s.Server.URL+"/api/osquery/config", bytes.NewReader(reqBody))
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

	return enrollResp.NodeKey
}

// testCertificateRenewal tests the SCEP certificate renewal flow with proof-of-possession
func testCertificateRenewal(t *testing.T, s *Suite, existingCert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey, nodeKey string, isOsquery bool) {
	ctx := t.Context()

	// Get the original certificate's host_id before renewal (it will get revoked)
	originalStoredCert, err := s.DS.GetHostIdentityCertBySerialNumber(ctx, existingCert.SerialNumber.Uint64())
	require.NoError(t, err)
	require.NotNil(t, originalStoredCert)
	require.NotNil(t, originalStoredCert.HostID, "Original certificate should have host_id")
	originalHostID := *originalStoredCert.HostID

	// Generate a new ECC key pair for the renewed certificate
	newEccPrivateKey, err := ecdsa.GenerateKey(eccPrivateKey.Curve, rand.Reader)
	require.NoError(t, err)

	// Create the renewal data
	serialHex := fmt.Sprintf("0x%x", existingCert.SerialNumber.Bytes())

	// Sign the message with the existing private key
	hash := sha256.Sum256([]byte(serialHex))
	signature, err := ecdsa.SignASN1(rand.Reader, eccPrivateKey, hash[:])
	require.NoError(t, err)

	renewalData := types.RenewalData{
		SerialNumber: serialHex,
		Signature:    base64.StdEncoding.EncodeToString(signature),
	}

	renewalDataJSON, err := json.Marshal(renewalData)
	require.NoError(t, err)

	// Create CSR with renewal extension
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: existingCert.Subject.CommonName,
			},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
			ExtraExtensions: []pkix.Extension{
				{
					Id:    types.RenewalExtensionOID,
					Value: renewalDataJSON,
				},
			},
		},
		// No challenge password for renewal
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, newEccPrivateKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	// Create SCEP client
	scepURL := fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", s.Server.URL)
	scepClient, err := scepclient.New(scepURL, s.Logger)
	require.NoError(t, err)

	// Get CA certificate
	resp, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(resp)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts)

	// Create temporary RSA key for SCEP envelope
	tempRSAKey, tempRSACert := createTempRSAKeyAndCert(t, existingCert.Subject.CommonName)

	// Create SCEP PKI message for renewal
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   tempRSAKey,
		SignerCert:  tempRSACert,
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(s.Logger))
	require.NoError(t, err)

	// Send PKI operation request
	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	require.NoError(t, err)

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(s.Logger), scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)

	// The renewal should succeed
	require.Equal(t, scep.SUCCESS, pkiMsgResp.PKIStatus, "Renewal should succeed")

	// Decrypt PKI envelope using RSA key
	err = pkiMsgResp.DecryptPKIEnvelope(tempRSACert, tempRSAKey)
	require.NoError(t, err)

	// Verify we got a new certificate
	require.NotNil(t, pkiMsgResp.CertRepMessage)
	require.NotNil(t, pkiMsgResp.CertRepMessage.Certificate)

	renewedCert := pkiMsgResp.CertRepMessage.Certificate
	require.NotNil(t, renewedCert)

	// Verify renewed certificate properties
	assert.Equal(t, existingCert.Subject.CommonName, renewedCert.Subject.CommonName, "Common name should be preserved")
	assert.Equal(t, x509.ECDSA, renewedCert.PublicKeyAlgorithm)

	// Verify the renewed certificate has the new public key
	renewedPubKey, ok := renewedCert.PublicKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Renewed certificate should contain ECC public key")
	assert.True(t, newEccPrivateKey.PublicKey.Equal(renewedPubKey), "Renewed certificate should have the new public key")

	// Verify the renewed certificate has a different serial number
	assert.NotEqual(t, existingCert.SerialNumber, renewedCert.SerialNumber, "Renewed certificate should have a new serial number")

	// Verify the renewed certificate maintains the host_id association
	renewedStoredCert, err := s.DS.GetHostIdentityCertBySerialNumber(ctx, renewedCert.SerialNumber.Uint64())
	require.NoError(t, err)
	require.NotNil(t, renewedStoredCert)
	require.NotNil(t, renewedStoredCert.HostID, "Renewed certificate should maintain host_id association")
	require.Equal(t, originalHostID, *renewedStoredCert.HostID, "Renewed certificate should have the same host_id as the original")

	// Test that we can use the renewed certificate to access the config endpoint
	t.Run("test config endpoint with renewed certificate", func(t *testing.T) {
		var configReq interface{}
		var configURL string

		if isOsquery {
			configReq = osqueryConfigRequest{NodeKey: nodeKey}
			configURL = s.Server.URL + "/api/osquery/config"
		} else {
			configReq = orbitConfigRequest{OrbitNodeKey: nodeKey}
			configURL = s.Server.URL + "/api/fleet/orbit/config"
		}

		configReqBody, err := json.Marshal(configReq)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", configURL, bytes.NewReader(configReqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create signer with the renewed certificate and new private key
		signer := createHTTPSigner(t, newEccPrivateKey, renewedCert)
		err = signer.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		// Should succeed with the renewed certificate
		require.Equal(t, http.StatusOK, httpResp.StatusCode, "Config request with renewed certificate should succeed")
	})

	// Test that config endpoint does not work with old certificate after renewal
	t.Run("config endpoint fails with old certificate after renewal", func(t *testing.T) {
		var configReq interface{}
		var configURL string

		if isOsquery {
			configReq = osqueryConfigRequest{NodeKey: nodeKey}
			configURL = s.Server.URL + "/api/osquery/config"
		} else {
			configReq = orbitConfigRequest{OrbitNodeKey: nodeKey}
			configURL = s.Server.URL + "/api/fleet/orbit/config"
		}

		configReqBody, err := json.Marshal(configReq)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", configURL, bytes.NewReader(configReqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create signer with the OLD certificate and OLD private key
		signer := createHTTPSigner(t, eccPrivateKey, existingCert)
		err = signer.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		// Should fail because the old certificate has been revoked
		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Config request with old certificate should fail after renewal")
	})

	// Test that renewal cannot be retried with the same serial number
	t.Run("renewal fails when retrying with same serial", func(t *testing.T) {
		// Try to renew again using the same old certificate serial number
		// This should fail because the certificate has already been revoked

		// Generate another new key pair for this attempt
		anotherNewKey, err := ecdsa.GenerateKey(eccPrivateKey.Curve, rand.Reader)
		require.NoError(t, err)

		// Use the same renewal data as before (same serial and signature)
		retryCSRTemplate := x509util.CertificateRequest{
			CertificateRequest: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName: existingCert.Subject.CommonName,
				},
				SignatureAlgorithm: x509.ECDSAWithSHA256,
				ExtraExtensions: []pkix.Extension{
					{
						Id:    types.RenewalExtensionOID,
						Value: renewalDataJSON, // Reuse the same renewal data
					},
				},
			},
		}

		retryCSRDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &retryCSRTemplate, anotherNewKey)
		require.NoError(t, err)
		retryCSR, err := x509.ParseCertificateRequest(retryCSRDerBytes)
		require.NoError(t, err)

		// Create new temp RSA key for SCEP envelope
		retryTempRSAKey, retryTempRSACert := createTempRSAKeyAndCert(t, existingCert.Subject.CommonName)

		// Create SCEP PKI message for retry
		retryPkiMsgReq := &scep.PKIMessage{
			MessageType: scep.PKCSReq,
			Recipients:  caCerts,
			SignerKey:   retryTempRSAKey,
			SignerCert:  retryTempRSACert,
		}

		retryMsg, err := scep.NewCSRRequest(retryCSR, retryPkiMsgReq, scep.WithLogger(s.Logger))
		require.NoError(t, err)

		// Send PKI operation request
		retryRespBytes, err := scepClient.PKIOperation(ctx, retryMsg.Raw)
		require.NoError(t, err)

		// Parse response
		retryPkiMsgResp, err := scep.ParsePKIMessage(retryRespBytes, scep.WithLogger(s.Logger), scep.WithCACerts(retryMsg.Recipients))
		require.NoError(t, err)

		// Should fail - the certificate has already been revoked
		require.Equal(t, scep.FAILURE, retryPkiMsgResp.PKIStatus, "Renewal retry with same serial should fail")
	})
}

func testDeleteHostAndReenroll(t *testing.T, s *Suite, cert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey, nodeKey string) {
	ctx := t.Context()

	// Get the host using the orbit node key
	hostToDelete, err := s.DS.LoadHostByOrbitNodeKey(ctx, nodeKey)
	require.NoError(t, err)
	require.NotNil(t, hostToDelete, "Should find the enrolled host")

	// Delete the host using the API endpoint
	s.Do(t, "DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostToDelete.ID), nil, http.StatusOK)

	// Try to enroll the same host with the same certificate - this should fail
	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + cert.Subject.CommonName,
		HardwareSerial:    "test-serial-" + cert.Subject.CommonName,
		Hostname:          "test-hostname-" + cert.Subject.CommonName,
		OsqueryIdentifier: cert.Subject.CommonName,
	}

	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	signer := createHTTPSigner(t, eccPrivateKey, cert)
	err = signer.Sign(req)
	require.NoError(t, err)

	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// This should fail because the host certificate should be deleted when the host is deleted
	require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Enrollment with deleted host certificate should fail")
}

func testDeleteHostAndReenrollOsquery(t *testing.T, s *Suite, cert *x509.Certificate, eccPrivateKey *ecdsa.PrivateKey, nodeKey string) {
	ctx := t.Context()

	// Get the host using the osquery node key
	hostToDelete, err := s.DS.LoadHostByNodeKey(ctx, nodeKey)
	require.NoError(t, err)
	require.NotNil(t, hostToDelete, "Should find the enrolled host")

	// Delete the host using the API endpoint
	s.Do(t, "DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostToDelete.ID), nil, http.StatusOK)

	// Try to enroll the same host with the same certificate - this should fail
	enrollRequest := contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   testEnrollmentSecret,
		HostIdentifier: cert.Subject.CommonName,
		HostDetails: map[string]map[string]string{
			"osquery_info": {
				"version": "5.0.0",
			},
		},
	}

	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/osquery/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	signer := createHTTPSigner(t, eccPrivateKey, cert)
	err = signer.Sign(req)
	require.NoError(t, err)

	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// This should fail because the host certificate should be deleted when the host is deleted
	require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Enrollment with deleted host certificate should fail")
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
	scepClient, err := scepclient.New(scepURL, s.Logger)
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

func testWrongCertAuthentication(t *testing.T, s *Suite) {
	// Test that hosts cannot use another host's certificate for authentication

	// Create two P384 certificates for different hosts
	certHost1, eccPrivateKeyHost1 := testGetCertWithCurve(t, s, elliptic.P384())
	certHost2, eccPrivateKeyHost2 := testGetCertWithCurve(t, s, elliptic.P384())

	// Create signers for both hosts
	signerHost1 := createHTTPSigner(t, eccPrivateKeyHost1, certHost1)
	signerHost2 := createHTTPSigner(t, eccPrivateKeyHost2, certHost2)

	// Generate a local ECC P384 private key (not from Fleet SCEP)
	localPrivateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)

	// Create a signer using the local private key with a fake certificate serial
	localSigner, err := fleethttpsig.Signer(
		"999999", // Fake certificate serial number
		localPrivateKey,
		httpsig.Algo_ECDSA_P384_SHA384,
	)
	require.NoError(t, err)

	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + certHost1.Subject.CommonName,
		HardwareSerial:    "test-serial-" + certHost1.Subject.CommonName,
		Hostname:          "test-hostname-" + certHost1.Subject.CommonName,
		OsqueryIdentifier: certHost1.Subject.CommonName,
	}

	// Test enrollment with wrong certificate
	enrollHostWithOtherHostCertShouldFail := func(t *testing.T) {
		reqBody, err := json.Marshal(enrollRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Sign with host2's signer (wrong cert)
		err = signerHost2.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		// Should fail because the certificate doesn't match the host identifier
		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Enrollment with wrong certificate should fail")
	}
	t.Run("enroll host1 with host2 cert should fail", enrollHostWithOtherHostCertShouldFail)

	// Test enrollment with local private key
	enrollHostWithLocalPrivateKeyShouldFail := func(t *testing.T) {
		reqBody, err := json.Marshal(enrollRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Sign with local private key (not managed by Fleet)
		err = localSigner.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		// Should fail because the certificate is not managed by Fleet
		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Enrollment with local private key should fail")
	}
	t.Run("enroll host1 with local private key should fail", enrollHostWithLocalPrivateKeyShouldFail)

	// Successfully enroll host1 with correct certificate
	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Sign with host1's signer (correct cert)
	err = signerHost1.Sign(req)
	require.NoError(t, err)

	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Enrollment with correct certificate should succeed")

	var enrollResp enrollOrbitResponse
	err = json.NewDecoder(httpResp.Body).Decode(&enrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, enrollResp.OrbitNodeKey)
	nodeKeyHost1 := enrollResp.OrbitNodeKey

	type orbitConfigRequest struct {
		OrbitNodeKey string `json:"orbit_node_key"`
	}

	t.Run("re-enroll host1 with host2 cert should fail", enrollHostWithOtherHostCertShouldFail)
	t.Run("re-enroll host1 with local private key should fail", enrollHostWithLocalPrivateKeyShouldFail)

	// Try to use host1's endpoint with host2's certificate
	t.Run("host1 config with host2 cert should fail", func(t *testing.T) {
		configRequest := orbitConfigRequest{
			OrbitNodeKey: nodeKeyHost1,
		}

		reqBody, err := json.Marshal(configRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Sign with host2's signer (wrong cert)
		err = signerHost2.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Config request with wrong certificate should fail")
	})

	// Successfully enroll host2 with correct certificate
	enrollRequest2 := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + certHost2.Subject.CommonName,
		HardwareSerial:    "test-serial-" + certHost2.Subject.CommonName,
		Hostname:          "test-hostname-" + certHost2.Subject.CommonName,
		OsqueryIdentifier: certHost2.Subject.CommonName,
	}

	reqBody, err = json.Marshal(enrollRequest2)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Sign with host2's signer (correct cert)
	err = signerHost2.Sign(req)
	require.NoError(t, err)

	httpResp, err = client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Enrollment with correct certificate should succeed")

	enrollResp = enrollOrbitResponse{}
	err = json.NewDecoder(httpResp.Body).Decode(&enrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, enrollResp.OrbitNodeKey)
	nodeKeyHost2 := enrollResp.OrbitNodeKey

	t.Run("re-enroll host1 with host2-enrolled cert should still fail", enrollHostWithOtherHostCertShouldFail)

	// Try to use host2's endpoint with host1's certificate
	t.Run("host2 config with host1 cert should fail", func(t *testing.T) {
		configRequest := orbitConfigRequest{
			OrbitNodeKey: nodeKeyHost2,
		}

		reqBody, err := json.Marshal(configRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Sign with host1's signer (wrong cert)
		err = signerHost1.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Config request with wrong certificate should fail")
	})

	// Test config request with local private key
	t.Run("config request with local private key should fail", func(t *testing.T) {
		configRequest := orbitConfigRequest{
			OrbitNodeKey: nodeKeyHost1,
		}

		reqBody, err := json.Marshal(configRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Sign with local private key (not managed by Fleet)
		err = localSigner.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		// Should fail because the certificate is not managed by Fleet
		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Config request with local private key should fail")
	})

	// Test enrollment failures after host is enrolled - use different host identifiers to avoid re-enrollment
	t.Run("enroll new host with host1 cert should fail after enrollment", func(t *testing.T) {
		newHostEnrollRequest := contract.EnrollOrbitRequest{
			EnrollSecret:      testEnrollmentSecret,
			HardwareUUID:      "test-uuid-new-host-wrong-cert",
			HardwareSerial:    "test-serial-new-host-wrong-cert",
			Hostname:          "test-hostname-new-host-wrong-cert",
			OsqueryIdentifier: "new-host-wrong-cert",
		}

		reqBody, err := json.Marshal(newHostEnrollRequest)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Sign with host1's signer (wrong cert for this new host)
		err = signerHost1.Sign(req)
		require.NoError(t, err)

		client := fleethttp.NewClient()
		httpResp, err := client.Do(req)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		// Should fail because the certificate doesn't match the host identifier
		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Enrollment with wrong certificate should fail even after other hosts are enrolled")
	})
}

// testRealSecureHWAndSCEP uses the SecureHW and SCEP packages that are used by Orbit. Only the TPM device is fake/simulated.
func testRealSecureHWAndSCEP(t *testing.T, s *Suite) {
	t.Parallel()
	ctx := t.Context()

	// Create TPM simulator
	sim, err := simulator.OpenSimulator()
	require.NoError(t, err)

	// Create a temporary directory for metadata
	tempDir := t.TempDir()

	// Create a zerolog logger for the test
	zerologLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create SecureHW instance with TPM simulator
	tpmHW, err := securehw.NewTestSecureHW(sim, tempDir, zerologLogger)
	require.NoError(t, err)

	// Create a new key in the TPM
	tpmKey, err := tpmHW.CreateKey()
	require.NoError(t, err)

	// Set up cleanup - the TPM hardware will be closed once at the end
	t.Cleanup(func() {
		if err := tpmHW.Close(); err != nil {
			// Don't fail if already closed
			t.Logf("TPM close error (may be expected): %v", err)
		}
	})

	// Verify we can get the public key
	pubKey, err := tpmKey.Public()
	require.NoError(t, err)
	eccPubKey, ok := pubKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Expected ECC public key")

	// Create enrollment secret
	err = s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{
		{
			Secret: testEnrollmentSecret,
		},
	})
	require.NoError(t, err)

	// Generate a unique common name
	commonName := generateRandomString(16)

	// Create SCEP client with the TPM key
	scepClient, err := orbitscep.NewClient(
		orbitscep.WithSigningKey(tpmKey),
		orbitscep.WithURL(fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", s.Server.URL)),
		orbitscep.WithCommonName(commonName),
		orbitscep.WithChallenge(testEnrollmentSecret),
		orbitscep.WithLogger(zerologLogger),
	)
	require.NoError(t, err)

	// Fetch certificate using SCEP
	cert, err := scepClient.FetchCert(ctx)
	require.NoError(t, err)
	require.NotNil(t, cert)

	// Verify certificate properties
	assert.Equal(t, commonName, cert.Subject.CommonName)
	assert.Equal(t, x509.ECDSA, cert.PublicKeyAlgorithm)

	// Verify the certificate's public key matches our TPM key
	certPubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Certificate should contain ECC public key")
	assert.True(t, eccPubKey.Equal(certPubKey), "Certificate public key should match TPM key")

	// Test enrollment with HTTP signature using TPM key
	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + commonName,
		HardwareSerial:    "test-serial-" + commonName,
		Hostname:          "test-hostname-" + commonName,
		OsqueryIdentifier: commonName,
	}

	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Get HTTP signer from TPM key
	httpSigner, err := tpmKey.HTTPSigner()
	require.NoError(t, err)

	// Determine algorithm based on the curve
	var algo httpsig.Algorithm
	switch httpSigner.ECCAlgorithm() {
	case securehw.ECCAlgorithmP256:
		algo = httpsig.Algo_ECDSA_P256_SHA256
	case securehw.ECCAlgorithmP384:
		algo = httpsig.Algo_ECDSA_P384_SHA384
	default:
		t.Fatalf("Unsupported ECC algorithm from TPM")
	}

	// Create HTTP signature signer
	signer, err := fleethttpsig.Signer(
		fmt.Sprintf("%d", cert.SerialNumber.Uint64()),
		httpSigner,
		algo,
	)
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
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Request with TPM-based HTTP signature should succeed")

	// Parse the response
	var enrollResp enrollOrbitResponse
	err = json.NewDecoder(httpResp.Body).Decode(&enrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, enrollResp.OrbitNodeKey, "Should receive orbit node key")
	require.NoError(t, enrollResp.Err)

	// Test that we can load the key from storage
	require.NoError(t, tpmKey.Close()) // Close the original key

	loadedKey, err := tpmHW.LoadKey()
	require.NoError(t, err)

	// Verify loaded key has same public key
	loadedPubKey, err := loadedKey.Public()
	require.NoError(t, err)
	loadedECCPubKey, ok := loadedPubKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Loaded key should be ECC")
	assert.True(t, eccPubKey.Equal(loadedECCPubKey), "Loaded key should match original")

	// Test config endpoint with loaded key
	configRequest := orbitConfigRequest{
		OrbitNodeKey: enrollResp.OrbitNodeKey,
	}

	configReqBody, err := json.Marshal(configRequest)
	require.NoError(t, err)

	configReq, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(configReqBody))
	require.NoError(t, err)
	configReq.Header.Set("Content-Type", "application/json")

	// Sign with loaded key
	loadedHTTPSigner, err := loadedKey.HTTPSigner()
	require.NoError(t, err)

	loadedSigner, err := fleethttpsig.Signer(
		fmt.Sprintf("%d", cert.SerialNumber.Uint64()),
		loadedHTTPSigner,
		algo,
	)
	require.NoError(t, err)

	err = loadedSigner.Sign(configReq)
	require.NoError(t, err)

	httpResp, err = client.Do(configReq)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Config request with loaded TPM key should succeed")

	t.Run("renew certificate with real SecureHW and SCEP client", func(t *testing.T) {
		// Get the original certificate's host_id before renewal (it will get revoked)
		originalStoredCert, err := s.DS.GetHostIdentityCertBySerialNumber(ctx, cert.SerialNumber.Uint64())
		require.NoError(t, err)
		require.NotNil(t, originalStoredCert)
		require.NotNil(t, originalStoredCert.HostID, "Original certificate should have host_id")
		originalHostID := *originalStoredCert.HostID
		// Save the current certificate to the expected location
		certPath := filepath.Join(tempDir, constant.FleetHTTPSignatureCertificateFileName)
		certFile, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
		require.NoError(t, err)
		err = pem.Encode(certFile, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		require.NoError(t, err)
		require.NoError(t, certFile.Close())

		// Now we can use hostidentity.RenewCertificate directly since SecureHW is exported
		// Create a Credentials struct with our test TPM
		credentials := &hostidentity.Credentials{
			Certificate:     cert,
			SecureHWKey:     loadedKey,
			CertificatePath: certPath,
			SecureHW:        tpmHW,
		}

		// Use the hostidentity.RenewCertificate method directly
		renewedCert, err := hostidentity.RenewCertificate(
			ctx,
			tempDir,
			credentials,
			fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", s.Server.URL),
			"",   // rootCA - empty for insecure
			true, // insecure
			zerologLogger,
		)
		require.NoError(t, err)
		require.NotNil(t, renewedCert)

		// The RenewCertificate method should have updated credentials.SecureHWKey
		// and saved the renewed certificate

		// Verify renewed certificate properties
		assert.Equal(t, cert.Subject.CommonName, renewedCert.Subject.CommonName, "Common name should be preserved")
		assert.NotEqual(t, cert.SerialNumber, renewedCert.SerialNumber, "Serial number should be different")
		assert.Equal(t, x509.ECDSA, renewedCert.PublicKeyAlgorithm)

		// Verify the renewed certificate has a new public key (from the new TPM key)
		renewedPubKey, ok := renewedCert.PublicKey.(*ecdsa.PublicKey)
		require.True(t, ok, "Renewed certificate should contain ECC public key")
		assert.False(t, certPubKey.Equal(renewedPubKey), "Renewed certificate should have a different public key")

		// Verify the new key's public key matches the renewed certificate
		// The new key is now in credentials.SecureHWKey
		newPubKey, err := credentials.SecureHWKey.Public()
		require.NoError(t, err)
		newECCPubKey, ok := newPubKey.(*ecdsa.PublicKey)
		require.True(t, ok, "New key should be ECC")
		assert.True(t, renewedPubKey.Equal(newECCPubKey), "Renewed certificate public key should match new TPM key")

		// Verify the renewed certificate maintains the host_id association
		renewedStoredCert, err := s.DS.GetHostIdentityCertBySerialNumber(ctx, renewedCert.SerialNumber.Uint64())
		require.NoError(t, err)
		require.NotNil(t, renewedStoredCert)
		require.NotNil(t, renewedStoredCert.HostID, "Renewed certificate should maintain host_id association")
		require.Equal(t, originalHostID, *renewedStoredCert.HostID, "Renewed certificate should have the same host_id as the original")

		// Test that we can use the renewed certificate and new key
		renewedConfigRequest := orbitConfigRequest{
			OrbitNodeKey: enrollResp.OrbitNodeKey,
		}

		renewedConfigReqBody, err := json.Marshal(renewedConfigRequest)
		require.NoError(t, err)

		renewedConfigReq, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(renewedConfigReqBody))
		require.NoError(t, err)
		renewedConfigReq.Header.Set("Content-Type", "application/json")

		// Sign with renewed certificate and new key
		renewedHTTPSigner, err := credentials.SecureHWKey.HTTPSigner()
		require.NoError(t, err)

		// Determine algorithm for renewed key
		var renewedAlgo httpsig.Algorithm
		switch renewedHTTPSigner.ECCAlgorithm() {
		case securehw.ECCAlgorithmP256:
			renewedAlgo = httpsig.Algo_ECDSA_P256_SHA256
		case securehw.ECCAlgorithmP384:
			renewedAlgo = httpsig.Algo_ECDSA_P384_SHA384
		default:
			t.Fatalf("Unsupported ECC algorithm from renewed TPM key")
		}

		renewedSigner, err := fleethttpsig.Signer(
			fmt.Sprintf("%d", renewedCert.SerialNumber.Uint64()),
			renewedHTTPSigner,
			renewedAlgo,
		)
		require.NoError(t, err)

		err = renewedSigner.Sign(renewedConfigReq)
		require.NoError(t, err)

		httpResp, err = client.Do(renewedConfigReq)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		require.Equal(t, http.StatusOK, httpResp.StatusCode, "Config request with renewed certificate should succeed")

		// Test that old certificate no longer works
		// Since the old key was closed and replaced, we need to recreate the signer with the old serial
		oldConfigReq, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(renewedConfigReqBody))
		require.NoError(t, err)
		oldConfigReq.Header.Set("Content-Type", "application/json")

		// Create a signer with the old certificate serial but it should fail since the cert was replaced
		oldSerialSigner, err := fleethttpsig.Signer(
			fmt.Sprintf("%d", cert.SerialNumber.Uint64()),
			renewedHTTPSigner, // Using new key with old serial
			renewedAlgo,
		)
		require.NoError(t, err)

		err = oldSerialSigner.Sign(oldConfigReq)
		require.NoError(t, err)

		httpResp, err = client.Do(oldConfigReq)
		require.NoError(t, err)
		defer httpResp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, httpResp.StatusCode, "Config request with old certificate serial should fail after renewal")

		// Verify the old key backup was cleaned up by RenewCertificate
		oldKeyPath := filepath.Join(tempDir, constant.FleetHTTPSignatureTPMKeyBackupFileName)
		_, err = os.Stat(oldKeyPath)
		require.True(t, os.IsNotExist(err), "Old key backup should have been removed by RenewCertificate")

		// Clean up the new key
		t.Cleanup(func() {
			_ = credentials.SecureHWKey.Close()
		})
	})
}
