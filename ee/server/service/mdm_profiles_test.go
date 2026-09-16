package service

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/stretchr/testify/require"
)

// testCertB64 is a stand-in for the CA cert; its value never varies across the
// cases below, so any diff between them comes from the payload guards.
const testCertB64 = "VEVTVF9DRVJUSUZJQ0FURQ=="

func renderFileVaultProfile(t *testing.T, enforcement, escrow bool) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, fileVaultProfileTemplate.Execute(&buf, fileVaultProfileOptions{
		PayloadIdentifier:    mobileconfig.FleetFileVaultPayloadIdentifier,
		PayloadName:          mdm.FleetFileVaultProfileName,
		Base64DerCertificate: testCertB64,
		EnableEnforcement:    enforcement,
		EnableEscrow:         escrow,
	}))
	return buf.String()
}

// dictsOf returns the PayloadType of every payload in the profile, in order.
func dictsOf(t *testing.T, profile string) []string {
	t.Helper()
	var types []string
	for line := range strings.SplitSeq(profile, "\n") {
		line = strings.TrimSpace(line)
		const prefix, suffix = "<string>", "</string>"
		if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
			continue
		}
		v := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
		switch v {
		case mobileconfig.FleetFileVaultPayloadType,
			mobileconfig.FleetRecoveryKeyEscrowPayloadType,
			"com.apple.security.pkcs1",
			mobileconfig.FleetCustomSettingsPayloadType:
			types = append(types, v)
		}
	}
	return types
}

// TestFileVaultProfileBothOnIsByteStable is the load-bearing assertion of the
// per-platform work: an existing customer with both macOS settings on must get
// a profile identical to what Fleet shipped before the settings were split, or
// every Mac gets a needless re-push on upgrade.
func TestFileVaultProfileBothOnIsByteStable(t *testing.T) {
	want, err := os.ReadFile("testdata/filevault/both_on.xml")
	require.NoError(t, err)
	require.Equal(t, string(want), renderFileVaultProfile(t, true, true))
}

func TestFileVaultProfilePayloadsPerCombination(t *testing.T) {
	bothOn := renderFileVaultProfile(t, true, true)

	t.Run("both on carries all four payloads", func(t *testing.T) {
		require.Equal(t, []string{
			mobileconfig.FleetFileVaultPayloadType,
			mobileconfig.FleetRecoveryKeyEscrowPayloadType,
			"com.apple.security.pkcs1",
			mobileconfig.FleetCustomSettingsPayloadType,
		}, dictsOf(t, bothOn))
		require.Contains(t, bothOn, "<key>ShowRecoveryKey</key>\n\t\t\t<false/>")
	})

	t.Run("enforce only drops the escrow payloads and shows the recovery key", func(t *testing.T) {
		got := renderFileVaultProfile(t, true, false)
		require.Equal(t, []string{
			mobileconfig.FleetFileVaultPayloadType,
			mobileconfig.FleetCustomSettingsPayloadType,
		}, dictsOf(t, got))
		// with nowhere to escrow the key, the user has to be shown it
		require.Contains(t, got, "<key>ShowRecoveryKey</key>\n\t\t\t<true/>")
		require.NotContains(t, got, testCertB64)
	})

	t.Run("escrow only drops the enforcement payloads", func(t *testing.T) {
		got := renderFileVaultProfile(t, false, true)
		require.Equal(t, []string{
			mobileconfig.FleetRecoveryKeyEscrowPayloadType,
			"com.apple.security.pkcs1",
		}, dictsOf(t, got))
		// nothing forces or prevents disabling FileVault; Fleet only escrows
		require.NotContains(t, got, "dontAllowFDEDisable")
		require.NotContains(t, got, "<key>ShowRecoveryKey</key>")
		require.Contains(t, got, testCertB64)
	})

	t.Run("every combination is well-formed and structurally whole", func(t *testing.T) {
		for _, tc := range []struct{ enforcement, escrow bool }{
			{true, true}, {true, false}, {false, true},
		} {
			got := renderFileVaultProfile(t, tc.enforcement, tc.escrow)
			require.Equal(t, strings.Count(got, "<dict>"), strings.Count(got, "</dict>"))
			require.Equal(t, strings.Count(got, "<array>"), strings.Count(got, "</array>"))
			// no guard left a blank or whitespace-only line behind
			for line := range strings.SplitSeq(got, "\n") {
				require.NotEmpty(t, strings.TrimSpace(line))
			}
			require.NotContains(t, got, "{{")
		}
	})
}
