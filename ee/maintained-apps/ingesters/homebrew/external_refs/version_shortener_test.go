package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeVersionShortener(t *testing.T) {
	shortener := makeVersionShortener(2)

	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "fewer segments than keep", version: "1", expected: "1"},
		{name: "equal segments to keep", version: "1.2", expected: "1.2"},
		{name: "one extra segment", version: "1.2.3", expected: "1.2"},
		{name: "many extra segments", version: "1.2.3.4.5", expected: "1.2"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "test-app"}
			result, err := shortener(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestMakeVersionShortenerKeep3(t *testing.T) {
	shortener := makeVersionShortener(3)

	tcs := []struct {
		name     string
		version  string
		slug     string
		expected string
	}{
		{name: "citrix workspace", version: "25.11.1.42", slug: "citrix-workspace", expected: "25.11.1"},
		{name: "grammarly desktop", version: "1.160.0.0", slug: "grammarly-desktop", expected: "1.160.0"},
		{name: "anka virtualization", version: "3.8.6.212", slug: "anka-virtualization", expected: "3.8.6"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: tc.slug}
			result, err := shortener(app)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestSublimeVersionTransformer(t *testing.T) {
	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "numeric version", version: "4200", expected: "Build 4200"},
		{name: "already prefixed", version: "Build 4200", expected: "Build 4200"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "sublime-text"}
			result, err := SublimeVersionTransformer(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestMySQLWorkbenchVersionTransformer(t *testing.T) {
	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "normal version", version: "8.0.46", expected: "8.0.46.CE"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "mysql-workbench"}
			result, err := MySQLWorkbenchVersionTransformer(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestLensVersionTransformer(t *testing.T) {
	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "normal version", version: "2026.3.251250", expected: "2026.3.251250-latest"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "lens"}
			result, err := LensVersionTransformer(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}
