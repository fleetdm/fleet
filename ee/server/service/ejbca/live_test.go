package ejbca

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

// TestLiveEJBCA runs the client against a real EJBCA, for example the keyfactor/ejbca-ce container. It is skipped unless
// EJBCA_LIVE_URL is set. The PEM files are the client certificate and key of an EJBCA role that may enroll with the profiles, such as
// the output of openssl pkcs12 -in fleet.p12 -nodes:
//
//	EJBCA_LIVE_URL=https://localhost:8443 EJBCA_LIVE_CERT=cert.pem EJBCA_LIVE_KEY=key.pem EJBCA_LIVE_TRUST_BUNDLE=ca.pem \
//	EJBCA_LIVE_CA=ManagementCA EJBCA_LIVE_CERT_PROFILE=cert-profile EJBCA_LIVE_EE_PROFILE=ee-profile \
//	go test ./ee/server/service/ejbca -run TestLiveEJBCA -v
func TestLiveEJBCA(t *testing.T) {
	url := os.Getenv("EJBCA_LIVE_URL")
	if url == "" {
		t.Skip("EJBCA_LIVE_URL is not set")
	}
	read := func(env string) string {
		data, err := os.ReadFile(os.Getenv(env))
		require.NoError(t, err)
		return string(data)
	}

	config := fleet.EJBCACA{
		URL:                           url,
		MTLSCertPEM:                   read("EJBCA_LIVE_CERT"),
		MTLSKeyPEM:                    read("EJBCA_LIVE_KEY"),
		TrustCABundlePEM:              read("EJBCA_LIVE_TRUST_BUNDLE"),
		CertificateAuthorityName:      os.Getenv("EJBCA_LIVE_CA"),
		CertificateProfileName:        os.Getenv("EJBCA_LIVE_CERT_PROFILE"),
		EndEntityProfileName:          os.Getenv("EJBCA_LIVE_EE_PROFILE"),
		CertificateCommonName:         "fleet-live-test@example.com",
		UsernameTemplate:              "FLEET-LIVE-TEST",
		CertificateUserPrincipalNames: []string{"fleet-live-test@example.com"},
	}
	s := NewService()
	require.NoError(t, s.VerifyConnection(t.Context(), config))

	// Enrolling the same username again has to work, as it does when Fleet reissues a certificate.
	for range 2 {
		cert, err := s.GetCertificate(t.Context(), config)
		require.NoError(t, err)
		_, issued, err := pkcs12.Decode(cert.PfxData, cert.Password)
		require.NoError(t, err)
		require.Equal(t, config.CertificateCommonName, issued.Subject.CommonName)
		require.Equal(t, config.CertificateUserPrincipalNames, upnsInCertificate(t, issued))
	}
}
