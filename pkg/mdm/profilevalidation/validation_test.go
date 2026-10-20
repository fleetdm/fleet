package profilevalidation

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalDoc returns a document that sits exactly on the floors and has every must-have entry.
func minimalDoc() map[string]any {
	profiles := map[string]any{
		"com.apple.applicationaccess": map[string]any{"keys": map[string]any{"allowCamera": map[string]any{"type": "boolean"}}},
	}
	for i := len(profiles); i < minAppleProfileTypes; i++ {
		profiles[fmt.Sprintf("com.example.profile%d", i)] = map[string]any{}
	}
	declarations := map[string]any{
		"com.apple.configuration.passcode.settings": map[string]any{},
	}
	for i := len(declarations); i < minAppleConfigurationDeclaration; i++ {
		declarations[fmt.Sprintf("%sexample%d", appleConfigurationPrefix, i)] = map[string]any{}
	}
	return map[string]any{
		"format_version": 1,
		"version":        map[string]any{"apple": map[string]any{"release": "Release-v27.0", "commit": "abc"}, "generated_at": "2026-10-15T00:00:00Z"},
		"apple_envelope": map[string]any{
			"TopLevel":          map[string]any{"keys": map[string]any{}},
			"CommonPayloadKeys": map[string]any{"keys": map[string]any{}},
		},
		"apple_declaration_envelope": map[string]any{"keys": map[string]any{}},
		"apple_profiles":             profiles,
		"apple_declarations":         declarations,
		"windows":                    map[string]any{},
		"windows_xsds":               map[string]any{},
	}
}

func section(doc map[string]any, name string) map[string]any {
	return doc[name].(map[string]any)
}

// removeOne deletes one entry from a section that isn't a must-have, dropping it below its floor.
func removeOne(doc map[string]any, name string, keep string) {
	for k := range section(doc, name) {
		if k != keep {
			delete(section(doc, name), k)
			return
		}
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(doc map[string]any)
		wantErr string
	}{
		{name: "minimal valid", mutate: func(doc map[string]any) {}},
		{name: "extra unknown fields", mutate: func(doc map[string]any) {
			doc["some_future_section"] = map[string]any{"x": 1}
			section(doc, "version")["microsoft"] = map[string]any{"drop": "x"}
		}},
		{name: "format_version 2", mutate: func(doc map[string]any) { doc["format_version"] = 2 }, wantErr: "unsupported profile validation schema format version: 2"},
		{name: "format_version 0", mutate: func(doc map[string]any) { doc["format_version"] = 0 }, wantErr: "unsupported profile validation schema format version: 0"},
		{name: "format_version missing", mutate: func(doc map[string]any) { delete(doc, "format_version") }, wantErr: "unsupported profile validation schema format version: 0"},
		{name: "missing TopLevel", mutate: func(doc map[string]any) { delete(section(doc, "apple_envelope"), "TopLevel") }, wantErr: "apple_envelope is missing"},
		{name: "missing CommonPayloadKeys", mutate: func(doc map[string]any) { delete(section(doc, "apple_envelope"), "CommonPayloadKeys") }, wantErr: "apple_envelope is missing"},
		{name: "missing declaration envelope", mutate: func(doc map[string]any) { delete(doc, "apple_declaration_envelope") }, wantErr: "apple_declaration_envelope is missing"},
		{name: "profiles below floor", mutate: func(doc map[string]any) {
			removeOne(doc, "apple_profiles", "com.apple.applicationaccess")
		}, wantErr: "apple_profiles has 99 payload types, want at least 100"},
		{name: "configurations below floor", mutate: func(doc map[string]any) {
			removeOne(doc, "apple_declarations", "com.apple.configuration.passcode.settings")
		}, wantErr: "apple_declarations has 39 configuration declarations, want at least 40"},
		{name: "non-configuration declarations don't count toward floor", mutate: func(doc map[string]any) {
			removeOne(doc, "apple_declarations", "com.apple.configuration.passcode.settings")
			section(doc, "apple_declarations")["com.apple.asset.credential.acme"] = map[string]any{}
			section(doc, "apple_declarations")["com.apple.management.properties"] = map[string]any{}
		}, wantErr: "apple_declarations has 39 configuration declarations"},
		{name: "missing must-have profile", mutate: func(doc map[string]any) {
			delete(section(doc, "apple_profiles"), "com.apple.applicationaccess")
			section(doc, "apple_profiles")["com.example.filler"] = map[string]any{}
		}, wantErr: "apple_profiles is missing com.apple.applicationaccess"},
		{name: "missing must-have key", mutate: func(doc map[string]any) {
			section(doc, "apple_profiles")["com.apple.applicationaccess"] = map[string]any{"keys": map[string]any{}}
		}, wantErr: "apple_profiles com.apple.applicationaccess is missing key allowCamera"},
		{name: "missing must-have declaration", mutate: func(doc map[string]any) {
			delete(section(doc, "apple_declarations"), "com.apple.configuration.passcode.settings")
			section(doc, "apple_declarations")[appleConfigurationPrefix+"filler"] = map[string]any{}
		}, wantErr: "apple_declarations is missing com.apple.configuration.passcode.settings"},
		{name: "override profiles don't count toward floor", mutate: func(doc map[string]any) {
			removeOne(doc, "apple_profiles", "com.apple.applicationaccess")
			section(doc, "apple_profiles")["com.example.filler"] = map[string]any{"source": "override"}
		}, wantErr: "apple_profiles has 99 payload types, want at least 100 non-overridden"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := minimalDoc()
			c.mutate(doc)
			data, err := json.Marshal(doc)
			require.NoError(t, err)

			got, err := Parse(data)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 1, got.FormatVersion)
			assert.Len(t, got.AppleProfiles, minAppleProfileTypes)
		})
	}
}

func TestParseInvalidInput(t *testing.T) {
	for _, in := range []string{"", "not json", `{"format_version": "1"}`, `{"format_version": 1, "apple_profiles": []}`} {
		_, err := Parse([]byte(in))
		assert.Error(t, err, "input %q", in)
	}
}
