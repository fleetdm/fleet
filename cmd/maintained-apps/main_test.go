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

// existingAppsList is written in compact form so that any rewrite of the file is detectable: the
// encoder in updateAppsListFile always emits indented JSON with a trailing newline.
const existingAppsList = `{"version":2,"apps":[` +
	`{"name":"Docker Desktop","slug":"docker-desktop/darwin","platform":"darwin","unique_identifier":"com.docker.docker","description":"Containers & microservices — <b>fast</b>."},` +
	`{"name":"Zoom","slug":"zoom/darwin","platform":"darwin","unique_identifier":"us.zoom.xos","description":"Zoom is a video conferencing tool."}` +
	`]}`

// setupAppsListFile makes a temp directory the working directory and seeds it with an apps.json
// holding the given contents, since updateAppsListFile resolves its path relative to the CWD.
func setupAppsListFile(t *testing.T, contents string) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(path.Join(root, maintained_apps.OutputPath), 0o755))
	t.Chdir(root)

	appListFilePath := path.Join(maintained_apps.OutputPath, "apps.json")
	require.NoError(t, os.WriteFile(appListFilePath, []byte(contents), 0o644))

	return appListFilePath
}

func readAppsListFile(t *testing.T, appListFilePath string) (maintained_apps.FMAListFile, string) {
	t.Helper()

	rawBytes, err := os.ReadFile(appListFilePath)
	require.NoError(t, err)

	var appsList maintained_apps.FMAListFile
	require.NoError(t, json.Unmarshal(rawBytes, &appsList))

	return appsList, string(rawBytes)
}

// requireAppsListNotWritten asserts apps.json was left alone. The seeded file is compact, while
// every write from updateAppsListFile emits indented JSON, so the absence of newlines means no
// rewrite happened. (A JSON-equality comparison can't show this: a rewrite is semantically equal.)
func requireAppsListNotWritten(t *testing.T, appListFilePath string) {
	t.Helper()

	rawBytes, err := os.ReadFile(appListFilePath)
	require.NoError(t, err)
	require.NotContains(t, string(rawBytes), "\n")
	require.Len(t, rawBytes, len(existingAppsList))
}

func TestUpdateAppsListFile(t *testing.T) {
	// Cannot run t.Parallel(): these subtests t.Chdir.

	t.Run("new app is appended", func(t *testing.T) {
		appListFilePath := setupAppsListFile(t, existingAppsList)

		err := updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
			Name:             "Firefox",
			Slug:             "firefox/darwin",
			UniqueIdentifier: "org.mozilla.firefox",
		})
		require.NoError(t, err)

		appsList, raw := readAppsListFile(t, appListFilePath)
		require.Equal(t, 2, appsList.Version)
		require.Len(t, appsList.Apps, 3)

		// Sorted by slug.
		require.Equal(t, []string{"docker-desktop/darwin", "firefox/darwin", "zoom/darwin"}, []string{
			appsList.Apps[0].Slug, appsList.Apps[1].Slug, appsList.Apps[2].Slug,
		})
		require.Equal(t, maintained_apps.FMAListFileApp{
			Name:             "Firefox",
			Slug:             "firefox/darwin",
			Platform:         "darwin",
			UniqueIdentifier: "org.mozilla.firefox",
		}, appsList.Apps[1])

		// Existing entries, including descriptions (which have no manifest counterpart), are
		// untouched, and non-ASCII/HTML characters are not escaped.
		require.Equal(t, "Containers & microservices — <b>fast</b>.", appsList.Apps[0].Description)
		require.Equal(t, "Zoom is a video conferencing tool.", appsList.Apps[2].Description)
		require.Contains(t, raw, "Containers & microservices — <b>fast</b>.")
		require.NotContains(t, raw, `\u`)
	})

	t.Run("existing app with a changed unique_identifier is updated", func(t *testing.T) {
		appListFilePath := setupAppsListFile(t, existingAppsList)

		err := updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
			Name:             "Docker Desktop",
			Slug:             "docker-desktop/darwin",
			UniqueIdentifier: "com.electron.dockerdesktop",
		})
		require.NoError(t, err)

		appsList, _ := readAppsListFile(t, appListFilePath)
		require.Len(t, appsList.Apps, 2)
		require.Equal(t, maintained_apps.FMAListFileApp{
			Name:             "Docker Desktop",
			Slug:             "docker-desktop/darwin",
			Platform:         "darwin",
			UniqueIdentifier: "com.electron.dockerdesktop",
			Description:      "Containers & microservices — <b>fast</b>.",
		}, appsList.Apps[0])
		require.Equal(t, "us.zoom.xos", appsList.Apps[1].UniqueIdentifier)
	})

	t.Run("existing app with a changed name is updated", func(t *testing.T) {
		appListFilePath := setupAppsListFile(t, existingAppsList)

		err := updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
			Name:             "Zoom Workplace",
			Slug:             "zoom/darwin",
			UniqueIdentifier: "us.zoom.xos",
		})
		require.NoError(t, err)

		appsList, _ := readAppsListFile(t, appListFilePath)
		require.Len(t, appsList.Apps, 2)
		require.Equal(t, "Zoom Workplace", appsList.Apps[1].Name)
		require.Equal(t, "us.zoom.xos", appsList.Apps[1].UniqueIdentifier)
	})

	t.Run("unchanged app is not written", func(t *testing.T) {
		appListFilePath := setupAppsListFile(t, existingAppsList)

		err := updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
			Name:             "Zoom",
			Slug:             "zoom/darwin",
			UniqueIdentifier: "us.zoom.xos",
		})
		require.NoError(t, err)

		requireAppsListNotWritten(t, appListFilePath)
	})

	t.Run("slug without a platform is rejected", func(t *testing.T) {
		appListFilePath := setupAppsListFile(t, existingAppsList)

		err := updateAppsListFile(t.Context(), &maintained_apps.FMAManifestApp{
			Name:             "Firefox",
			Slug:             "firefox",
			UniqueIdentifier: "org.mozilla.firefox",
		})
		require.ErrorContains(t, err, "invalid platform found for slug firefox")

		requireAppsListNotWritten(t, appListFilePath)
	})
}
