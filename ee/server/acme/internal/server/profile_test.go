package server

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACMEDirectoryURLVariableSubstitution verifies that the
// $FLEET_VAR_ACME_DIRECTORY_URL_<CA_NAME> variable is correctly
// substituted in a mobileconfig profile.
func TestACMEDirectoryURLVariableSubstitution(t *testing.T) {
	profileXML := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>DirectoryURL</key>
			<string>$FLEET_VAR_ACME_DIRECTORY_URL_smallstep-ra</string>
			<key>PayloadType</key>
			<string>com.apple.security.acme</string>
		</dict>
	</array>
</dict>
</plist>`

	// Simulate the substitution that profile_processor.go does
	caName := "smallstep-ra"
	serverURL := "https://fleet.example.com"
	acmeDirectoryURL := serverURL + "/api/acme/" + caName + "/directory"

	result, err := profiles.ReplaceExactFleetPrefixVariableInXML(
		string(fleet.FleetVarACMEDirectoryURLPrefix),
		caName,
		profileXML,
		acmeDirectoryURL,
	)
	require.NoError(t, err)

	assert.Contains(t, result, "https://fleet.example.com/api/acme/smallstep-ra/directory")
	assert.NotContains(t, result, "FLEET_VAR_ACME")

	t.Logf("Substituted profile:\n%s", result)
}

// TestACMEDirectoryURLVariableBothFormats verifies both $VAR and ${VAR} formats work.
func TestACMEDirectoryURLVariableBothFormats(t *testing.T) {
	for _, format := range []string{
		`<string>$FLEET_VAR_ACME_DIRECTORY_URL_myca</string>`,
		`<string>${FLEET_VAR_ACME_DIRECTORY_URL_myca}</string>`,
	} {
		t.Run(format, func(t *testing.T) {
			profileXML := `<dict><key>DirectoryURL</key>` + format + `</dict>`

			result, err := profiles.ReplaceExactFleetPrefixVariableInXML(
				string(fleet.FleetVarACMEDirectoryURLPrefix),
				"myca",
				profileXML,
				"https://fleet.example.com/api/acme/myca/directory",
			)
			require.NoError(t, err)
			assert.Contains(t, result, "https://fleet.example.com/api/acme/myca/directory")
			assert.NotContains(t, result, "FLEET_VAR")
		})
	}
}
