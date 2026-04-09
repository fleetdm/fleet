package packaging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsWixTemplateEUAToken(t *testing.T) {
	baseOpt := Options{
		FleetURL:      "https://fleet.example.com",
		EnrollSecret:  "secret",
		OrbitChannel:  "stable",
		OsquerydChannel: "stable",
		DesktopChannel: "stable",
		NativePlatform: "windows",
		Architecture:  ArchAmd64,
	}

	t.Run("EUA_TOKEN property and flag included when enabled", func(t *testing.T) {
		opt := baseOpt
		opt.EnableEUATokenProperty = true

		var buf bytes.Buffer
		err := windowsWixTemplate.Execute(&buf, opt)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `<Property Id="EUA_TOKEN" Value="dummy"/>`)
		assert.Contains(t, output, `--eua-token="[EUA_TOKEN]"`)
	})

	t.Run("EUA_TOKEN property and flag absent when disabled", func(t *testing.T) {
		opt := baseOpt
		opt.EnableEUATokenProperty = false

		var buf bytes.Buffer
		err := windowsWixTemplate.Execute(&buf, opt)
		require.NoError(t, err)

		output := buf.String()
		assert.NotContains(t, output, `EUA_TOKEN`)
		assert.NotContains(t, output, `--eua-token`)
	})

	t.Run("EUA_TOKEN flag appears in ServiceInstall Arguments", func(t *testing.T) {
		opt := baseOpt
		opt.EnableEUATokenProperty = true

		var buf bytes.Buffer
		err := windowsWixTemplate.Execute(&buf, opt)
		require.NoError(t, err)

		// Find the ServiceInstall Arguments line and verify eua-token is in it.
		for _, line := range strings.Split(buf.String(), "\n") {
			if strings.Contains(line, "Arguments=") && strings.Contains(line, "--fleet-url") {
				assert.Contains(t, line, `--eua-token="[EUA_TOKEN]"`,
					"eua-token flag should be in ServiceInstall Arguments")
				return
			}
		}
		t.Fatal("ServiceInstall Arguments line not found in template output")
	})
}
