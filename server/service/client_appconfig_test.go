package service

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func makeJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func writeTempFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, body, 0o600))
	return path
}

func TestValidateOrgLogoFile(t *testing.T) {
	t.Parallel()

	t.Run("accepts png", func(t *testing.T) {
		assert.NoError(t, validateOrgLogoFile(writeTempFile(t, "logo.png", makePNG(t))))
	})
	t.Run("accepts jpeg", func(t *testing.T) {
		assert.NoError(t, validateOrgLogoFile(writeTempFile(t, "logo.jpg", makeJPEG(t))))
	})
	t.Run("accepts svg", func(t *testing.T) {
		svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"></svg>`)
		assert.NoError(t, validateOrgLogoFile(writeTempFile(t, "logo.svg", svg)))
	})
	t.Run("rejects unknown format", func(t *testing.T) {
		err := validateOrgLogoFile(writeTempFile(t, "logo.txt", []byte("not an image")))
		require.Error(t, err)
		assert.ErrorContains(t, err, "PNG, JPEG, WebP, or SVG")
	})
	t.Run("rejects oversized file", func(t *testing.T) {
		// fleet.ValidateOrgLogoBytes fires its size check before
		// image.DecodeConfig, so the body content doesn't need to
		// decode as a real image.
		body := make([]byte, orgLogoMaxFileSize+1)
		err := validateOrgLogoFile(writeTempFile(t, "big.png", body))
		require.Error(t, err)
		assert.ErrorContains(t, err, "100KB or less")
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
	require.NoError(t, os.WriteFile(pngPath, makePNG(t), 0o600))

	orgSettings := func(orgInfo map[string]any) map[string]any {
		return map[string]any{"org_info": orgInfo}
	}

	t.Run("path key plans upload and strips every URL key for the mode", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
			"org_logo_url_dark_mode":  "",
		})
		actions, err := c.planAndStripOrgLogos(os, dir, false, logFn)
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

	t.Run("external URL set rides on the PATCH and mirrors deprecated alias", func(t *testing.T) {
		// Switching path → URL must preserve the new URL.
		os := orgSettings(map[string]any{
			"org_logo_url_dark_mode": "https://example.com/logo.png",
		})
		actions, err := c.planAndStripOrgLogos(os, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions, "URL changes ride on the PATCH; no follow-up action")

		orgInfo := os["org_info"].(map[string]any)
		assert.Equal(t, "https://example.com/logo.png", orgInfo["org_logo_url_dark_mode"])
		assert.Equal(t, "https://example.com/logo.png", orgInfo["org_logo_url"])
	})

	t.Run("explicit empty URL mirrors deprecated alias as empty", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_url_light_mode": "",
		})
		actions, err := c.planAndStripOrgLogos(os, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions)

		// Both new and deprecated keys must be sent as "" — otherwise the
		// server preserves the deprecated field on merge and copies it
		// back into the new one in NormalizeLogoFields.
		orgInfo := os["org_info"].(map[string]any)
		assert.Empty(t, orgInfo["org_logo_url_light_mode"])
		assert.Empty(t, orgInfo["org_logo_url_light_background"])
	})

	t.Run("clearing new URL keeps the deprecated alias in sync", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_url_dark_mode":  "",
			"org_logo_url_light_mode": "",
		})
		actions, err := c.planAndStripOrgLogos(os, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions)

		orgInfo := os["org_info"].(map[string]any)
		assert.Empty(t, orgInfo["org_logo_url_dark_mode"])
		assert.Empty(t, orgInfo["org_logo_url"], "deprecated dark alias must be sent as \"\"")
		assert.Empty(t, orgInfo["org_logo_url_light_mode"])
		assert.Empty(t, orgInfo["org_logo_url_light_background"], "deprecated light alias must be sent as \"\"")
	})

	t.Run("missing keys preserve current state", func(t *testing.T) {
		os := orgSettings(map[string]any{"org_name": "ACME"})
		actions, err := c.planAndStripOrgLogos(os, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions, "absent keys must not trigger any action")
	})

	t.Run("both path and url for same mode rejected", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
			"org_logo_url_dark_mode":  "https://example.com/logo.png",
		})
		_, err := c.planAndStripOrgLogos(os, dir, false, logFn)
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot specify both")
	})

	t.Run("missing org_info is no-op", func(t *testing.T) {
		actions, err := c.planAndStripOrgLogos(map[string]any{}, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions)
	})

	t.Run("both modes set are processed independently", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
			"org_logo_url_light_mode": "https://example.com/light.png",
		})
		actions, err := c.planAndStripOrgLogos(os, dir, false, logFn)
		require.NoError(t, err)
		require.Len(t, actions, 1, "only the path-key mode queues an action; URL changes ride on the PATCH")
		assert.Equal(t, fleet.OrgLogoModeDark, actions[0].mode)
		assert.NotEmpty(t, actions[0].uploadPath, "dark mode should plan an upload")

		orgInfo := os["org_info"].(map[string]any)
		// Dark: every URL key for the mode is stripped (PUT will set them).
		for _, k := range []string{"org_logo_path_dark_mode", "org_logo_url_dark_mode", "org_logo_url"} {
			_, present := orgInfo[k]
			assert.False(t, present, "%s should be stripped", k)
		}
		// Light: URL ride on the PATCH with the deprecated alias mirrored.
		assert.Equal(t, "https://example.com/light.png", orgInfo["org_logo_url_light_mode"])
		assert.Equal(t, "https://example.com/light.png", orgInfo["org_logo_url_light_background"])
	})

	t.Run("missing path file surfaces a validation error", func(t *testing.T) {
		os := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "does-not-exist.png",
		})
		_, err := c.planAndStripOrgLogos(os, dir, false, logFn)
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
		_, err := c.planAndStripOrgLogos(settings, dir, false, logFn)
		require.Error(t, err)
		assert.ErrorContains(t, err, "PNG, JPEG, WebP, or SVG")
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
		_, err := c.planAndStripOrgLogos(osBad, dir, true, captureLog)
		require.Error(t, err)

		// Valid file should plan an upload and log the would-upload line.
		osGood := orgSettings(map[string]any{
			"org_logo_path_dark_mode": "logo.png",
		})
		actions, err := c.planAndStripOrgLogos(osGood, dir, true, captureLog)
		require.NoError(t, err)
		require.Len(t, actions, 1)
		require.NotEmpty(t, logs)
		joined := strings.Join(logs, "\n")
		assert.Contains(t, joined, "would upload org logo (dark)")
	})
}
