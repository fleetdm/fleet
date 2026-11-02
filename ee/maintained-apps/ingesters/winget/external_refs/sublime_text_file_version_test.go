package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSublimeTextFileVersion(t *testing.T) {
	tcs := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{name: "build 4215", url: "https://download.sublimetext.com/sublime_text_build_4215_x64_setup.exe", expected: "4.2.1.5"},
		{name: "build 4200", url: "https://download.sublimetext.com/sublime_text_build_4200_x64_setup.exe", expected: "4.2.0.0"},
		{name: "build 4180", url: "https://download.sublimetext.com/sublime_text_build_4180_x64_setup.exe", expected: "4.1.8.0"},
		{name: "no build in URL", url: "https://download.sublimetext.com/latest/windows", wantErr: true},
		{name: "5-digit build", url: "https://download.sublimetext.com/sublime_text_build_10001_x64_setup.exe", wantErr: true},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{
				Slug:         "sublime-text/windows",
				Version:      "4.0.0.421500",
				InstallerURL: tc.url,
			}
			result, err := SublimeTextFileVersion(app)
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, "4.0.0.421500", result.Version)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}
