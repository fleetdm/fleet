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
		FleetURL:        "https://fleet.example.com",
		EnrollSecret:    "secret",
		OrbitChannel:    "stable",
		OsquerydChannel: "stable",
		DesktopChannel:  "stable",
		NativePlatform:  "windows",
		Architecture:    ArchAmd64,
	}

	t.Run("EUA_TOKEN property and flag included when enabled", func(t *testing.T) {
		opt := baseOpt
		opt.EnableEUATokenProperty = true

		var buf bytes.Buffer
		err := windowsWixTemplate.Execute(&buf, opt)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `<Property Id="EUA_TOKEN" Value="dummy"/>`)

		var argsLine string
		for line := range strings.SplitSeq(output, "\n") {
			if strings.Contains(line, "Arguments=") && strings.Contains(line, "--fleet-url") {
				argsLine = line
				break
			}
		}
		require.NotEmpty(t, argsLine, "ServiceInstall Arguments line not found in template output")
		assert.Contains(t, argsLine, `--eua-token="[EUA_TOKEN]"`,
			"eua-token flag should be in ServiceInstall Arguments")
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
}
