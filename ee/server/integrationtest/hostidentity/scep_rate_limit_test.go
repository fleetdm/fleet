package hostidentity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
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
	s := SetUpSuiteWithConfig(t, "integrationtest.SCEPRateLimit", false, func(cfg *config.FleetConfig) {
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
		initialCert, err := requestSCEPCertificate(t, s, hostID)
		require.NoError(t, err)
		require.NotNil(t, initialCert)
		assert.Equal(t, hostID, initialCert.Subject.CommonName)

		// Second certificate request immediately after - should fail due to rate limit
		rateLimitedCert, err := requestSCEPCertificate(t, s, hostID)
		require.Error(t, err)
		require.Nil(t, rateLimitedCert)
		// The error will be a generic SCEP failure, but we can verify rate limiting is working
		// by confirming the request fails when it should succeed without rate limiting

		// Wait for a small duration (less than cooldown) and try again - should still fail
		time.Sleep(1 * time.Second)
		stillRateLimitedCert, err := requestSCEPCertificate(t, s, hostID)
		require.Error(t, err)
		require.Nil(t, stillRateLimitedCert)

		// Different host should be able to get certificate
		differentHostID := "test-host-different"
		differentHostCert, err := requestSCEPCertificate(t, s, differentHostID)
		require.NoError(t, err)
		require.NotNil(t, differentHostCert)
		assert.Equal(t, differentHostID, differentHostCert.Subject.CommonName)
	})
}

// requestSCEPCertificate is a helper function to request a SCEP certificate for a given host identifier
func requestSCEPCertificate(t *testing.T, s *Suite, hostIdentifier string) (*x509.Certificate, error) {
	ctx := context.Background()

	// Create ECC private key
	eccPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

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
	if err != nil {
		return nil, err
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		return nil, err
	}

	// Create SCEP client
	scepURL := s.Server.URL + "/api/fleet/orbit/host_identity/scep"
	timeout := 30 * time.Second
	scepClient, err := scepclient.New(scepURL, s.Logger, &timeout)
	if err != nil {
		return nil, err
	}

	// Get CA certificate
	caCertsBytes, _, err := scepClient.GetCACert(ctx, "")
	if err != nil {
		return nil, err
	}
	caCerts, err := x509.ParseCertificates(caCertsBytes)
	if err != nil {
		return nil, err
	}
	if len(caCerts) == 0 {
		return nil, fmt.Errorf("no CA certificates returned")
	}

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
	if err != nil {
		return nil, err
	}

	// Send PKI operation request
	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, err
	}

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithCACerts(msg.Recipients))
	if err != nil {
		return nil, err
	}

	// Check response status
	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		if pkiMsgResp.FailInfo != "" {
			return nil, fmt.Errorf("SCEP request failed: %s", pkiMsgResp.FailInfo)
		}
		return nil, fmt.Errorf("SCEP request failed with status: %v", pkiMsgResp.PKIStatus)
	}

	// Decrypt PKI envelope
	err = pkiMsgResp.DecryptPKIEnvelope(deviceCert, tempRSAKey)
	if err != nil {
		return nil, err
	}

	// Extract certificate
	certRepMsg := pkiMsgResp.CertRepMessage
	if certRepMsg == nil || certRepMsg.Certificate == nil {
		return nil, fmt.Errorf("no certificate in SCEP response")
	}

	return certRepMsg.Certificate, nil
}
