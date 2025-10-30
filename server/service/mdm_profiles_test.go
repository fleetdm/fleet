package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestValidateProfileCertificateAuthorityVariables(t *testing.T) {
	t.Parallel()
	groupedCAs := &fleet.GroupedCertificateAuthorities{
		DigiCert: []fleet.DigiCertCA{
			newMockDigicertCA("https://example.com", "caName"),
		},
		CustomScepProxy: []fleet.CustomSCEPProxyCA{
			newMockCustomSCEPProxyCA("https://example.com", "scepName"),
		},
		Smallstep: []fleet.SmallstepSCEPProxyCA{
			newMockSmallstepSCEPProxyCA("https://example.com", "https://example.com/challenge", "smallstepName"),
		},
	}

	cases := []struct {
		name    string
		profile string
		errMsg  string
	}{
		{
			name: "DigiCert badCA",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_bad", "$FLEET_VAR_DIGICERT_DATA_bad", "Name",
				"com.apple.security.pkcs12"),
			errMsg: "_bad does not exist",
		},

		{
			name: "DigiCert password shows up an extra time",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "$FLEET_VAR_DIGICERT_DATA_caName",
				"$FLEET_VAR_DIGICERT_PASSWORD_caName",
				"com.apple.security.pkcs12"),
			errMsg: "$FLEET_VAR_DIGICERT_PASSWORD_caName is already present in configuration profile",
		},
		{
			name: "DigiCert data shows up an extra time",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "$FLEET_VAR_DIGICERT_DATA_caName",
				"$FLEET_VAR_DIGICERT_DATA_caName",
				"com.apple.security.pkcs12"),
			errMsg: "$FLEET_VAR_DIGICERT_DATA_caName is already present in configuration profile",
		},
		{
			name: "Custom SCEP badCA",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_bad", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_bad", "Name",
				"com.apple.security.scep"),
			errMsg: "_bad does not exist",
		},
		{
			name: "Custom SCEP challenge shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName",
				"com.apple.security.scep"),
			errMsg: "$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName is already present in configuration profile",
		},
		{
			name: "Custom SCEP url shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"com.apple.security.scep"),
			errMsg: "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName is already present in configuration profile",
		},
		{
			name: "NDES challenge shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"$FLEET_VAR_NDES_SCEP_CHALLENGE",
				"com.apple.security.scep"),
			errMsg: "$FLEET_VAR_NDES_SCEP_CHALLENGE is already present in configuration profile",
		},
		{
			name: "NDES url shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"com.apple.security.scep"),
			errMsg: "$FLEET_VAR_NDES_SCEP_PROXY_URL is already present in configuration profile",
		},
		{
			name: "Smallstep badCA",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_bad", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_bad", "Name",
				"com.apple.security.scep"),
			errMsg: "_bad does not exist",
		},
		{
			name: "Smallstep challenge shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName",
				"com.apple.security.scep"),
			errMsg: "$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName is already present in configuration profile",
		},
		{
			name: "Smallstep url shows up an extra time",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"com.apple.security.scep"),
			errMsg: "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName is already present in configuration profile",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pass a premium license for testing (we're not testing license validation here)
			premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
			digicertVars, customScepVars, ndesVars, smallstepVars, err := validateProfileCertificateAuthorityVariables(tc.profile, premiumLic, groupedCAs)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
				require.Nil(t, digicertVars)
				require.Nil(t, customScepVars)
				require.Nil(t, ndesVars)
				require.Nil(t, smallstepVars)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
