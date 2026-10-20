package main

import (
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeAppleFixture writes files (relative path → YAML) into a fresh download dir.
func writeAppleFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	return dir
}

func profileYAML(payloadType string, keys ...string) string {
	s := "payload:\n  payloadtype: " + payloadType + "\npayloadkeys:\n"
	for _, k := range keys {
		s += "- key: " + k + "\n  type: <string>\n"
	}
	return s
}

func declarationYAML(declarationType string, keys ...string) string {
	s := "payload:\n  declarationtype: " + declarationType + "\npayloadkeys:\n"
	for _, k := range keys {
		s += "- key: " + k + "\n  type: <string>\n"
	}
	return s
}

// baseFixture is the smallest download loadApple accepts.
func baseFixture() map[string]string {
	return map[string]string{
		"mdm/profiles/TopLevel.yaml":                      profileYAML("TopLevel", "PayloadVersion"),
		"mdm/profiles/CommonPayloadKeys.yaml":             profileYAML("CommonPayloadKeys", "PayloadUUID"),
		"declarative/declarations/declarationbase.yaml":   declarationYAML("any", "Type", "Identifier"),
		"declarative/declarations/configurations/pc.yaml": declarationYAML("com.apple.configuration.passcode.settings", "MinimumLength"),
		"mdm/profiles/com.apple.applicationaccess.yaml":   profileYAML("com.apple.applicationaccess", "allowCamera"),
	}
}

func fileNames(g *appleGroup) []string {
	var names []string
	for _, f := range g.Files {
		names = append(names, f.Path)
	}
	return names
}

func TestLoadAppleGrouping(t *testing.T) {
	files := baseFixture()
	// Several files, one type; keyed by payloadtype, not filename.
	files["mdm/profiles/com.apple.MCX(WiFi).yaml"] = profileYAML("com.apple.MCX", "WiFiKey")
	files["mdm/profiles/com.apple.MCX(Accounts).yaml"] = profileYAML("com.apple.MCX", "AccountsKey", "Shared")
	files["mdm/profiles/com.apple.MCX(EnergySaver).yaml"] = profileYAML("com.apple.MCX", "Shared")
	// Same-looking names, different types.
	files["mdm/profiles/loginwindow.yaml"] = profileYAML("loginwindow", "A")
	files["mdm/profiles/com.apple.loginwindow.yaml"] = profileYAML("com.apple.loginwindow", "B")
	// Not declarations.
	files["declarative/declarations/assets/credentials/acme.yaml"] = "payload:\n  credentialtype: com.apple.credential.acme\n"
	files["declarative/declarations/assets/data.yaml"] = declarationYAML("com.apple.asset.data", "Reference")
	files["mdm/profiles/README.md"] = "not yaml, ignored"

	src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
	require.NoError(t, err)

	assert.Equal(t, []string{"mdm/profiles/TopLevel.yaml"}, fileNames(src.TopLevel))
	assert.Equal(t, []string{"mdm/profiles/CommonPayloadKeys.yaml"}, fileNames(src.CommonPayloadKeys))
	assert.Equal(t, []string{"declarative/declarations/declarationbase.yaml"}, fileNames(src.DeclarationBase))

	assert.ElementsMatch(t, []string{"com.apple.applicationaccess", "com.apple.MCX", "loginwindow", "com.apple.loginwindow"}, slices.Collect(maps.Keys(src.Profiles)))
	// Sorted path order, so output is deterministic.
	assert.Equal(t, []string{
		"mdm/profiles/com.apple.MCX(Accounts).yaml",
		"mdm/profiles/com.apple.MCX(EnergySaver).yaml",
		"mdm/profiles/com.apple.MCX(WiFi).yaml",
	}, fileNames(src.Profiles["com.apple.MCX"]))
	assert.Equal(t, "Shared", keyName(src.Profiles["com.apple.MCX"].Files[1].PayloadKeys.Content[0]))

	assert.ElementsMatch(t, []string{"com.apple.configuration.passcode.settings", "com.apple.asset.data"}, slices.Collect(maps.Keys(src.Declarations)))
}

func TestLoadAppleKeyNameThroughAlias(t *testing.T) {
	files := baseFixture()
	files["mdm/profiles/com.example.yaml"] = "payload:\n  payloadtype: com.example\npayloadkeys:\n- &shared\n  key: Reused\n  type: <string>\n"
	files["mdm/profiles/com.example2.yaml"] = "payload:\n  payloadtype: com.example2\nshared: &k\n  key: ViaAlias\n  type: <string>\npayloadkeys:\n- *k\n"

	src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
	require.NoError(t, err)
	assert.Equal(t, "Reused", keyName(src.Profiles["com.example"].Files[0].PayloadKeys.Content[0]))
	assert.Equal(t, "ViaAlias", keyName(src.Profiles["com.example2"].Files[0].PayloadKeys.Content[0]))
}

func TestLoadAppleErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(files map[string]string)
		wantErr string
	}{
		{name: "profile without payloadtype", mutate: func(f map[string]string) {
			f["mdm/profiles/broken.yaml"] = "payload:\n  title: x\n"
		}, wantErr: "mdm/profiles/broken.yaml has no payload.payloadtype"},
		{name: "missing TopLevel", mutate: func(f map[string]string) {
			delete(f, "mdm/profiles/TopLevel.yaml")
		}, wantErr: "missing TopLevel.yaml or CommonPayloadKeys.yaml"},
		{name: "missing CommonPayloadKeys", mutate: func(f map[string]string) {
			delete(f, "mdm/profiles/CommonPayloadKeys.yaml")
		}, wantErr: "missing TopLevel.yaml or CommonPayloadKeys.yaml"},
		{name: "missing declaration envelope", mutate: func(f map[string]string) {
			delete(f, "declarative/declarations/declarationbase.yaml")
		}, wantErr: "has no declaration envelope"},
		{name: "missing declarations dir", mutate: func(f map[string]string) {
			for k := range f {
				if filepath.Dir(k) != "mdm/profiles" {
					delete(f, k)
				}
			}
		}, wantErr: "declarative/declarations not found"},
		{name: "invalid YAML", mutate: func(f map[string]string) {
			f["mdm/profiles/bad.yaml"] = "payload: [unclosed\n"
		}, wantErr: "parse mdm/profiles/bad.yaml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := baseFixture()
			c.mutate(files)
			_, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}
