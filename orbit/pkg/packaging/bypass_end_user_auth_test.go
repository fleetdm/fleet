package packaging

import (
	"bytes"
	"regexp"
	"strconv"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBypassEndUserAuthTemplates verifies the --bypass-end-user-auth switch is wired into the generated Linux env file
// and Windows MSI service environment when enabled, and absent when not, plus the Windows BYPASS_END_USER_AUTH MSI property.
// macOS is intentionally excluded.
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

	t.Run("windows msi service environment", func(t *testing.T) {
		opt := baseOpt
		opt.BypassEndUserAuth = true
		assert.Contains(t, windowsServiceEnvironment(t, opt), "ORBIT_BYPASS_END_USER_AUTH=true")
		opt.BypassEndUserAuth = false
		assert.NotContains(t, windowsServiceEnvironment(t, opt), "ORBIT_BYPASS_END_USER_AUTH=true")
	})

	t.Run("windows msi BYPASS_END_USER_AUTH property", func(t *testing.T) {
		propertyRe := regexp.MustCompile(`<Property Id="BYPASS_END_USER_AUTH" Value="([^"]*)" Secure="yes"/>`)
		for _, tc := range []struct {
			name         string
			bypass       bool
			wantProperty string
		}{
			{name: "defaults to false without the flag", bypass: false, wantProperty: "False"},
			{name: "flag sets the default to true", bypass: true, wantProperty: "True"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				opt := baseOpt
				opt.EnableBypassEndUserAuthProperty = true
				opt.BypassEndUserAuth = tc.bypass

				var buf bytes.Buffer
				require.NoError(t, windowsWixTemplate.Execute(&buf, opt))
				match := propertyRe.FindStringSubmatch(buf.String())
				require.Len(t, match, 2)
				assert.Equal(t, tc.wantProperty, match[1])
				// Orbit reads ORBIT_BYPASS_END_USER_AUTH as a bool flag and exits if the value doesn't parse.
				_, err := strconv.ParseBool(match[1])
				require.NoError(t, err)

				entries := windowsServiceEnvironment(t, opt)
				assert.Contains(t, entries, "ORBIT_BYPASS_END_USER_AUTH=[BYPASS_END_USER_AUTH]")
				assert.NotContains(t, entries, "ORBIT_BYPASS_END_USER_AUTH=true")
			})
		}

		t.Run("property absent when disabled", func(t *testing.T) {
			opt := baseOpt
			opt.BypassEndUserAuth = true

			var buf bytes.Buffer
			require.NoError(t, windowsWixTemplate.Execute(&buf, opt))
			assert.NotContains(t, buf.String(), "BYPASS_END_USER_AUTH\"")
			assert.NotContains(t, buf.String(), "[BYPASS_END_USER_AUTH]")
		})
	})

	// Guard the deliberate macOS exclusion: the flag must never leak into the launchd plist.
	t.Run("macos launchd plist excluded", func(t *testing.T) {
		assert.NotContains(t, render(t, macosLaunchdTemplate, true), "ORBIT_BYPASS_END_USER_AUTH")
	})
}
