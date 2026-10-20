package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// overrideSchema is a small schema with the shapes overrides must handle: envelopes, nested
// dictionaries, arrays, and key names containing spaces and dots.
func overrideSchema(t *testing.T) *profilevalidation.ProfileValidationSchema {
	t.Helper()
	var s profilevalidation.ProfileValidationSchema
	require.NoError(t, json.Unmarshal([]byte(`{
		"apple_envelope": {
			"TopLevel": {"source": "apple", "keys": {"PayloadVersion": {"type": "integer"}}},
			"CommonPayloadKeys": {"source": "apple", "keys": {"PayloadUUID": {"type": "string"}}}
		},
		"apple_declaration_envelope": {"source": "apple", "keys": {"Type": {"type": "string"}}},
		"apple_profiles": {
			"com.apple.security.scep": {"source": "apple", "keys": {
				"Key Type": {"type": "string"},
				"PayloadContent": {"type": "dictionary", "keys": {
					"SubjectAltName": {"type": "dictionary", "keys": {"uniformResourceIdentifier": {"type": "string"}}}}},
				"Dock": {"type": "array", "items": {"type": "dictionary", "keys": {"Type": {"type": "string"}}}}}},
			"com.apple.MCX": {"source": "apple", "keys": {
				"com.apple.EnergySaver.portable": {"type": "dictionary", "keys": {"BatteryPower": {"type": "integer"}}},
				"com.apple.EnergySaver.portable.BatteryPower": {"type": "dictionary", "keys": {"Sleep": {"type": "integer"}}}}},
			"com.apple.Safari": {"source": "apple", "keys": {"ShowFullURL": {"type": "boolean"}}}
		},
		"apple_declarations": {
			"com.apple.configuration.passcode.settings": {"source": "apple", "keys": {"MinimumLength": {"type": "integer"}}}
		}
	}`), &s))
	return &s
}

func writeOverrides(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "overrides.yml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func applyYAML(t *testing.T, s *profilevalidation.ProfileValidationSchema, content string) error {
	t.Helper()
	return applyOverrides(t.Context(), slog.New(slog.DiscardHandler), writeOverrides(t, content), s)
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func TestApplyOverridesNoFile(t *testing.T) {
	s := overrideSchema(t)
	before := jsonOf(t, s)
	logger := slog.New(slog.DiscardHandler)
	require.NoError(t, applyOverrides(t.Context(), logger, "", s))
	require.NoError(t, applyOverrides(t.Context(), logger, filepath.Join(t.TempDir(), "missing.yml"), s))
	require.NoError(t, applyYAML(t, s, "# nothing yet\n"))
	assert.Equal(t, before, jsonOf(t, s))
}

func TestApplyOverridesLoosen(t *testing.T) {
	s := overrideSchema(t)
	require.NoError(t, applyYAML(t, s, `
loosen:
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: PayloadContent.SubjectAltName.uniformResourceIdentifier
    action: types
    types: [array, string]
    reason: devices accept a list
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: Key Type
    action: any
    reason: anything goes
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: Dock[].Type
    action: types
    types: [integer]
    reason: through an array
  - kind: apple_declaration
    declaration_type: com.apple.configuration.passcode.settings
    key_path: MinimumLength
    action: types
    types: [string]
    reason: declarations too
`))
	scep := notNil(t, s.AppleProfiles["com.apple.security.scep"])
	// Types are unioned, never replaced; listing the existing type again doesn't duplicate it.
	assert.JSONEq(t, `{"type": ["string", "array"], "override": "devices accept a list"}`,
		jsonOf(t, notNil(t, notNil(t, scep.Keys["PayloadContent"]).Keys["SubjectAltName"]).Keys["uniformResourceIdentifier"]))
	assert.JSONEq(t, `{"type": "any", "open": true, "override": "anything goes"}`, jsonOf(t, scep.Keys["Key Type"]))
	assert.JSONEq(t, `{"type": ["string", "integer"], "override": "through an array"}`, jsonOf(t, notNil(t, scep.Keys["Dock"]).Items.Keys["Type"]))
	assert.JSONEq(t, `{"type": ["integer", "string"], "override": "declarations too"}`,
		jsonOf(t, notNil(t, s.AppleDeclarations["com.apple.configuration.passcode.settings"]).Keys["MinimumLength"]))
	// The payload type itself isn't marked; only what was changed.
	assert.Empty(t, scep.Override)
}

func TestApplyOverridesAdd(t *testing.T) {
	s := overrideSchema(t)
	require.NoError(t, applyYAML(t, s, `
add:
  - {kind: apple_profile, payload_type: TopLevel, name: PayloadEnabled, key: {type: boolean}, reason: top}
  - {kind: apple_profile, payload_type: CommonPayloadKeys, name: PayloadEnabled, key: {type: boolean}, reason: common}
  - {kind: apple_declaration, declaration_type: DeclarationBase, name: PayloadScope, key: {type: string, rangelist: [System, User]}, reason: fleet}
  - {kind: apple_profile, payload_type: com.apple.MCX, name: com.apple.login.mcx.Dotted, key: {type: boolean}, reason: dotted name}
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: PayloadContent
    name: ExtendedKeyUsage
    key: {type: array, items: {type: string}}
    reason: nested
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: Dock[]
    name: Label
    key: {type: string}
    reason: into array items
`))
	assert.JSONEq(t, `{"type": "boolean", "override": "top"}`, jsonOf(t, s.AppleEnvelope.TopLevel.Keys["PayloadEnabled"]))
	assert.JSONEq(t, `{"type": "boolean", "override": "common"}`, jsonOf(t, s.AppleEnvelope.CommonPayloadKeys.Keys["PayloadEnabled"]))
	assert.JSONEq(t, `{"type": "string", "rangelist": ["System", "User"], "override": "fleet"}`,
		jsonOf(t, s.AppleDeclarationEnvelope.Keys["PayloadScope"]))
	assert.JSONEq(t, `{"type": "boolean", "override": "dotted name"}`, jsonOf(t, notNil(t, s.AppleProfiles["com.apple.MCX"]).Keys["com.apple.login.mcx.Dotted"]))
	scep := notNil(t, s.AppleProfiles["com.apple.security.scep"])
	assert.JSONEq(t, `{"type": "array", "items": {"type": "string"}, "override": "nested"}`, jsonOf(t, notNil(t, scep.Keys["PayloadContent"]).Keys["ExtendedKeyUsage"]))
	assert.JSONEq(t, `{"type": "string", "override": "into array items"}`, jsonOf(t, notNil(t, scep.Keys["Dock"]).Items.Keys["Label"]))
	// Existing keys are untouched.
	assert.JSONEq(t, `{"type": "integer"}`, jsonOf(t, s.AppleEnvelope.TopLevel.Keys["PayloadVersion"]))
}

func TestApplyOverridesAddWhenAppleCaughtUp(t *testing.T) {
	// Apple now defines the key: keep going, merge loosest, and keep the reason.
	s := overrideSchema(t)
	require.NoError(t, applyYAML(t, s, `
add:
  - {kind: apple_profile, payload_type: com.apple.Safari, name: ShowFullURL, key: {type: string}, reason: was missing upstream}
`))
	assert.JSONEq(t, `{"type": ["boolean", "string"], "override": "was missing upstream"}`,
		jsonOf(t, notNil(t, s.AppleProfiles["com.apple.Safari"]).Keys["ShowFullURL"]))
}

func TestApplyOverridesAllow(t *testing.T) {
	s := overrideSchema(t)
	require.NoError(t, applyYAML(t, s, `
allow:
  - {kind: apple_profile, payload_type: com.google.Chrome, reason: vendor}
  - kind: apple_profile
    payload_type: us.zoom.config
    reason: vendor with keys
    keys:
      ZAutoSSOLogin: {type: boolean}
  - kind: apple_profile
    payload_type: com.apple.Safari
    reason: Apple shipped it
    keys:
      ShowFullURL: {type: integer}
      NewKey: {type: string}
`))
	assert.JSONEq(t, `{"source": "override", "open": true, "override": "vendor"}`, jsonOf(t, s.AppleProfiles["com.google.Chrome"]))
	assert.JSONEq(t, `{"source": "override", "open": true, "override": "vendor with keys", "keys": {"ZAutoSSOLogin": {"type": "boolean"}}}`,
		jsonOf(t, s.AppleProfiles["us.zoom.config"]))
	// An allow for a type Apple now ships keeps Apple's definition and merges ours in loosest.
	assert.JSONEq(t, `{"source": "apple", "override": "Apple shipped it", "keys": {
		"ShowFullURL": {"type": ["boolean", "integer"], "override": "Apple shipped it"},
		"NewKey": {"type": "string", "override": "Apple shipped it"}}}`, jsonOf(t, s.AppleProfiles["com.apple.Safari"]))
}

func TestApplyOverridesWindowsIsNoOp(t *testing.T) {
	s := overrideSchema(t)
	before := jsonOf(t, s)
	require.NoError(t, applyYAML(t, s, `
loosen:
  - {kind: windows, loc_uri: ./Device/Vendor/MSFT/Policy/Config/Example/Setting, key_path: x, action: any, reason: later}
`))
	assert.Equal(t, before, jsonOf(t, s))
}

func TestApplyOverridesFileErrors(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantErrs []string
	}{
		{name: "unknown field", yaml: "add:\n  - {kind: apple_profile, payload_type: x, name: y, key: {type: string}, reasn: typo}\n", wantErrs: []string{`unknown field "reasn"`}},
		{name: "unknown field in a key definition", yaml: "add:\n  - {kind: apple_profile, payload_type: x, name: y, key: {typ: string}, reason: r}\n", wantErrs: []string{`unknown field "typ"`}},
		{name: "unknown section", yaml: "remove: []\n", wantErrs: []string{`unknown field "remove"`}},
		{name: "not YAML", yaml: "loosen: [unclosed\n", wantErrs: []string{"overrides"}},
		{
			name: "every problem in the file is reported together",
			yaml: `
loosen:
  - {kind: apple_profile, payload_type: com.apple.MCX, key_path: a, action: types, types: [string], reason: r}
  - {kind: apple_profile, payload_type: com.apple.MCX, key_path: a, action: widen, reason: ""}
  - {kind: apple_profile, payload_type: com.apple.MCX, action: types, reason: r}
  - {kind: apple_profile, payload_type: com.apple.MCX, key_path: b, action: any, types: [string], reason: r}
  - {kind: apple_profile, payload_type: com.apple.MCX, key_path: c, action: any, name: n, reason: r}
add:
  - {kind: apple_profle, payload_type: x, name: n, key: {type: string}, reason: r}
  - {kind: apple_profile, payload_type: x, key: {type: string}, reason: r}
  - {kind: apple_profile, payload_type: x, name: n, reason: r}
  - {kind: apple_profile, payload_type: x, declaration_type: y, name: m, key: {type: string}, reason: r}
  - {kind: apple_declaration, payload_type: x, name: m, key: {type: string}, reason: r}
  - {kind: windows, name: m, key: {type: string}, reason: r}
allow:
  - {kind: apple_declaration, declaration_type: com.example.x, reason: r}
  - {kind: apple_profile, payload_type: TopLevel, reason: r}
  - {kind: apple_profile, payload_type: com.example.y, key_path: k, reason: r}
  - {kind: apple_profile, payload_type: com.example.z, reason: r, keys: {K: {}}}
  - {kind: apple_profile, payload_type: com.example.z, reason: again}
`,
			wantErrs: []string{
				"loosen[1] (apple_profile com.apple.MCX a): reason is required",
				`loosen[1] (apple_profile com.apple.MCX a): action must be "types" or "any", got "widen"`,
				"loosen[1] (apple_profile com.apple.MCX a): targets the same thing as loosen[0]",
				"loosen[2] (apple_profile com.apple.MCX ): key_path is required",
				`loosen[2] (apple_profile com.apple.MCX ): action "types" needs a non-empty types list`,
				`loosen[3] (apple_profile com.apple.MCX b): action "any" takes no types`,
				"loosen[4] (apple_profile com.apple.MCX c): loosen takes no name, key, or keys",
				`add[0] (apple_profle  ): kind must be apple_profile, apple_declaration, or windows, got "apple_profle"`,
				"add[1] (apple_profile x ): name is required",
				"add[2] (apple_profile x ): key with a type is required",
				"add[3] (apple_profile x ): apple_profile entries need payload_type (and no declaration_type or loc_uri)",
				"add[4] (apple_declaration  ): apple_declaration entries need declaration_type",
				"add[5] (windows  ): windows entries need loc_uri",
				"allow[0] (apple_declaration com.example.x ): allow is apple_profile only",
				"allow[1] (apple_profile TopLevel ): the profile envelope can't be allowed",
				"allow[2] (apple_profile com.example.y k): allow takes only payload_type, keys, and reason",
				"allow[3] (apple_profile com.example.z ): keys.K needs a type",
				"allow[4] (apple_profile com.example.z ): targets the same thing as allow[3]",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := overrideSchema(t)
			before := jsonOf(t, s)
			err := applyYAML(t, s, c.yaml)
			require.Error(t, err)
			for _, want := range c.wantErrs {
				assert.Contains(t, err.Error(), want)
			}
			// Nothing is applied when the file itself is wrong.
			assert.Equal(t, before, jsonOf(t, s))
		})
	}
}

func TestApplyOverridesSchemaErrors(t *testing.T) {
	// The file is valid; the entries don't fit the schema. All are reported together.
	err := applyYAML(t, overrideSchema(t), `
loosen:
  - {kind: apple_profile, payload_type: com.apple.MCX, key_path: noSuchKey, action: any, reason: r}
  - {kind: apple_profile, payload_type: com.example.missing, key_path: x, action: any, reason: r}
add:
  - {kind: apple_profile, payload_type: com.apple.security.scep, key_path: PayloadContent.missing, name: n, key: {type: string}, reason: r}
  - {kind: apple_profile, payload_type: com.apple.security.scep, key_path: Key Type, name: n, key: {type: string}, reason: r}
  - {kind: apple_declaration, declaration_type: com.apple.configuration.nope, name: n, key: {type: string}, reason: r}
`)
	require.Error(t, err)
	for _, want := range []string{
		`loosen[0] (apple_profile com.apple.MCX noSuchKey): key_path "noSuchKey" not found (no key matches "noSuchKey"); Apple may have renamed or removed it`,
		"loosen[1] (apple_profile com.example.missing x): com.example.missing isn't in the schema",
		`add[0] (apple_profile com.apple.security.scep PayloadContent.missing): key_path "PayloadContent.missing" not found (no key matches "missing")`,
		`add[1] (apple_profile com.apple.security.scep Key Type): key_path "Key Type" is a [string], not a dictionary`,
		"add[2] (apple_declaration com.apple.configuration.nope ): com.apple.configuration.nope isn't in the schema",
	} {
		assert.Contains(t, err.Error(), want)
	}
	assert.Len(t, strings.Split(err.Error(), "\n"), 6, "a header line plus one line per problem")
}

func TestResolveKeyPath(t *testing.T) {
	s := overrideSchema(t)
	scep := &profilevalidation.AppleKey{Keys: notNil(t, s.AppleProfiles["com.apple.security.scep"]).Keys}
	mcx := &profilevalidation.AppleKey{Keys: notNil(t, s.AppleProfiles["com.apple.MCX"]).Keys}

	cases := []struct {
		name     string
		root     *profilevalidation.AppleKey
		path     string
		want     *profilevalidation.AppleKey
		wantRest string
	}{
		{name: "empty path is the root", root: scep, path: "", want: scep},
		{name: "nested", root: scep, path: "PayloadContent.SubjectAltName", want: notNil(t, scep.Keys["PayloadContent"]).Keys["SubjectAltName"]},
		{name: "name with a space", root: scep, path: "Key Type", want: scep.Keys["Key Type"]},
		{name: "array items", root: scep, path: "Dock[].Type", want: notNil(t, scep.Keys["Dock"]).Items.Keys["Type"]},
		{name: "array itself", root: scep, path: "Dock[]", want: notNil(t, scep.Keys["Dock"]).Items},
		{name: "partial match reports the rest", root: scep, path: "PayloadContent.Nope.Deeper", want: scep.Keys["PayloadContent"], wantRest: "Nope.Deeper"},
		{name: "items on a non-array", root: scep, path: "Key Type[]", want: scep.Keys["Key Type"], wantRest: "[]"},
		{name: "prefix of a key name isn't a match", root: scep, path: "Key", want: scep, wantRest: "Key"},
		// Both "com.apple.EnergySaver.portable" and "com.apple.EnergySaver.portable.BatteryPower" exist;
		// the longest name that fits wins.
		{
			name: "longest dotted name wins", root: mcx, path: "com.apple.EnergySaver.portable.BatteryPower.Sleep",
			want: notNil(t, mcx.Keys["com.apple.EnergySaver.portable.BatteryPower"]).Keys["Sleep"],
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, rest := resolveKeyPath(c.root, c.path)
			assert.Same(t, c.want, got)
			assert.Equal(t, c.wantRest, rest)
		})
	}
}

func TestSeedOverridesFileIsValid(t *testing.T) {
	// The committed seed must decode and validate; applying it to real data is covered by
	// regenerating the embedded schema.
	data, err := os.ReadFile("../../pkg/mdm/profilevalidation/inputs/overrides.yml")
	require.NoError(t, err)
	file, err := decodeOverrides(data)
	require.NoError(t, err)
	require.NoError(t, file.validate())
	assert.Len(t, file.Allow, 20, "the ticket's seed allow list")
}

func TestConvertAppleDropsServerToken(t *testing.T) {
	files := baseFixture()
	files["declarative/declarations/declarationbase.yaml"] = declarationYAML("any", "Type", "Identifier", "ServerToken")
	src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
	require.NoError(t, err)

	var s profilevalidation.ProfileValidationSchema
	require.NoError(t, convertApple(t.Context(), slog.New(slog.DiscardHandler), src, &s))
	assert.NotContains(t, s.AppleDeclarationEnvelope.Keys, "ServerToken")
	assert.Contains(t, s.AppleDeclarationEnvelope.Keys, "Identifier")
}
