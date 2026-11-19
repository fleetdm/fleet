package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/require"
)

func TestJSONEncoderPreservesHTML(t *testing.T) {
	testData := struct {
		Description string `json:"description"`
	}{
		Description: `Test with HTML: <a href="https://example.com">link</a> & special chars < >`,
	}

	// Test with SetEscapeHTML(false)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(testData); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	result := buf.String()

	// Verify HTML characters are preserved, not escaped
	if strings.Contains(result, `\u003c`) {
		t.Error("Found escaped '<' character (\\u003c) - HTML escaping is still enabled")
	}
	if strings.Contains(result, `\u003e`) {
		t.Error("Found escaped '>' character (\\u003e) - HTML escaping is still enabled")
	}
	if strings.Contains(result, `\u0026`) {
		t.Error("Found escaped '&' character (\\u0026) - HTML escaping is still enabled")
	}

	// Verify HTML characters are present (note: quotes inside JSON are still escaped)
	if !strings.Contains(result, `<a href=\"https://example.com\">`) {
		t.Error("HTML anchor tag was not preserved correctly")
	}
	if !strings.Contains(result, ` & `) {
		t.Error("Ampersand character was not preserved correctly")
	}

	t.Logf("Successfully preserved HTML in JSON output: %s", result)
}

func TestUpdateAppsListFileRejectsSameNameOnSamePlatform(t *testing.T) {
	ctx := t.Context()
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(maintained_apps.OutputPath, 0o755))

	appsList := `{"version": 2, "apps": [{"name": "Docker Desktop", "slug": "docker-desktop/darwin", "platform": "darwin"}, {"name": "Zen Browser", "slug": "zen/darwin", "platform": "darwin"}]}`
	require.NoError(t, os.WriteFile(path.Join(maintained_apps.OutputPath, "apps.json"), []byte(appsList), 0o644))

	// add a macOS app with the same name, the ingester should return an error
	err := updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{Name: "Docker Desktop", Slug: "docker/darwin"})
	require.ErrorContains(t, err, "docker-desktop/darwin already uses on the same platform")

	// add a Windows app with the same name, the ingester should add it
	err = updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{Name: "Docker Desktop", Slug: "docker/windows"})
	require.NoError(t, err)

	// refresh an existing macOS app whose incoming name matches another app, the ingester should
	// not return an error because the name of an existing app isn't updated
	err = updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{Name: "Docker Desktop", Slug: "zen/darwin"})
	require.NoError(t, err)
}
