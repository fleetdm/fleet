package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestValidateCertificateAuthoritiesSpec(t *testing.T) {
	// TODO(hca): placeholder for additional tests to the extent not otherwise covered by client_test
}

func TestPreprocessCAFields(t *testing.T) {
	t.Run("DigiCert CA fields are trimmed and normalized", func(t *testing.T) {
		digiCertCA := &DigiCertCA{
			Name:      "  DigiCert CA  ",
			URL:       "  https://digicert.com  ",
			ProfileID: "  profile_id  ",
		}

		digiCertCA.Preprocess()

		require.Equal(t, "DigiCert CA", digiCertCA.Name)
		require.Equal(t, "https://digicert.com", digiCertCA.URL)
		require.Equal(t, "profile_id", digiCertCA.ProfileID)
	})

	t.Run("DigiCert CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		digiCertCAUpdate := &DigiCertCAUpdatePayload{
			Name:      ptr.String("  DigiCert CA  "),
			URL:       ptr.String("  https://digicert.com  "),
			ProfileID: ptr.String("  profile_id  "),
		}

		digiCertCAUpdate.Preprocess()

		require.Equal(t, "DigiCert CA", *digiCertCAUpdate.Name)
		require.Equal(t, "https://digicert.com", *digiCertCAUpdate.URL)
		require.Equal(t, "profile_id", *digiCertCAUpdate.ProfileID)
	})

	t.Run("Custom SCEP Proxy CA fields are trimmed and normalized", func(t *testing.T) {
		customSCEPProxyCA := &CustomSCEPProxyCA{
			Name: "  Custom SCEP Proxy CA  ",
			URL:  "  https://scep-proxy.com  ",
		}

		customSCEPProxyCA.Preprocess()

		require.Equal(t, "Custom SCEP Proxy CA", customSCEPProxyCA.Name)
		require.Equal(t, "https://scep-proxy.com", customSCEPProxyCA.URL)
	})

	t.Run("Custom SCEP Proxy CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		customSCEPProxyCAUpdate := &CustomSCEPProxyCAUpdatePayload{
			Name: ptr.String("  Custom SCEP Proxy CA  "),
			URL:  ptr.String("  https://scep-proxy.com  "),
		}

		customSCEPProxyCAUpdate.Preprocess()

		require.Equal(t, "Custom SCEP Proxy CA", *customSCEPProxyCAUpdate.Name)
		require.Equal(t, "https://scep-proxy.com", *customSCEPProxyCAUpdate.URL)
	})

	t.Run("NDES CA fields are trimmed and normalized", func(t *testing.T) {
		ndesCA := &NDESSCEPProxyCA{
			URL:      "  https://ndes.com  ",
			AdminURL: "  https://ndes.com/admin  ",
			Username: "  admin  ",
			Password: "  password  ",
		}

		ndesCA.Preprocess()

		require.Equal(t, "https://ndes.com", ndesCA.URL)
		require.Equal(t, "https://ndes.com/admin", ndesCA.AdminURL)
		require.Equal(t, "admin", ndesCA.Username)
		require.Equal(t, "  password  ", ndesCA.Password)
	})

	t.Run("NDES CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		ndesCAUpdate := &NDESSCEPProxyCAUpdatePayload{
			URL:      ptr.String("  https://ndes.com  "),
			AdminURL: ptr.String("  https://ndes.com/admin  "),
			Username: ptr.String("  admin  "),
			Password: ptr.String("  password  "),
		}

		ndesCAUpdate.Preprocess()

		require.Equal(t, "https://ndes.com", *ndesCAUpdate.URL)
		require.Equal(t, "https://ndes.com/admin", *ndesCAUpdate.AdminURL)
		require.Equal(t, "admin", *ndesCAUpdate.Username)
		require.Equal(t, "  password  ", *ndesCAUpdate.Password)
	})

	t.Run("Smallstep SCEP Proxy CA fields are trimmed and normalized", func(t *testing.T) {
		smallstepSCEPProxyCA := &SmallstepSCEPProxyCA{
			Name:         "  Smallstep SCEP Proxy CA  ",
			URL:          "  https://scep-proxy.com  ",
			ChallengeURL: "  https://scep-proxy.com/challenge  ",
			Username:     "  username  ",
			Password:     "  password  ",
		}

		smallstepSCEPProxyCA.Preprocess()

		require.Equal(t, "Smallstep SCEP Proxy CA", smallstepSCEPProxyCA.Name)
		require.Equal(t, "https://scep-proxy.com", smallstepSCEPProxyCA.URL)
		require.Equal(t, "https://scep-proxy.com/challenge", smallstepSCEPProxyCA.ChallengeURL)
		require.Equal(t, "username", smallstepSCEPProxyCA.Username)
		require.Equal(t, "  password  ", smallstepSCEPProxyCA.Password)
	})

	t.Run("Smallstep SCEP Proxy CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		smallstepSCEPProxyCAUpdate := &SmallstepSCEPProxyCAUpdatePayload{
			Name:         ptr.String("  Smallstep SCEP Proxy CA  "),
			URL:          ptr.String("  https://scep-proxy.com  "),
			ChallengeURL: ptr.String("  https://scep-proxy.com/challenge  "),
			Username:     ptr.String("  username  "),
			Password:     ptr.String("  password  "),
		}

		smallstepSCEPProxyCAUpdate.Preprocess()

		require.Equal(t, "Smallstep SCEP Proxy CA", *smallstepSCEPProxyCAUpdate.Name)
		require.Equal(t, "https://scep-proxy.com", *smallstepSCEPProxyCAUpdate.URL)
		require.Equal(t, "https://scep-proxy.com/challenge", *smallstepSCEPProxyCAUpdate.ChallengeURL)
		require.Equal(t, "username", *smallstepSCEPProxyCAUpdate.Username)
		require.Equal(t, "  password  ", *smallstepSCEPProxyCAUpdate.Password)
	})
}
