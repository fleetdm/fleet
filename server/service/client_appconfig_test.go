package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
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

	t.Run("external URL replacing Fleet-hosted blob defers URL write until after DELETE", func(t *testing.T) {
		// Regression test for https://github.com/fleetdm/fleet/pull/45230.
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
		assert.Equal(t, "https://example.com/logo.png", actions[0].replaceWithURL,
			"the new URL rides on the action so doGitOpsOrgLogos can re-PATCH it after the DELETE")

		// URL keys must be stripped from this PATCH — otherwise the
		// follow-up DELETE clobbers them.
		orgInfo := os["org_info"].(map[string]any)
		for _, k := range []string{"org_logo_url_dark_mode", "org_logo_url"} {
			_, present := orgInfo[k]
			assert.False(t, present, "%s should be stripped (DELETE would clobber it; URL is re-PATCHed after)", k)
		}
	})

	t.Run("external URL replacing another external URL plans no delete and PATCHes the URL", func(t *testing.T) {
		// No Fleet-hosted blob to clean up, so the PATCH can flip the URL
		// directly. The deprecated alias is mirrored to keep server-side
		// NormalizeLogoFields a no-op.
		os := orgSettings(map[string]any{
			"org_logo_url_dark_mode": "https://example.com/new.png",
		})
		actions, err := c.planAndStripOrgLogos(os, &fleet.OrgInfo{
			OrgLogoURLDarkMode: "https://example.com/old.png",
			OrgLogoURL:         "https://example.com/old.png",
		}, dir, false, logFn)
		require.NoError(t, err)
		assert.Empty(t, actions, "no Fleet-hosted blob → no DELETE needed")

		orgInfo := os["org_info"].(map[string]any)
		assert.Equal(t, "https://example.com/new.png", orgInfo["org_logo_url_dark_mode"])
		assert.Equal(t, "https://example.com/new.png", orgInfo["org_logo_url"])
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
		// Light: external URL replacing a Fleet-hosted blob → delete action
		// that carries the new URL (re-PATCHed after DELETE, see https://github.com/fleetdm/fleet/pull/45230).
		lightAct, ok := byMode[fleet.OrgLogoModeLight]
		require.True(t, ok)
		assert.Empty(t, lightAct.uploadPath, "light mode should plan a delete")
		assert.Equal(t, "https://example.com/light.png", lightAct.replaceWithURL)

		orgInfo := os["org_info"].(map[string]any)
		// Dark: every URL key for the mode is stripped (PUT will set them).
		for _, k := range []string{"org_logo_path_dark_mode", "org_logo_url_dark_mode", "org_logo_url"} {
			_, present := orgInfo[k]
			assert.False(t, present, "%s should be stripped", k)
		}
		// Light: URL keys also stripped — the follow-up DELETE would
		// clobber any URL written by this PATCH. doGitOpsOrgLogos
		// re-PATCHes the URL after the DELETE.
		for _, k := range []string{"org_logo_url_light_mode", "org_logo_url_light_background"} {
			_, present := orgInfo[k]
			assert.False(t, present, "%s should be stripped (DELETE would clobber it)", k)
		}
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

func TestDoGitOpsOrgLogosReplaceWithURL(t *testing.T) {
	type recorded struct {
		method string
		path   string
		query  string
		body   string
	}
	var calls []recorded

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, recorded{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			body:   string(body),
		})
		// appConfigResponse and deleteOrgLogoResponse are both tolerant of an empty object.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	baseURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	c := &Client{
		baseClient: &baseClient{
			BaseURL: baseURL,
			HTTP:    fleethttp.NewClient(),
		},
		token: "test-token",
	}

	actions := []orgLogoAction{
		{mode: fleet.OrgLogoModeDark, replaceWithURL: "https://example.com/dark.png"},
	}
	var logs []string
	logFn := func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}
	require.NoError(t, c.doGitOpsOrgLogos(actions, false, logFn))

	require.Len(t, calls, 2, "expected DELETE followed by PATCH")

	// 1. DELETE /api/latest/fleet/logo?mode=dark — clears the orphan blob
	//    (and clears URL fields server-side, which is exactly why the
	//    PATCH below has to run after).
	assert.Equal(t, http.MethodDelete, calls[0].method)
	assert.Equal(t, "/api/latest/fleet/logo", calls[0].path)
	assert.Equal(t, "mode=dark", calls[0].query)

	// 2. PATCH /api/latest/fleet/config — writes the new URL. Both keys
	//    must be mirrored to keep server-side NormalizeLogoFields a no-op.
	assert.Equal(t, http.MethodPatch, calls[1].method)
	assert.Equal(t, "/api/latest/fleet/config", calls[1].path)
	var payload struct {
		OrgInfo struct {
			OrgLogoURLDarkMode string `json:"org_logo_url_dark_mode"`
			OrgLogoURL         string `json:"org_logo_url"`
		} `json:"org_info"`
	}
	require.NoError(t, json.Unmarshal([]byte(calls[1].body), &payload))
	assert.Equal(t, "https://example.com/dark.png", payload.OrgInfo.OrgLogoURLDarkMode)
	assert.Equal(t, "https://example.com/dark.png", payload.OrgInfo.OrgLogoURL,
		"deprecated alias must be set to the same URL")

	require.Len(t, logs, 1)
	assert.Contains(t, logs[0], "replaced org logo (dark) with https://example.com/dark.png")
}

func TestDoGitOpsOrgLogosDryRun(t *testing.T) {
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		calls++
	}))
	defer ts.Close()

	baseURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	c := &Client{
		baseClient: &baseClient{
			BaseURL: baseURL,
			HTTP:    fleethttp.NewClient(),
		},
		token: "test-token",
	}

	actions := []orgLogoAction{
		{mode: fleet.OrgLogoModeLight, replaceWithURL: "https://example.com/light.png"},
	}
	var logs []string
	logFn := func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}
	require.NoError(t, c.doGitOpsOrgLogos(actions, true, logFn))

	assert.Equal(t, 0, calls, "dry-run must not hit the server")
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0], "would replace org logo (light) with https://example.com/light.png")
}
