package condaccess

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEnrollmentSecret = "test_secret"

func TestConditionalAccessSCEP(t *testing.T) {
	s := SetUpSuiteWithConfig(t, "integrationtest.ConditionalAccessSCEP", func(cfg *config.FleetConfig) {
		cfg.Osquery.EnrollCooldown = 0 // Disable rate limiting for most tests
	})

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"GetCACaps", testGetCACaps},
		{"GetCACert", testGetCACert},
		{"SCEPEnrollment", testSCEPEnrollment},
		{"InvalidChallenge", testInvalidChallenge},
		{"MissingUUID", testMissingUUID},
		{"NonExistentHost", testNonExistentHost},
		{"CertificateRotation", testCertificateRotation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.BaseSuite.DS, []string{
				"conditional_access_scep_serials", "conditional_access_scep_certificates",
			}...)
			c.fn(t, s)
		})
	}
}

func testGetCACaps(t *testing.T, s *Suite) {
	ctx := t.Context()

	// Create enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: testEnrollmentSecret}})
	require.NoError(t, err)

	// Request CA capabilities
	resp, err := http.Get(s.Server.URL + "/api/fleet/conditional_access/scep?operation=GetCACaps")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	caps := string(body)
	// Verify expected capabilities
	assert.Contains(t, caps, "SHA-256")
	assert.Contains(t, caps, "AES")
	assert.Contains(t, caps, "POSTPKIOperation")
	// Verify Renewal is NOT present
	assert.NotContains(t, caps, "Renewal")
}

func testGetCACert(t *testing.T, s *Suite) {
	// Request CA certificate
	resp, err := http.Get(s.Server.URL + "/api/fleet/conditional_access/scep?operation=GetCACert")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotEmpty(t, body)

	// Parse the certificate
	cert, err := x509.ParseCertificate(body)
	require.NoError(t, err)

	// Verify CA certificate attributes
	assert.Equal(t, "Fleet conditional access CA", cert.Subject.CommonName)
	assert.Contains(t, cert.Subject.Organization, "Local certificate authority")
	assert.True(t, cert.IsCA)

	// Verify RSA key
	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	require.True(t, ok, "CA cert should use RSA public key")
	assert.Equal(t, 2048, rsaPubKey.N.BitLen(), "RSA key should be 2048 bits")
}

func testSCEPEnrollment(t *testing.T, s *Suite) {
	ctx := t.Context()

	// Create enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: testEnrollmentSecret}})
	require.NoError(t, err)

	// Create a test host
	host, err := s.DS.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-scep-1"),
		NodeKey:         ptr.String("test-node-key-scep-1"),
		UUID:            "test-uuid-scep-1",
		Hostname:        "test-hostname-scep-1",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Request certificate via SCEP
	cert := requestSCEPCertificate(t, s, host.UUID, testEnrollmentSecret)
	require.NotNil(t, cert)

	// Verify certificate attributes
	assert.NotNil(t, cert.SerialNumber)
	assert.True(t, time.Now().Before(cert.NotAfter))
	assert.True(t, time.Now().After(cert.NotBefore))

	// Verify SAN URI contains the host UUID
	require.Len(t, cert.URIs, 1)
	assert.Equal(t, "urn:device:apple:uuid:"+host.UUID, cert.URIs[0].String())

	// Verify certificate is stored in database and linked to host
	hostID, err := s.DS.GetConditionalAccessCertHostIDBySerialNumber(ctx, uint64(cert.SerialNumber.Int64())) //nolint:gosec,G115
	require.NoError(t, err)
	assert.Equal(t, host.ID, hostID)

	// Verify certificate validity period (398 days, Apple's maximum)
	expectedMaxDuration := 398*24*time.Hour + 24*time.Hour // Allow 1 day tolerance
	expectedMinDuration := 398*24*time.Hour - 24*time.Hour
	actualDuration := cert.NotAfter.Sub(cert.NotBefore)
	assert.True(t, actualDuration >= expectedMinDuration && actualDuration <= expectedMaxDuration,
		"Certificate should be valid for approximately 398 days")
}

func testInvalidChallenge(t *testing.T, s *Suite) {
	ctx := t.Context()

	// Create enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: testEnrollmentSecret}})
	require.NoError(t, err)

	// Create a test host
	host, err := s.DS.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-invalid-1"),
		NodeKey:         ptr.String("test-node-key-invalid-1"),
		UUID:            "test-uuid-invalid-1",
		Hostname:        "test-hostname-invalid-1",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Try to enroll with invalid challenge
	httpResp, pkiMsgResp, cert := requestSCEPCertificateWithChallenge(t, s, host.UUID, "invalid-secret")
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "SCEP returns HTTP 200 even for failures")
	require.Equal(t, scep.FAILURE, pkiMsgResp.PKIStatus, "SCEP request should fail with invalid challenge")
	require.Nil(t, cert, "Certificate should not be issued with invalid challenge")
}

func testMissingUUID(t *testing.T, s *Suite) {
	ctx := t.Context()

	// Create enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: testEnrollmentSecret}})
	require.NoError(t, err)

	// Try to enroll without UUID in SAN URI
	httpResp, pkiMsgResp, cert := requestSCEPCertificateWithoutUUID(t, s, testEnrollmentSecret)
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "SCEP returns HTTP 200 even for failures")
	require.Equal(t, scep.FAILURE, pkiMsgResp.PKIStatus, "SCEP request should fail without UUID")
	require.Nil(t, cert, "Certificate should not be issued without UUID in SAN URI")

	// Verify no certificate was stored
	_, err = s.DS.GetConditionalAccessCertHostIDBySerialNumber(ctx, 1)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testNonExistentHost(t *testing.T, s *Suite) {
	ctx := t.Context()

	// Create enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: testEnrollmentSecret}})
	require.NoError(t, err)

	// Try to enroll with UUID for a host that doesn't exist
	httpResp, pkiMsgResp, cert := requestSCEPCertificateWithChallenge(t, s, "non-existent-uuid", testEnrollmentSecret)
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "SCEP returns HTTP 200 even for failures")
	require.Equal(t, scep.FAILURE, pkiMsgResp.PKIStatus, "SCEP request should fail for non-existent host")
	require.Nil(t, cert, "Certificate should not be issued for non-existent host")
}

// Helper functions

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

func requestSCEPCertificate(t *testing.T, s *Suite, hostUUID string, challenge string) *x509.Certificate {
	httpResp, pkiMsgResp, cert := requestSCEPCertificateWithChallenge(t, s, hostUUID, challenge)
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "SCEP request should succeed")
	require.Equal(t, scep.SUCCESS, pkiMsgResp.PKIStatus, "SCEP request should succeed")
	return cert
}

func requestSCEPCertificateWithChallenge(t *testing.T, s *Suite, hostUUID string, challenge string) (*http.Response, *scep.PKIMessage, *x509.Certificate) {
	deviceURI, err := url.Parse("urn:device:apple:uuid:" + hostUUID)
	require.NoError(t, err)

	return requestSCEPCertificateWithOptions(t, s, []*url.URL{deviceURI}, challenge)
}

func requestSCEPCertificateWithoutUUID(t *testing.T, s *Suite, challenge string) (*http.Response, *scep.PKIMessage, *x509.Certificate) {
	return requestSCEPCertificateWithOptions(t, s, nil, challenge)
}

func requestSCEPCertificateWithOptions(t *testing.T, s *Suite, uris []*url.URL, challenge string) (*http.Response, *scep.PKIMessage, *x509.Certificate) {
	ctx := context.Background()

	// Generate RSA key pair for the device (conditional access uses RSA, not ECC)
	deviceKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create SCEP client
	scepURL := fmt.Sprintf("%s/api/fleet/conditional_access/scep", s.Server.URL)
	scepClient, err := scepclient.New(scepURL, s.Logger)
	require.NoError(t, err)

	// Get CA certificate
	resp, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(resp)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts)

	// Create CSR template with SAN URI
	hostIdentifier := "test-device"
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: hostIdentifier,
			},
			URIs:               uris,
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: challenge,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, deviceKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	tempRSAKey, deviceCert := createTempRSAKeyAndCert(t, hostIdentifier)

	// Create SCEP PKI message
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   tempRSAKey,
		SignerCert:  deviceCert,
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(s.Logger))
	require.NoError(t, err)

	// Send PKI operation request using HTTP client directly to capture response
	httpReq, err := http.NewRequestWithContext(ctx, "POST", scepURL+"?operation=PKIOperation", strings.NewReader(string(msg.Raw)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/x-pki-message")

	httpResp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// For rate limit errors, we expect HTTP 429 and should return immediately
	if httpResp.StatusCode == http.StatusTooManyRequests {
		return httpResp, nil, nil
	}

	// For other errors, fail the test
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Expected HTTP 200 but got %s", httpResp.Status)

	// Read response body
	respBytes, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(s.Logger), scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)

	// Check for SCEP-level failure
	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		return httpResp, pkiMsgResp, nil
	}

	// Decrypt PKI envelope using RSA key
	err = pkiMsgResp.DecryptPKIEnvelope(deviceCert, tempRSAKey)
	require.NoError(t, err)

	// Verify we got a certificate
	require.NotNil(t, pkiMsgResp.CertRepMessage)
	require.NotNil(t, pkiMsgResp.CertRepMessage.Certificate)

	cert := pkiMsgResp.CertRepMessage.Certificate
	return httpResp, pkiMsgResp, cert
}

// testCertificateRotation tests the grace period behavior during certificate rotation.
// This validates that after a host requests a new certificate via SCEP, both the old and new certificates
// continue to work for authentication until the grace period expires (cleaned up by periodic job).
func testCertificateRotation(t *testing.T, s *Suite) {
	ctx := t.Context()

	// Create enrollment secret
	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: testEnrollmentSecret}})
	require.NoError(t, err)

	// Create a test host
	host, err := s.DS.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-host-rotation"),
		NodeKey:         ptr.String("test-node-key-rotation"),
		UUID:            "test-uuid-rotation",
		Hostname:        "test-hostname-rotation",
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Request first certificate via SCEP (old cert)
	oldCert := requestSCEPCertificate(t, s, host.UUID, testEnrollmentSecret)
	require.NotNil(t, oldCert)

	// Make HTTP request to IdP SSO endpoint with old cert serial to verify authentication
	oldSerialHex := fmt.Sprintf("%X", oldCert.SerialNumber)
	req, err := http.NewRequestWithContext(ctx, "POST", s.Server.URL+"/api/fleet/conditional_access/idp/sso", nil)
	require.NoError(t, err)
	req.Header.Set("X-Client-Cert-Serial", oldSerialHex)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	// StatusBadRequest (400) indicates cert authentication succeeded but SAML request is invalid/empty
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "old cert should authenticate (400 = auth success, SAML parse fail)")

	// Request new certificate via SCEP (certificate rotation)
	newCert := requestSCEPCertificate(t, s, host.UUID, testEnrollmentSecret)
	require.NotNil(t, newCert)
	require.NotEqual(t, oldCert.SerialNumber, newCert.SerialNumber, "new cert should have different serial")

	// CRITICAL TEST: Both old and new certs should work via actual HTTP endpoint (grace period behavior)
	// This validates that old cert is NOT immediately revoked, allowing grace period for rotation
	req, err = http.NewRequestWithContext(ctx, "POST", s.Server.URL+"/api/fleet/conditional_access/idp/sso", nil)
	require.NoError(t, err)
	req.Header.Set("X-Client-Cert-Serial", oldSerialHex)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "old cert should still work after new cert issued (grace period)")

	newSerialHex := fmt.Sprintf("%X", newCert.SerialNumber)
	req, err = http.NewRequestWithContext(ctx, "POST", s.Server.URL+"/api/fleet/conditional_access/idp/sso", nil)
	require.NoError(t, err)
	req.Header.Set("X-Client-Cert-Serial", newSerialHex)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "new cert should work via HTTP endpoint")
}
