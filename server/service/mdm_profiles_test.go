package service

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// This test covers validation errors and not happy paths, that should be done in the respective platforms calling this, as they may depend on additional validations.
func TestValidateProfileCertificateAuthorityVariables(t *testing.T) {
	t.Parallel()
	groupedCAs := &fleet.GroupedCertificateAuthorities{
		DigiCert: []fleet.DigiCertCA{
			newMockDigicertCA("https://example.com", "caName"),
			newMockDigicertCA("https://example.com", "caName2"),
		},
		CustomScepProxy: []fleet.CustomSCEPProxyCA{
			newMockCustomSCEPProxyCA("https://example.com", "scepName"),
			newMockCustomSCEPProxyCA("https://example.com", "scepName2"),
		},
		Smallstep: []fleet.SmallstepSCEPProxyCA{
			newMockSmallstepSCEPProxyCA("https://example.com", "https://example.com/challenge", "smallstepName"),
			newMockSmallstepSCEPProxyCA("https://example.com", "https://example.com/challenge", "smallstepName2"),
		},
	}

	cases := []struct {
		name     string
		profile  string
		errMsg   string
		platform string // Should be one of fleet.MDMPlatformApple, fleet.MDMPlatformMicrosoft
	}{
		{
			name: "DigiCert badCA",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_bad", "$FLEET_VAR_DIGICERT_DATA_bad", "Name",
				"com.apple.security.pkcs12"),
			errMsg:   "_bad does not exist",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "DigiCert password shows up an extra time",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "$FLEET_VAR_DIGICERT_DATA_caName",
				"$FLEET_VAR_DIGICERT_PASSWORD_caName",
				"com.apple.security.pkcs12"),
			errMsg:   "$FLEET_VAR_DIGICERT_PASSWORD_caName is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "DigiCert data shows up an extra time",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "$FLEET_VAR_DIGICERT_DATA_caName",
				"$FLEET_VAR_DIGICERT_DATA_caName",
				"com.apple.security.pkcs12"),
			errMsg:   "$FLEET_VAR_DIGICERT_DATA_caName is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name:     "DigiCert password missing",
			profile:  digiCertForValidation("password", "$FLEET_VAR_DIGICERT_DATA_caName", "Name", "com.apple.security.pkcs12"),
			errMsg:   "Missing $FLEET_VAR_DIGICERT_PASSWORD_caName",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "DigiCert data missing",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "data", "Name",
				"com.apple.security.pkcs12"),
			errMsg:   "Missing $FLEET_VAR_DIGICERT_DATA_caName",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "DigiCert password and data CA names don't match",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "$FLEET_VAR_DIGICERT_DATA_caName2", "Name",
				"com.apple.security.pkcs12"),
			errMsg:   "Missing $FLEET_VAR_DIGICERT_DATA_caName in the profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Custom SCEP badCA",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_bad", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_bad", "Name",
				"com.apple.security.scep"),
			errMsg:   "_bad does not exist",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Custom SCEP challenge shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName",
				"com.apple.security.scep"),
			errMsg:   "$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Custom SCEP url shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"com.apple.security.scep"),
			errMsg:   "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name:     "Custom SCEP challenge missing for apple",
			profile:  customSCEPForValidation("challenge", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName", "Name", "com.apple.security.scep"),
			errMsg:   "SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>, $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>, and $FLEET_VAR_SCEP_RENEWAL_ID variables.",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Custom SCEP url missing for apple",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "https://bozo.com", "Name",
				"com.apple.security.scep"),
			errMsg:   "SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>, $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>, and $FLEET_VAR_SCEP_RENEWAL_ID variables.",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Custom SCEP renewal ID missing for apple",
			profile: strings.Replace(customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"Name",
				"com.apple.security.scep"), "$FLEET_VAR_SCEP_RENEWAL_ID", "", 1),
			errMsg:   "SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>, $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>, and $FLEET_VAR_SCEP_RENEWAL_ID variables.",
			platform: fleet.MDMPlatformApple,
		},
		{
			name:     "Custom SCEP challenge missing for apple",
			profile:  customSCEPForValidation("challenge", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName", "Name", "com.apple.security.scep"),
			errMsg:   "SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME> and $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME> variables.",
			platform: fleet.MDMPlatformMicrosoft,
		},
		{
			name: "Custom SCEP url missing for apple",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "https://bozo.com", "Name",
				"com.apple.security.scep"),
			errMsg:   "SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME> and $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME> variables.",
			platform: fleet.MDMPlatformMicrosoft,
		},
		{
			name: "Custom SCEP challenge and url CA names don't match",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName2",
				"Name", "com.apple.security.scep"),
			errMsg:   "Missing $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName in the profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "NDES challenge shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"$FLEET_VAR_NDES_SCEP_CHALLENGE",
				"com.apple.security.scep"),
			errMsg:   "$FLEET_VAR_NDES_SCEP_CHALLENGE is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "NDES url shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"com.apple.security.scep"),
			errMsg:   "$FLEET_VAR_NDES_SCEP_PROXY_URL is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name:     "NDES challenge missing",
			profile:  customSCEPForValidation("challenge", "$FLEET_VAR_NDES_SCEP_PROXY_URL", "Name", "com.apple.security.scep"),
			errMsg:   fleet.NDESSCEPVariablesMissingErrMsg,
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "NDES url missing",
			profile: customSCEPForValidation("$FLEET_VAR_NDES_SCEP_CHALLENGE", "https://bozo.com", "Name",
				"com.apple.security.scep"),
			errMsg:   fleet.NDESSCEPVariablesMissingErrMsg,
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Smallstep badCA",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_bad", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_bad", "Name",
				"com.apple.security.scep"),
			errMsg:   "_bad does not exist",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Smallstep challenge shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName",
				"com.apple.security.scep"),
			errMsg:   "$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Smallstep url shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"com.apple.security.scep"),
			errMsg:   "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName is already present in configuration profile",
			platform: fleet.MDMPlatformApple,
		},
		{
			name:     "Smallstep challenge missing",
			profile:  customSCEPForValidation("challenge", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName", "Name", "com.apple.security.scep"),
			errMsg:   "Smallstep certificate authority requires: $FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>, $FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>, and $FLEET_VAR_SCEP_RENEWAL_ID variables.",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Smallstep url missing",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "https://bozo.com", "Name",
				"com.apple.security.scep"),
			errMsg:   "Smallstep certificate authority requires: $FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>, $FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>, and $FLEET_VAR_SCEP_RENEWAL_ID variables.",
			platform: fleet.MDMPlatformApple,
		},
		{
			name: "Smallstep challenge and url CA names don't match",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName2",
				"Name", "com.apple.security.scep"),
			errMsg:   "Missing $FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName in the profile",
			platform: fleet.MDMPlatformApple,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pass a premium license for testing (we're not testing license validation here)
			premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
			err := validateProfileCertificateAuthorityVariables(tc.profile, premiumLic, tc.platform, groupedCAs, nil, nil, nil, nil)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
