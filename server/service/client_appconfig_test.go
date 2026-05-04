package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pngBytes  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x01, 0x02}
	jpegBytes = []byte{0xFF, 0xD8, 0xFF, 0x00, 0x01, 0x02, 0x03, 0x04}
	webpBytes = []byte("RIFF\x10\x00\x00\x00WEBPVP8 ")
)

func writeTempFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, body, 0o600))
	return path
}

func TestValidateOrgLogoFile(t *testing.T) {
	t.Parallel()

	t.Run("accepts png", func(t *testing.T) {
		assert.NoError(t, validateOrgLogoFile(writeTempFile(t, "logo.png", pngBytes)))
	})
	t.Run("accepts jpeg", func(t *testing.T) {
		assert.NoError(t, validateOrgLogoFile(writeTempFile(t, "logo.jpg", jpegBytes)))
	})
	t.Run("accepts webp", func(t *testing.T) {
		assert.NoError(t, validateOrgLogoFile(writeTempFile(t, "logo.webp", webpBytes)))
	})
	t.Run("rejects unknown format", func(t *testing.T) {
		err := validateOrgLogoFile(writeTempFile(t, "logo.txt", []byte("not an image")))
		require.Error(t, err)
		assert.ErrorContains(t, err, "PNG, JPEG, or WebP")
	})
	t.Run("rejects oversized file", func(t *testing.T) {
		body := make([]byte, orgLogoMaxFileSizeBytes+1)
		copy(body, pngBytes)
		err := validateOrgLogoFile(writeTempFile(t, "big.png", body))
		require.Error(t, err)
		assert.ErrorContains(t, err, "max allowed")
	})
	t.Run("missing file", func(t *testing.T) {
		err := validateOrgLogoFile(filepath.Join(t.TempDir(), "absent.png"))
		require.Error(t, err)
	})
}

func TestPlanAndStripOrgLogos(t *testing.T) {
	t.Parallel()

	c := &Client{}
	logFn := func(string, ...any) {}
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "logo.png")
	require.NoError(t, os.WriteFile(pngPath, pngBytes, 0o600))

	orgSettings := func(orgInfo map[string]any) map[string]any {
		return map[string]any{"org_info": orgInfo}
	}

	t.Run("path key plans upload and strips every URL key for the mode", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
			"org_logo_url_dark_mode":  "",
		})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{}, dir, false, logFn)
		require.NoError(t, err)
		require.Len(t, actions, 1)
		assert.Equal(t, fleet.OrgLogoModeDark, actions[0].mode)
		assert.NotEmpty(t, actions[0].uploadPath)

		orgInfo := os["org_info"].(map[string]any)
		for _, k := range []string{"org_logo_path_dark_mode", "org_logo_url_dark_mode", "org_logo_url"} {
			_, present := orgInfo[k]
			assert.False(t, present, "%s should be stripped (PUT controls the stored URLs)", k)
		}
	})

	t.Run("external URL with current Fleet-hosted blob plans delete and mirrors deprecated alias", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_url_dark_mode": "https://example.com/logo.png",
		})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{
			OrgLogoURLDarkMode: "https://fleet.example.com/api/latest/fleet/logo?mode=dark",
		}, dir, false, logFn)
		require.NoError(t, err)
		require.Len(t, actions, 1)
		assert.Equal(t, fleet.OrgLogoModeDark, actions[0].mode)
		assert.Empty(t, actions[0].uploadPath, "empty uploadPath signals delete")

		// URL key kept so PATCH writes the external URL, and the deprecated
		// alias is mirrored so server-side NormalizeLogoFields can't undo it.
		orgInfo := os["org_info"].(map[string]any)
		assert.Equal(t, "https://example.com/logo.png", orgInfo["org_logo_url_dark_mode"])
		assert.Equal(t, "https://example.com/logo.png", orgInfo["org_logo_url"])
	})

	t.Run("explicit empty URL with Fleet-hosted blob plans delete and mirrors deprecated alias as empty", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_url_light_mode": "",
		})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{
			OrgLogoURLLightMode: "/api/latest/fleet/logo?mode=light",
		}, dir, false, logFn)
		require.NoError(t, err)
		require.Len(t, actions, 1)
		assert.Equal(t, fleet.OrgLogoModeLight, actions[0].mode)
		assert.Empty(t, actions[0].uploadPath)

		// Both new and deprecated keys must be sent as "" — otherwise the
		// server preserves the deprecated field on merge and copies it back
		// into the new one in NormalizeLogoFields.
		orgInfo := os["org_info"].(map[string]any)
		assert.Empty(t, orgInfo["org_logo_url_light_mode"])
		assert.Empty(t, orgInfo["org_logo_url_light_background"])
	})

	t.Run("clearing new URL keeps the deprecated alias in sync", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_url_dark_mode":  "",
			"org_logo_url_light_mode": "",
		})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{
			OrgLogoURLDarkMode:        "https://customer.example.com/dark.png",
			OrgLogoURL:                "https://customer.example.com/dark.png",
			OrgLogoURLLightMode:       "https://customer.example.com/light.png",
			OrgLogoURLLightBackground: "https://customer.example.com/light.png",
		}, dir, false, logFn)
		require.NoError(t, err)
		// Current URLs aren't Fleet-hosted, so no DELETE actions queued.
		assert.Empty(t, actions)

		orgInfo := os["org_info"].(map[string]any)
		assert.Empty(t, orgInfo["org_logo_url_dark_mode"])
		assert.Empty(t, orgInfo["org_logo_url"], "deprecated dark alias must be sent as \"\"")
		assert.Empty(t, orgInfo["org_logo_url_light_mode"])
		assert.Empty(t, orgInfo["org_logo_url_light_background"], "deprecated light alias must be sent as \"\"")
	})

	t.Run("missing keys preserve current state", func(t *testing.T) {
		os := orgSettings(map[string]any{"org_name": "ACME"})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{
			OrgLogoURLDarkMode: "/api/latest/fleet/logo?mode=dark",
		}, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions, "absent keys must not trigger any action")
	})

	t.Run("both path and url for same mode rejected", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
			"org_logo_url_dark_mode":  "https://example.com/logo.png",
		})
		_, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{}, dir, false, logFn)
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot specify both")
	})

	t.Run("missing org_info is no-op", func(t *testing.T) {
		actions, err := c.planAndStripOrgLogos(map[string]any{}, &fleet.OrgInfo{}, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions)
	})

	t.Run("both modes set are processed independently", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
			"org_logo_url_light_mode": "https://example.com/light.png",
		})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{
			OrgLogoURLLightMode: "/api/latest/fleet/logo?mode=light", // current is Fleet-hosted
		}, dir, false, logFn)
		require.NoError(t, err)
		require.Len(t, actions, 2)

		byMode := map[fleet.OrgLogoMode]orgLogoAction{}
		for _, a := range actions {
			byMode[a.mode] = a
		}
		// Dark: path → upload action.
		darkAct, ok := byMode[fleet.OrgLogoModeDark]
		require.True(t, ok)
		assert.NotEmpty(t, darkAct.uploadPath, "dark mode should plan an upload")
		// Light: external URL replacing a Fleet-hosted blob → delete action.
		lightAct, ok := byMode[fleet.OrgLogoModeLight]
		require.True(t, ok)
		assert.Empty(t, lightAct.uploadPath, "light mode should plan a delete")

		orgInfo := os["org_info"].(map[string]any)
		// Dark: every URL key for the mode is stripped (PUT will set them).
		for _, k := range []string{"org_logo_path_dark_mode", "org_logo_url_dark_mode", "org_logo_url"} {
			_, present := orgInfo[k]
			assert.False(t, present, "%s should be stripped", k)
		}
		// Light: URL key kept so PATCH writes the external URL, and the
		// deprecated alias is mirrored to keep the server's
		// NormalizeLogoFields a no-op.
		assert.Equal(t, "https://example.com/light.png", orgInfo["org_logo_url_light_mode"])
		assert.Equal(t, "https://example.com/light.png", orgInfo["org_logo_url_light_background"])
	})

	t.Run("missing path file surfaces a validation error", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "does-not-exist.png",
		})
		_, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{}, dir, false, logFn)
		require.Error(t, err)
		require.ErrorContains(t, err, "dark")
		require.ErrorContains(t, err, "does-not-exist.png")
	})

	t.Run("invalid file format surfaces a validation error", func(t *testing.T) {
		badPath := filepath.Join(dir, "bad.png")
		require.NoError(t, os.WriteFile(badPath, []byte("not an image"), 0o600))
		settings := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "bad.png",
		})
		_, err := c.planAndStripOrgLogos(settings, &fleet.OrgInfo{}, dir, false, logFn)
		require.Error(t, err)
		assert.ErrorContains(t, err, "PNG, JPEG, or WebP")
	})

	t.Run("dry run still validates and logs would-upload", func(t *testing.T) {
		var logs []string
		captureLog := func(format string, args ...any) {
			logs = append(logs, fmt.Sprintf(format, args...))
		}

		// Bad file should error in dry-run.
		osBad := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "does-not-exist.png",
		})
		_, err := c.planAndStripOrgLogos(osBad, &fleet.OrgInfo{}, dir, true, captureLog)
		require.Error(t, err)

		// Valid file should plan an upload and log the would-upload line.
		osGood := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
		})
		actions, err := c.planAndStripOrgLogos(osGood, &fleet.OrgInfo{}, dir, true, captureLog)
		require.NoError(t, err)
		require.Len(t, actions, 1)
		require.NotEmpty(t, logs)
		joined := strings.Join(logs, "\n")
		assert.Contains(t, joined, "would upload org logo (dark)")
	})
}
