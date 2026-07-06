package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestCheckNoVersionRegression(t *testing.T) {
	ctx := t.Context()

	writeManifest := func(t *testing.T, version string) string {
		t.Helper()
		outFile := maintained_apps.FMAManifestFile{
			Versions: []*maintained_apps.FMAManifestApp{{Version: version}},
		}
		bytes, err := json.Marshal(outFile)
		require.NoError(t, err)
		p := filepath.Join(t.TempDir(), "app.json")
		require.NoError(t, os.WriteFile(p, bytes, 0o644))
		return p
	}

	cases := []struct {
		name      string
		existing  string
		ingested  string
		wantError bool
	}{
		{name: "upgrade is allowed", existing: "1.2.3", ingested: "1.2.4"},
		{name: "same version is allowed", existing: "1.2.3", ingested: "1.2.3"},
		{name: "downgrade is rejected", existing: "1.2.3", ingested: "1.2.2", wantError: true},
		{name: "major downgrade is rejected", existing: "26.001.21691", ingested: "25.001.20630", wantError: true},
		{name: "four-part upgrade is allowed", existing: "16.0.19822.20114", ingested: "16.0.19929.20062"},
		{name: "empty existing version is skipped", existing: "", ingested: "1.0.0"},
		{name: "empty ingested version is skipped", existing: "1.0.0", ingested: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := writeManifest(t, c.existing)
			err := checkNoVersionRegression(ctx, p, &maintained_apps.FMAManifestApp{
				Slug:    "app/windows",
				Version: c.ingested,
			})
			if c.wantError {
				require.ErrorContains(t, err, "version regression")
				return
			}
			require.NoError(t, err)
		})
	}
}
