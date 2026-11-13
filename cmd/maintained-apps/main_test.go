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

func TestUpdateAppsListFileExactNameMatch(t *testing.T) {
	ctx := t.Context()
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(maintained_apps.OutputPath, 0o755))
	appsPath := path.Join(maintained_apps.OutputPath, "apps.json")

	write := func(f maintained_apps.FMAListFile) {
		b, err := json.MarshalIndent(f, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(appsPath, append(b, '\n'), 0o644))
	}
	read := func() map[string]maintained_apps.FMAListFileApp {
		b, err := os.ReadFile(appsPath)
		require.NoError(t, err)
		var f maintained_apps.FMAListFile
		require.NoError(t, json.Unmarshal(b, &f))
		bySlug := make(map[string]maintained_apps.FMAListFileApp)
		for _, a := range f.Apps {
			bySlug[a.Slug] = a
		}
		return bySlug
	}

	write(maintained_apps.FMAListFile{Version: 2, Apps: []maintained_apps.FMAListFileApp{
		{Name: "Microsoft Teams", Slug: "microsoft-teams/windows", Platform: "windows", UniqueIdentifier: "Microsoft Teams", Description: "Hand-written."},
		{Name: "Granola", Slug: "granola/windows", Platform: "windows", UniqueIdentifier: "Granola", ExactNameMatch: true},
	}})

	// The flag follows the input on existing entries, and nothing else changes.
	require.NoError(t, updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{
		Name: "Renamed", Slug: "microsoft-teams/windows", UniqueIdentifier: "Microsoft Teams", ExactNameMatch: true,
	}))
	require.NoError(t, updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{
		Name: "Granola", Slug: "granola/windows", UniqueIdentifier: "Granola",
	}))
	apps := read()
	require.True(t, apps["microsoft-teams/windows"].ExactNameMatch)
	require.Equal(t, "Microsoft Teams", apps["microsoft-teams/windows"].Name)
	require.Equal(t, "Hand-written.", apps["microsoft-teams/windows"].Description)
	require.False(t, apps["granola/windows"].ExactNameMatch)

	// A new app is added with its flag.
	require.NoError(t, updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{
		Name: "Slack", Slug: "slack/windows", UniqueIdentifier: "Slack", ExactNameMatch: true,
	}))
	apps = read()
	require.Len(t, apps, 3)
	require.True(t, apps["slack/windows"].ExactNameMatch)

	// An unchanged app leaves the file as it is.
	before, err := os.ReadFile(appsPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(appsPath, append(before, '\n'), 0o644))
	require.NoError(t, updateAppsListFile(ctx, &maintained_apps.FMAManifestApp{
		Name: "Slack", Slug: "slack/windows", UniqueIdentifier: "Slack", ExactNameMatch: true,
	}))
	after, err := os.ReadFile(appsPath)
	require.NoError(t, err)
	require.Equal(t, append(before, '\n'), after)
}
