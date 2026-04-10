package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
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

func TestUpdateAppsListFileAt_UpdatesMetadataPreservesDescription(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	listPath := filepath.Join(dir, "apps.json")
	initial := maintained_apps.FMAListFile{
		Version: 2,
		Apps: []maintained_apps.FMAListFileApp{
			{
				Name:             "Old Name",
				Slug:             "my-app/darwin",
				Platform:         "darwin",
				UniqueIdentifier: "com.old.id",
				Description:      "Hand-written blurb.",
			},
		},
	}
	if err := writeAppsListJSON(listPath, &initial); err != nil {
		t.Fatal(err)
	}

	app := &maintained_apps.FMAManifestApp{
		Slug:             "my-app/darwin",
		Name:             "New Name",
		UniqueIdentifier: "com.new.id",
		Version:          "1", // non-empty so IsEmpty is false if reused elsewhere
		InstallerURL:     "https://example.com/x",
		InstallScriptRef: "a",
		UninstallScriptRef: "b",
		SHA256:           "c",
		Queries:          maintained_apps.FMAQueries{Exists: "SELECT 1"},
	}
	if err := updateAppsListFileAt(ctx, listPath, app); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(listPath)
	if err != nil {
		t.Fatal(err)
	}
	var got maintained_apps.FMAListFile
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Apps) != 1 {
		t.Fatalf("apps: %d", len(got.Apps))
	}
	row := got.Apps[0]
	if row.Name != "New Name" || row.UniqueIdentifier != "com.new.id" || row.Description != "Hand-written blurb." {
		t.Fatalf("got %+v", row)
	}
}

func TestUpdateAppsListFileAt_NoWriteWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	listPath := filepath.Join(dir, "apps.json")
	initial := maintained_apps.FMAListFile{
		Version: 2,
		Apps: []maintained_apps.FMAListFileApp{
			{
				Name:             "Same",
				Slug:             "my-app/darwin",
				Platform:         "darwin",
				UniqueIdentifier: "com.same",
				Description:      "D",
			},
		},
	}
	if err := writeAppsListJSON(listPath, &initial); err != nil {
		t.Fatal(err)
	}
	stamp, err := os.Stat(listPath)
	if err != nil {
		t.Fatal(err)
	}

	app := &maintained_apps.FMAManifestApp{
		Slug:             "my-app/darwin",
		Name:             "Same",
		UniqueIdentifier: "com.same",
		Version:          "1",
		InstallerURL:     "https://example.com/x",
		InstallScriptRef: "a",
		UninstallScriptRef: "b",
		SHA256:           "c",
		Queries:          maintained_apps.FMAQueries{Exists: "SELECT 1"},
	}
	if err := updateAppsListFileAt(ctx, listPath, app); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(listPath)
	if err != nil {
		t.Fatal(err)
	}
	if after.ModTime() != stamp.ModTime() {
		t.Fatal("expected no write when row unchanged")
	}
}
