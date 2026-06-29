package service

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// forceKeysBlock is exactly the two FileVault2 force keys (and surrounding
// whitespace) that prompt_enablement_at=login emits and logout drops.
const forceKeysBlock = `
			<key>DeferForceAtUserLoginMaxBypassAttempts</key>
			<integer>0</integer>
			<key>ForceEnableInSetupAssistant</key>
			<true/>`

func renderFileVaultProfile(t *testing.T, promptAt string) string {
	t.Helper()
	var sb strings.Builder
	err := fileVaultProfileTemplate.Execute(&sb, fileVaultProfileOptions{
		PayloadIdentifier:    "com.fleetdm.fleet.mdm.filevault",
		PayloadName:          "Disk encryption",
		Base64DerCertificate: "Zm9v", // "foo"
		PromptEnablementAt:   promptAt,
	})
	require.NoError(t, err)
	return sb.String()
}

func TestFileVaultProfileTemplatePromptEnablementAt(t *testing.T) {
	login := renderFileVaultProfile(t, fleet.FileVaultPromptEnablementAtLogin)
	logout := renderFileVaultProfile(t, fleet.FileVaultPromptEnablementAtLogout)

	// login output contains both force keys.
	require.Contains(t, login, "DeferForceAtUserLoginMaxBypassAttempts")
	require.Contains(t, login, "ForceEnableInSetupAssistant")
	require.Contains(t, login, forceKeysBlock)

	// logout output drops exactly the two force keys, and is otherwise identical
	// to login (removing the force-key block from login yields logout byte-for-byte).
	require.NotContains(t, logout, "DeferForceAtUserLoginMaxBypassAttempts")
	require.NotContains(t, logout, "ForceEnableInSetupAssistant")
	require.Equal(t, strings.Replace(login, forceKeysBlock, "", 1), logout)

	// Both renderings keep the FileVault2 payload core and the other payloads.
	for _, out := range []string{login, logout} {
		require.Contains(t, out, "<key>Defer</key>")
		require.Contains(t, out, "<key>Enable</key>\n\t\t\t<string>On</string>")
		require.Contains(t, out, "<key>ShowRecoveryKey</key>")
		require.Contains(t, out, "<key>dontAllowFDEDisable</key>")
		require.Contains(t, out, "FileVault Recovery Key Escrow")
		require.Contains(t, out, "Certificate Root")
	}
}
