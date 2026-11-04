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

func TestUpdateAppsListFile(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(maintained_apps.OutputPath, 0o755))
	appsFilePath := path.Join(maintained_apps.OutputPath, "apps.json")

	// Compact JSON, so a rewrite of the file is detectable.
	original := `{"version":2,"apps":[` +
		`{"name":"blip","slug":"blip/darwin","platform":"darwin","unique_identifier":"net.blip.macos","description":"Blip sends files."},` +
		`{"name":"Postman","slug":"postman/windows","platform":"windows","unique_identifier":"Postman x64 11.75.4","description":"Postman is an API platform."},` +
		`{"name":"Zoom","slug":"zoom/darwin","platform":"darwin","unique_identifier":"us.zoom.xos","description":"Zoom is for meetings."}` +
		`]}`
	require.NoError(t, os.WriteFile(appsFilePath, []byte(original), 0o644))

	// An entry that already matches its input leaves the file untouched.
	require.NoError(t, updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
		Name: "Zoom", Slug: "zoom/darwin", UniqueIdentifier: "us.zoom.xos",
	}))
	b, err := os.ReadFile(appsFilePath)
	require.NoError(t, err)
	require.Equal(t, original, string(b))

	// Existing entries pick up name and identifier changes from their inputs, in place.
	require.NoError(t, updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
		Name: "Blip", Slug: "blip/darwin", UniqueIdentifier: "net.blip.macos",
	}))
	require.NoError(t, updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
		Name: "Postman", Slug: "postman/windows", UniqueIdentifier: "Postman x64",
	}))

	// New entries are inserted in slug order.
	require.NoError(t, updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
		Name: "Notion", Slug: "notion/windows", UniqueIdentifier: "Notion",
	}))

	b, err = os.ReadFile(appsFilePath)
	require.NoError(t, err)
	var got maintained_apps.FMAListFile
	require.NoError(t, json.Unmarshal(b, &got))
	require.Equal(t, maintained_apps.FMAListFile{
		Version: 2,
		Apps: []maintained_apps.FMAListFileApp{
			{Name: "Blip", Slug: "blip/darwin", Platform: "darwin", UniqueIdentifier: "net.blip.macos", Description: "Blip sends files."},
			{Name: "Notion", Slug: "notion/windows", Platform: "windows", UniqueIdentifier: "Notion"},
			{Name: "Postman", Slug: "postman/windows", Platform: "windows", UniqueIdentifier: "Postman x64", Description: "Postman is an API platform."},
			{Name: "Zoom", Slug: "zoom/darwin", Platform: "darwin", UniqueIdentifier: "us.zoom.xos", Description: "Zoom is for meetings."},
		},
	}, got)
}
