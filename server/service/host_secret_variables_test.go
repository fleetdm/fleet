package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// Host-secret placeholders are written only by Fleet into the profiles and
// commands it manages; every user-provided content path must refuse them, or a
// profile author could have Fleet inject another host secret (recovery lock
// password, unlock token, enroll secret) into a payload of their choosing.
func TestUserProvidedContentRejectsHostSecretPlaceholders(t *testing.T) {
	premium := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	placeholders := []string{
		fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret),
		fleet.HostSecretPlaceholder(fleet.HostSecretRecoveryLockPassword),
		"${" + fleet.HostSecretPrefix + fleet.HostSecretMDMUnlockToken + "}",
	}

	validators := map[string]func(contents string) error{
		"apple configuration profile": func(contents string) error {
			_, err := validateConfigProfileFleetVariables(contents, premium, &fleet.GroupedCertificateAuthorities{})
			return err
		},
		"apple declaration": func(contents string) error {
			_, err := validateDeclarationFleetVariables(contents, premium)
			return err
		},
		"windows profile": func(contents string) error {
			_, err := validateWindowsProfileFleetVariables(contents, premium, &fleet.GroupedCertificateAuthorities{})
			return err
		},
	}

	for name, validate := range validators {
		t.Run(name, func(t *testing.T) {
			for _, placeholder := range placeholders {
				err := validate("<string>value: " + placeholder + "</string>")
				require.Error(t, err, placeholder)
				var badRequest *fleet.BadRequestError
				require.ErrorAs(t, err, &badRequest)
				require.Contains(t, err.Error(), "reserved for profiles managed by Fleet")
			}

			// content without a placeholder is unaffected, including a legitimate
			// Fleet variable and a plain org-wide secret reference
			require.NoError(t, validate("<string>$FLEET_VAR_HOST_UUID $FLEET_SECRET_MY_VALUE</string>"))
			require.NoError(t, validate("<string>no variables here</string>"))
		})
	}
}
