package maintained_apps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNoUnexpectedSharedDarwinIdentifiers fails when two unrelated macOS FMAs share a
// bundle identifier. Shared identifiers are ambiguous and are only handled by
// ReconcileMaintainedAppSoftwareNames and fleetMaintainedAppsTeamJoin for variants of
// the same app (see https://github.com/fleetdm/fleet/issues/42445). Variants are
// expressed with an "@" suffix on the slug's app token (firefox / firefox@esr,
// druva-insync / druva-insync@govcloud), so apps whose slugs share that base token may
// share an identifier; anything else is a collision to fix.
func TestNoUnexpectedSharedDarwinIdentifiers(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	b, err := os.ReadFile(filepath.Join(base, "ee/maintained-apps/outputs/apps.json"))
	require.NoError(t, err)

	var appsList AppsList
	require.NoError(t, json.Unmarshal(b, &appsList))

	for identifier, slugs := range unrelatedSharedDarwinIdentifiers(appsList.Apps) {
		require.Failf(t, "shared macOS bundle identifier",
			"macOS bundle identifier %q is shared by unrelated Fleet-maintained apps %v.\n"+
				"Only variants of the same app (slugs that differ by an \"@<variant>\" suffix) may share a "+
				"bundle identifier; otherwise matching installed software to an FMA is ambiguous.",
			identifier, slugs)
	}
}

func TestUnrelatedSharedDarwinIdentifiers(t *testing.T) {
	apps := []appListing{
		{Slug: "firefox/darwin", Platform: "darwin", UniqueIdentifier: "org.mozilla.firefox"},
		{Slug: "firefox@esr/darwin", Platform: "darwin", UniqueIdentifier: "org.mozilla.firefox"},
		{Slug: "druva-insync/darwin", Platform: "darwin", UniqueIdentifier: "com.druva.inSyncClient"},
		{Slug: "druva-insync@govcloud/darwin", Platform: "darwin", UniqueIdentifier: "com.druva.inSyncClient"},
		// Windows DisplayName collisions are expected and out of scope.
		{Slug: "amazon-corretto-21/windows", Platform: "windows", UniqueIdentifier: "Amazon Corretto (x64)"},
		{Slug: "amazon-corretto-25/windows", Platform: "windows", UniqueIdentifier: "Amazon Corretto (x64)"},
		{Slug: "solo/darwin", Platform: "darwin", UniqueIdentifier: "com.example.solo"},
	}
	require.Empty(t, unrelatedSharedDarwinIdentifiers(apps))

	apps = append(apps, appListing{Slug: "other-browser/darwin", Platform: "darwin", UniqueIdentifier: "org.mozilla.firefox"})
	require.Equal(t, map[string][]string{
		"org.mozilla.firefox": {"firefox/darwin", "firefox@esr/darwin", "other-browser/darwin"},
	}, unrelatedSharedDarwinIdentifiers(apps))
}

// unrelatedSharedDarwinIdentifiers returns, per macOS bundle identifier, the sorted
// slugs of apps sharing it when those apps are not variants of a single app.
func unrelatedSharedDarwinIdentifiers(apps []appListing) map[string][]string {
	slugsByIdentifier := make(map[string][]string)
	for _, app := range apps {
		if app.Platform != "darwin" || app.UniqueIdentifier == "" {
			continue
		}
		slugsByIdentifier[app.UniqueIdentifier] = append(slugsByIdentifier[app.UniqueIdentifier], app.Slug)
	}

	violations := make(map[string][]string)
	for identifier, slugs := range slugsByIdentifier {
		if len(slugs) <= 1 {
			continue
		}
		bases := make(map[string]struct{})
		for _, slug := range slugs {
			bases[slugAppBase(slug)] = struct{}{}
		}
		if len(bases) > 1 {
			sort.Strings(slugs)
			violations[identifier] = slugs
		}
	}
	return violations
}

// slugAppBase returns the app token of a slug ("<app>[@<variant>]/<platform>") without
// its variant suffix, e.g. "firefox@esr/darwin" -> "firefox".
func slugAppBase(slug string) string {
	app, _, _ := strings.Cut(slug, "/")
	app, _, _ = strings.Cut(app, "@")
	return app
}

func TestSlugAppBase(t *testing.T) {
	for slug, want := range map[string]string{
		"firefox/darwin":               "firefox",
		"firefox@esr/darwin":           "firefox",
		"druva-insync@govcloud/darwin": "druva-insync",
		"affinity-photo@1/darwin":      "affinity-photo",
		"1password":                    "1password",
	} {
		require.Equal(t, want, slugAppBase(slug), slug)
	}
}
