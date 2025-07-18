package hostidentity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCEPRateLimit(t *testing.T) {
	// Set up suite with rate limiting configuration
	cooldown := 5 * time.Minute
	s := SetUpSuiteWithConfig(t, "integrationtest.HostIdentitySCEPRateLimit", false, func(cfg *config.FleetConfig) {
		cfg.Osquery.EnrollCooldown = cooldown
	})

	defer mysql.TruncateTables(t, s.BaseSuite.DS, []string{
		"host_identity_scep_serials", "host_identity_scep_certificates",
	}...)

	t.Run("RateLimitSameHost", func(t *testing.T) {
		// Create an enrollment secret
		ctx := t.Context()
		err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{
			{
				Secret: testEnrollmentSecret,
			},
		})
		require.NoError(t, err)

		// Create a unique host identifier (CN)
		hostID := "test-host-rate-limit"

		// First certificate request - should succeed
		_, initialCert := requestSCEPCertificate(t, s, hostID)
		require.NotNil(t, initialCert)
		assert.Equal(t, hostID, initialCert.Subject.CommonName)

		// Second certificate request immediately after - should fail due to rate limit with HTTP 429
		rateLimitedResp, rateLimitedCert := requestSCEPCertificate(t, s, hostID)
		require.Nil(t, rateLimitedCert)
		require.Equal(t, http.StatusTooManyRequests, rateLimitedResp.StatusCode, "Should return HTTP 429 for rate limit")

		// Wait for a small duration (less than cooldown) and try again - should still fail with HTTP 429
		time.Sleep(500 * time.Millisecond)
		stillRateLimitedResp, stillRateLimitedCert := requestSCEPCertificate(t, s, hostID)
		require.Nil(t, stillRateLimitedCert)
		require.Equal(t, http.StatusTooManyRequests, stillRateLimitedResp.StatusCode, "Should still return HTTP 429 for rate limit")

		// Different host should be able to get certificate
		differentHostID := "test-host-different"
		_, differentHostCert := requestSCEPCertificate(t, s, differentHostID)
		require.NotNil(t, differentHostCert)
		assert.Equal(t, differentHostID, differentHostCert.Subject.CommonName)
	})
}

// requestSCEPCertificateWithStatus is a helper function to request a SCEP certificate for a given host identifier
// It returns the HTTP response and the certificate (if successful)
func requestSCEPCertificate(t *testing.T, s *Suite, hostIdentifier string) (*http.Response, *x509.Certificate) {
	ctx := t.Context()

	// Create ECC private key
	eccPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create CSR
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

	// Create SCEP client
	scepURL := s.Server.URL + "/api/fleet/orbit/host_identity/scep"
	timeout := 30 * time.Second
	scepClient, err := scepclient.New(scepURL, s.Logger, scepclient.WithTimeout(&timeout))
	require.NoError(t, err)

	// Get CA certificate
	caCertsBytes, _, err := scepClient.GetCACert(ctx, "")
	require.NoError(t, err)
	caCerts, err := x509.ParseCertificates(caCertsBytes)
	require.NoError(t, err)
	require.NotEmpty(t, caCerts, "no CA certificates returned")

	// Create temporary RSA key and cert for SCEP protocol
	tempRSAKey, deviceCert := createTempRSAKeyAndCert(t, hostIdentifier)

	// Create SCEP PKI message
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   tempRSAKey,
		SignerCert:  deviceCert,
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq)
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
		return httpResp, nil
	}

	// For other errors, fail the test
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Expected HTTP 200 but got %s", httpResp.Status)

	// Read response body
	respBytes, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)

	// Check response status
	require.Equal(t, scep.SUCCESS, pkiMsgResp.PKIStatus, "SCEP request failed with status: %v, failInfo: %s", pkiMsgResp.PKIStatus, pkiMsgResp.FailInfo)

	// Decrypt PKI envelope
	err = pkiMsgResp.DecryptPKIEnvelope(deviceCert, tempRSAKey)
	require.NoError(t, err)

	// Extract certificate
	certRepMsg := pkiMsgResp.CertRepMessage
	require.NotNil(t, certRepMsg, "no certificate response message")
	require.NotNil(t, certRepMsg.Certificate, "no certificate in SCEP response")

	return httpResp, certRepMsg.Certificate
}
