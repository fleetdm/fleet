//go:build !windows

// Windows is disabled because the TPM simulator requires CGO, which causes lint failures on Windows.

package hostidentity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/stretchr/testify/require"
)

func TestSCEPRateLimit(t *testing.T) {
	// Set up suite with rate limiting configuration
	cooldown := 5 * time.Minute
	s := SetUpSuiteWithConfig(t, "integrationtest.HostIdentitySCEPRateLimit", false, func(cfg *config.FleetConfig) {
		cfg.Osquery.EnrollCooldown = cooldown
	})

	defer mysqltest.TruncateTables(t, s.BaseSuite.DS, []string{
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
		require.Equal(t, hostID, initialCert.Subject.CommonName)

		// A valid certificate for the name already exists, so the shared secret cannot be used to request another one
		_, err = requestSCEPCertificate(t, s, hostID)
		require.Error(t, err)

		// Different host should be able to get certificate
		differentHostID := "test-host-different"
		differentHostCert, err := requestSCEPCertificate(t, s, differentHostID)
		require.NoError(t, err)
		require.Equal(t, differentHostID, differentHostCert.Subject.CommonName)
	})
}

// requestSCEPCertificate enrolls hostIdentifier with Fleet's host identity SCEP server.
func requestSCEPCertificate(t *testing.T, s *Suite, hostIdentifier string) (*x509.Certificate, error) {
	t.Helper()
	ctx := t.Context()

	eccPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
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

	timeout := 30 * time.Second
	scepClient, err := scepclient.New(s.Server.URL+"/api/fleet/orbit/host_identity/scep", s.Logger, scepclient.WithTimeout(&timeout))
	require.NoError(t, err)
	caCerts, err := enrollment.FetchCACerts(ctx, scepClient)
	require.NoError(t, err)
	signerKey, signerCert, err := enrollment.NewEphemeralSigner(csr.Subject)
	require.NoError(t, err)
	return enrollment.Enroll(ctx, scepClient, caCerts, enrollment.Request{
		CSR:        csr,
		SignerKey:  signerKey,
		SignerCert: signerCert,
		Logger:     s.Logger,
	})
}
