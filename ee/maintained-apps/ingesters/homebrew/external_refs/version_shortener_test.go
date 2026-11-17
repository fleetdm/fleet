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

func TestMakeVersionShortenerKeep2(t *testing.T) {
	shortener := makeVersionShortener(2)

	tcs := []struct {
		name     string
		version  string
		slug     string
		expected string
	}{
		{name: "android studio", version: "2025.3.2.6", slug: "android-studio", expected: "2025.3"},
		{name: "microsoft auto update", version: "4.82.26020434", slug: "microsoft-auto-update", expected: "4.82"},
		{name: "opera", version: "129.0.5823.28", slug: "opera", expected: "129.0"},
		{name: "twingate", version: "2026.29.22575", slug: "twingate", expected: "2026.29"},
		{name: "microsoft 365 copilot", version: "1.2608.0301", slug: "microsoft-365-copilot", expected: "1.2608"},
		{name: "numi", version: "3.35.743", slug: "numi", expected: "3.35"},
		{name: "origami studio", version: "231.0.0.0.0", slug: "origami-studio", expected: "231.0"},
		{name: "sqlpro hotfix", version: "2026.301.2", slug: "sqlpro-for-sqlite", expected: "2026.301"},
		{name: "sqlpro release", version: "2026.247", slug: "sqlpro-for-sqlite", expected: "2026.247"},
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
		{name: "onedrive", version: "26.139.0720.0007", slug: "onedrive", expected: "26.139.0720"},
		{name: "harmony sase", version: "12.11.0.12314", slug: "harmony-sase", expected: "12.11.0"},
		{name: "expressvpn", version: "14.3.1.15429", slug: "expressvpn", expected: "14.3.1"},
		{name: "cmux hotfix", version: "0.65.2.2", slug: "cmux", expected: "0.65.2"},
		{name: "cmux release", version: "0.65.2", slug: "cmux", expected: "0.65.2"},
		{name: "megasync", version: "6.7.0.3", slug: "megasync", expected: "6.7.0"},
		{name: "capcut", version: "9.5.0.4590", slug: "capcut", expected: "9.5.0"},
		{name: "shapr3d keeps a trailing zero", version: "26.202.0.12281", slug: "shapr3d", expected: "26.202.0"},
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

func TestSmallstepAgentVersionTransformer(t *testing.T) {
	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "numeric version", version: "0.68.0", expected: "v0.68.0"},
		{name: "already prefixed", version: "v0.68.0", expected: "v0.68.0"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "smallstepagent"}
			result, err := SmallstepAgentVersionTransformer(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestRaspberryPiImagerVersionTransformer(t *testing.T) {
	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "numeric version", version: "2.0.11.1", expected: "v2.0.11.1"},
		{name: "already prefixed", version: "v2.0.11.1", expected: "v2.0.11.1"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "raspberry-pi-imager"}
			result, err := RaspberryPiImagerVersionTransformer(app)
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

func TestShottrVersionTransformer(t *testing.T) {
	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "letter suffix", version: "1.9.3b", expected: "1.9.3"},
		{name: "no suffix", version: "1.9.2", expected: "1.9.2"},
		{name: "two segments", version: "1.9", expected: "1.9"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "shottr"}
			result, err := ShottrVersionTransformer(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestMakeTrailingZeroTrimmer(t *testing.T) {
	trimmer := makeTrailingZeroTrimmer(2)

	tcs := []struct {
		name     string
		version  string
		expected string
		wantErr  bool
	}{
		{name: "empty version", version: "", wantErr: true},
		{name: "trailing zero", version: "4.6.0", expected: "4.6"},
		{name: "nonzero patch", version: "4.5.1", expected: "4.5.1"},
		{name: "keeps min segments", version: "1.0.0", expected: "1.0"},
		{name: "interior zero", version: "2.0.9", expected: "2.0.9"},
		{name: "several trailing zeros", version: "3.1.0.0", expected: "3.1"},
		{name: "fewer segments than min", version: "7", expected: "7"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "test-app"}
			result, err := trimmer(app)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Version)
		})
	}
}

func TestAdGuardVersion(t *testing.T) {
	// AdGuard's bundle_short_version drops the build and, for x.y.0 releases,
	// the trailing zero.
	tcs := []struct {
		version  string
		expected string
	}{
		{version: "2.20.2.2629", expected: "2.20.2"},
		{version: "2.20.0.2626", expected: "2.20"},
	}

	for _, tc := range tcs {
		t.Run(tc.version, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: "adguard/darwin"}
			EnrichManifest(app)
			assert.Equal(t, tc.expected, app.Version)
		})
	}
}

func TestBundleVersionAppsShortenToShortVersion(t *testing.T) {
	// Each expected value is the app's CFBundleShortVersionString for that cask
	// version, which is what host inventory reports.
	tcs := []struct {
		slug     string
		version  string
		expected string
	}{
		{slug: "sigmaos/darwin", version: "1.19.0.4", expected: "1.19.0"},
		{slug: "malwarebytes/darwin", version: "5.29.2.4310", expected: "5.29.2"},
		{slug: "malwarebytes/darwin", version: "5.28.4.4244", expected: "5.28.4"},
		{slug: "plex-media-server/darwin", version: "1.43.5.11079", expected: "1.43.5"},
		{slug: "chrome-remote-desktop-host/darwin", version: "157.0.8103.38", expected: "157.0.8103"},
		{slug: "google-earth-pro/darwin", version: "7.3.7.1327", expected: "7.3"},
		{slug: "disk-drill/darwin", version: "6.3.2326", expected: "6.3"},
		{slug: "kdenlive/darwin", version: "26.08.3", expected: "26.08"},
		{slug: "syncmate/darwin", version: "8.11.604", expected: "8.11"},
		{slug: "redcine-x-pro/darwin", version: "65.2.1", expected: "65.2"},
		{slug: "redcine-x-pro/darwin", version: "65.1.3", expected: "65.1"},
		{slug: "bitrix24/darwin", version: "24.0.37.92", expected: "24.0.37"},
		{slug: "4k-slideshow-maker/darwin", version: "2.0.1", expected: "2.0"},
		{slug: "4k-stogram/darwin", version: "4.9.0", expected: "4.9"},
		{slug: "4k-video-downloader/darwin", version: "4.33.5", expected: "4.33"},
		{slug: "4k-video-to-mp3/darwin", version: "3.0.1", expected: "3.0"},
		{slug: "4k-youtube-to-mp3/darwin", version: "26.3.5", expected: "26.3"},
	}

	for _, tc := range tcs {
		t.Run(tc.slug+" "+tc.version, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tc.version, Slug: tc.slug}
			EnrichManifest(app)
			assert.Equal(t, tc.expected, app.Version)
		})
	}
}
