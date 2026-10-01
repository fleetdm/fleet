package packaging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsWixTemplateEUAToken(t *testing.T) {
	baseOpt := Options{
		FleetURL:        "https://fleet.example.com",
		EnrollSecret:    "secret",
		OrbitChannel:    "stable",
		OsquerydChannel: "stable",
		DesktopChannel:  "stable",
		NativePlatform:  "windows",
		Architecture:    ArchAmd64,
	}

	t.Run("EUA_TOKEN property and env var included when enabled", func(t *testing.T) {
		opt := baseOpt
		opt.EnableEUATokenProperty = true

		var buf bytes.Buffer
		require.NoError(t, windowsWixTemplate.Execute(&buf, opt))
		assert.Contains(t, buf.String(), `<Property Id="EUA_TOKEN" Value="dummy"/>`)
		assert.Contains(t, windowsServiceEnvironment(t, opt), "ORBIT_EUA_TOKEN=[EUA_TOKEN]")
	})

	t.Run("EUA_TOKEN property and env var absent when disabled", func(t *testing.T) {
		opt := baseOpt
		opt.EnableEUATokenProperty = false

		var buf bytes.Buffer
		require.NoError(t, windowsWixTemplate.Execute(&buf, opt))
		assert.NotContains(t, buf.String(), `EUA_TOKEN`)
	})
}
