package packaging

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBypassEndUserAuthTemplates verifies the --bypass-end-user-auth switch is wired into the generated Linux env file
// when enabled, and absent when not. macOS is intentionally excluded. The Windows MSI service arguments are covered by
// TestServiceArguments in orbit/pkg/packaging/msi.
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

	// render executes tmpl with the bypass option toggled and returns the generated output.
	render := func(t *testing.T, tmpl *template.Template, bypass bool) string {
		t.Helper()
		opt := baseOpt
		opt.BypassEndUserAuth = bypass
		var buf bytes.Buffer
		require.NoError(t, tmpl.Execute(&buf, opt))
		return buf.String()
	}

	t.Run("linux env file", func(t *testing.T) {
		assert.Contains(t, render(t, envTemplate, true), "ORBIT_BYPASS_END_USER_AUTH=true")
		assert.NotContains(t, render(t, envTemplate, false), "ORBIT_BYPASS_END_USER_AUTH")
	})

	// Guard the deliberate macOS exclusion: the flag must never leak into the launchd plist.
	t.Run("macos launchd plist excluded", func(t *testing.T) {
		assert.NotContains(t, render(t, macosLaunchdTemplate, true), "ORBIT_BYPASS_END_USER_AUTH")
	})
}
