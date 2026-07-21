package packaging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBypassEndUserAuthTemplates verifies that the --bypass-end-user-auth switch is wired into the
// generated package for every platform (Linux env file, macOS launchd plist, and Windows MSI), and
// that it is absent when the option is not set. See https://github.com/fleetdm/fleet/issues/46644.
func TestBypassEndUserAuthTemplates(t *testing.T) {
	baseOpt := Options{
		FleetURL:        "https://fleet.example.com",
		EnrollSecret:    "secret",
		OrbitChannel:    "stable",
		OsquerydChannel: "stable",
		DesktopChannel:  "stable",
		NativePlatform:  "windows",
		Architecture:    ArchAmd64,
	}

	t.Run("linux env file", func(t *testing.T) {
		t.Run("included when enabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = true
			var buf bytes.Buffer
			require.NoError(t, envTemplate.Execute(&buf, opt))
			assert.Contains(t, buf.String(), "ORBIT_BYPASS_END_USER_AUTH=true")
		})
		t.Run("absent when disabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = false
			var buf bytes.Buffer
			require.NoError(t, envTemplate.Execute(&buf, opt))
			assert.NotContains(t, buf.String(), "ORBIT_BYPASS_END_USER_AUTH")
		})
	})

	t.Run("macos launchd plist", func(t *testing.T) {
		t.Run("included when enabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = true
			var buf bytes.Buffer
			require.NoError(t, macosLaunchdTemplate.Execute(&buf, opt))
			assert.Contains(t, buf.String(), "<key>ORBIT_BYPASS_END_USER_AUTH</key>")
		})
		t.Run("absent when disabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = false
			var buf bytes.Buffer
			require.NoError(t, macosLaunchdTemplate.Execute(&buf, opt))
			assert.NotContains(t, buf.String(), "ORBIT_BYPASS_END_USER_AUTH")
		})
	})

	t.Run("windows msi", func(t *testing.T) {
		argsLine := func(t *testing.T, opt Options) string {
			t.Helper()
			var buf bytes.Buffer
			require.NoError(t, windowsWixTemplate.Execute(&buf, opt))
			for line := range strings.SplitSeq(buf.String(), "\n") {
				if strings.Contains(line, "Arguments=") && strings.Contains(line, "--fleet-url") {
					return line
				}
			}
			t.Fatal("ServiceInstall Arguments line not found in template output")
			return ""
		}

		t.Run("included when enabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = true
			assert.Contains(t, argsLine(t, opt), "--bypass-end-user-auth")
		})
		t.Run("absent when disabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = false
			assert.NotContains(t, argsLine(t, opt), "--bypass-end-user-auth")
		})
	})
}
