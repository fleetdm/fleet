package main

import (
	"encoding/json"
	"log/slog"
	"maps"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyFromJSON(t *testing.T, s string) *profilevalidation.AppleKey {
	t.Helper()
	var k profilevalidation.AppleKey
	require.NoError(t, json.Unmarshal([]byte(s), &k))
	return &k
}

func TestMergeKey(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want string
	}{
		{name: "identical", a: `{"type": "string", "required": true}`, b: `{"type": "string", "required": true}`, want: `{"type": "string", "required": true}`},
		{name: "types union", a: `{"type": "integer"}`, b: `{"type": ["string", "integer"]}`, want: `{"type": ["integer", "string"]}`},
		{name: "required only if both", a: `{"type": "string", "required": true}`, b: `{"type": "string"}`, want: `{"type": "string"}`},
		{name: "rangelist union", a: `{"type": "string", "rangelist": ["A", 1]}`, b: `{"type": "string", "rangelist": [1, "B"]}`, want: `{"type": "string", "rangelist": ["A", 1, "B"]}`},
		{name: "rangelist dropped if one side has none", a: `{"type": "string", "rangelist": ["A"]}`, b: `{"type": "string"}`, want: `{"type": "string"}`},
		{name: "range widened", a: `{"type": "integer", "range": {"min": 1, "max": 10}}`, b: `{"type": "integer", "range": {"min": 0, "max": 5}}`, want: `{"type": "integer", "range": {"min": 0, "max": 10}}`},
		{name: "range bound dropped if one side lacks it", a: `{"type": "integer", "range": {"min": 1, "max": 10}}`, b: `{"type": "integer", "range": {"min": 2}}`, want: `{"type": "integer", "range": {"min": 1}}`},
		{name: "range dropped if one side has none", a: `{"type": "integer", "repetition": {"min": 1}}`, b: `{"type": "integer"}`, want: `{"type": "integer"}`},
		{name: "format kept when equal", a: `{"type": "string", "format": "^a$"}`, b: `{"type": "string", "format": "^a$"}`, want: `{"type": "string", "format": "^a$"}`},
		{name: "format dropped when different", a: `{"type": "string", "format": "^a$"}`, b: `{"type": "string", "format": "^b$"}`, want: `{"type": "string"}`},
		{name: "valuetype dropped when different", a: `{"type": "string", "valuetype": "url"}`, b: `{"type": "string"}`, want: `{"type": "string"}`},
		{
			name: "keys merged recursively and unioned",
			a:    `{"type": "dictionary", "keys": {"X": {"type": "string", "required": true}, "OnlyA": {"type": "boolean"}}}`,
			b:    `{"type": "dictionary", "keys": {"X": {"type": "string"}, "OnlyB": {"type": "integer"}}}`,
			want: `{"type": "dictionary", "keys": {"X": {"type": "string"}, "OnlyA": {"type": "boolean"}, "OnlyB": {"type": "integer"}}}`,
		},
		{
			name: "open side keeps the other side's keys",
			a:    `{"type": "dictionary", "keys": {"cacheName": {"type": "string"}}}`,
			b:    `{"type": "dictionary", "open": true, "values": {"type": "any", "open": true}}`,
			want: `{"type": "dictionary", "keys": {"cacheName": {"type": "string"}}, "open": true, "values": {"type": "any", "open": true}}`,
		},
		{
			name: "open without values accepts anything and wins",
			a:    `{"type": "dictionary", "open": true, "values": {"type": "string"}}`,
			b:    `{"type": "dictionary", "open": true}`,
			want: `{"type": "dictionary", "open": true}`,
		},
		{
			name: "values of two open sides merged",
			a:    `{"type": "dictionary", "open": true, "values": {"type": "string", "rangelist": ["A"]}}`,
			b:    `{"type": "dictionary", "open": true, "values": {"type": "string", "rangelist": ["B"]}}`,
			want: `{"type": "dictionary", "open": true, "values": {"type": "string", "rangelist": ["A", "B"]}}`,
		},
		{
			name: "items merged",
			a:    `{"type": "array", "items": {"type": "integer", "range": {"min": 1}}}`,
			b:    `{"type": "array", "items": {"type": "integer", "range": {"min": 0}}}`,
			want: `{"type": "array", "items": {"type": "integer", "range": {"min": 0}}}`,
		},
		{name: "items dropped if one side has none", a: `{"type": "array", "items": {"type": "integer"}}`, b: `{"type": "array"}`, want: `{"type": "array"}`},
		{
			name: "platforms merged per OS",
			a:    `{"type": "string", "platforms": {"iOS": {"introduced": "13.0"}, "macOS": {"introduced": "10.15"}}}`,
			b:    `{"type": "string", "platforms": {"iOS": {"introduced": "n/a"}, "tvOS": {"introduced": "9.0"}}}`,
			want: `{"type": "string", "platforms": {"iOS": {"introduced": "13.0"}, "macOS": {"introduced": "10.15"}, "tvOS": {"introduced": "9.0"}}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := &appleMerger{ctx: t.Context(), logger: slog.New(slog.DiscardHandler)}
			got, err := json.Marshal(m.mergeKey("K", keyFromJSON(t, c.a), keyFromJSON(t, c.b)))
			require.NoError(t, err)
			assert.JSONEq(t, c.want, string(got))
		})
	}
}

func TestMergePlatform(t *testing.T) {
	p := func(introduced, deprecated, removed string, supervised bool) profilevalidation.ApplePlatform {
		return profilevalidation.ApplePlatform{Introduced: introduced, Deprecated: deprecated, Removed: removed, Supervised: supervised}
	}
	cases := []struct {
		name string
		a, b profilevalidation.ApplePlatform
		want profilevalidation.ApplePlatform
	}{
		{name: "earliest introduction", a: p("10.12.4", "", "", false), b: p("10.7", "", "", false), want: p("10.7", "", "", false)},
		{name: "numeric, not string, order", a: p("10.10", "", "", false), b: p("10.9", "", "", false), want: p("10.9", "", "", false)},
		{name: "n/a loses entirely", a: p("n/a", "", "", false), b: p("13.0", "14.0", "", true), want: p("13.0", "14.0", "", true)},
		{name: "empty introduced loses", a: p("13.0", "", "", false), b: p("", "", "", true), want: p("13.0", "", "", false)},
		{name: "both n/a", a: p("n/a", "", "", false), b: p("n/a", "", "", false), want: p("n/a", "", "", false)},
		{name: "deprecated and removed only if both, then the later", a: p("1.0", "26.0", "27.0", false), b: p("1.0", "25.0", "", false), want: p("1.0", "26.0", "", false)},
		{name: "supervised only if both", a: p("1.0", "", "", true), b: p("1.0", "", "", false), want: p("1.0", "", "", false)},
		{name: "supervised kept when both", a: p("1.0", "", "", true), b: p("2.0", "", "", true), want: p("1.0", "", "", true)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, mergePlatform(c.a, c.b))
			assert.Equal(t, c.want, mergePlatform(c.b, c.a), "merge must be symmetric")
		})
	}
}

func TestCompareVersions(t *testing.T) {
	assert.Positive(t, compareVersions("10.10", "10.9"))
	assert.Negative(t, compareVersions("10.9", "10.10"))
	assert.Zero(t, compareVersions("10.0", "10"))
	assert.Positive(t, compareVersions("10.13.4", "10.13"))
	assert.Zero(t, compareVersions("26.0", "26.0"))
	// Non-numeric falls back to string order instead of failing.
	assert.Negative(t, compareVersions("10.beta", "10.c"))
}

func TestConvertAppleGroup(t *testing.T) {
	load := func(t *testing.T, files map[string]string) *appleSources {
		all := baseFixture()
		maps.Copy(all, files)
		src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, all))
		require.NoError(t, err)
		return src
	}
	convert := func(t *testing.T, g *appleGroup) string {
		pt, err := convertAppleGroup(t.Context(), slog.New(slog.DiscardHandler), g)
		require.NoError(t, err)
		b, err := json.Marshal(pt)
		require.NoError(t, err)
		return string(b)
	}

	t.Run("stricter variant file doesn't make the general one stricter", func(t *testing.T) {
		src := load(t, map[string]string{
			"mdm/profiles/com.apple.extensiblesso(kerberos).yaml": `
payload:
  payloadtype: com.apple.extensiblesso
  supportedOS: {iOS: {introduced: '13.0'}, macOS: {introduced: '10.15'}}
payloadkeys:
- {key: Type, type: <string>, presence: required, rangelist: [Credential]}
- {key: Realm, type: <string>, presence: required}
- key: ExtensionData
  type: <dictionary>
  subkeys:
  - {key: cacheName, type: <string>}
- {key: KerberosOnly, type: <boolean>}`,
			"mdm/profiles/com.apple.extensiblesso.yaml": `
payload:
  payloadtype: com.apple.extensiblesso
  supportedOS: {iOS: {introduced: '13.0'}, macOS: {introduced: '10.15'}}
payloadkeys:
- {key: Type, type: <string>, presence: required, rangelist: [Credential, Redirect]}
- {key: Realm, type: <string>, presence: optional}
- key: ExtensionData
  type: <dictionary>
  subkeys:
  - {key: ANY, type: <any>}`,
		})
		assert.JSONEq(t, `{"source": "apple",
			"platforms": {"iOS": {"introduced": "13.0"}, "macOS": {"introduced": "10.15"}},
			"keys": {
				"Type": {"type": "string", "required": true, "rangelist": ["Credential", "Redirect"]},
				"Realm": {"type": "string"},
				"ExtensionData": {"type": "dictionary", "keys": {"cacheName": {"type": "string"}}, "open": true, "values": {"type": "any", "open": true}},
				"KerberosOnly": {"type": "boolean"}}}`, convert(t, src.Profiles["com.apple.extensiblesso"]))
	})

	t.Run("keys keep their own file's platforms relative to the merged type", func(t *testing.T) {
		src := load(t, map[string]string{
			"mdm/profiles/com.apple.MCX(Accounts).yaml": `
payload:
  payloadtype: com.apple.MCX
  supportedOS: {iOS: {introduced: n/a}, macOS: {introduced: '10.7'}}
payloadkeys:
- {key: cachedaccounts, type: <boolean>}`,
			"mdm/profiles/com.apple.MCX(TimeServer).yaml": `
payload:
  payloadtype: com.apple.MCX
  supportedOS: {iOS: {introduced: n/a}, macOS: {introduced: '10.12.4'}}
payloadkeys:
- {key: timeServer, type: <string>}
- key: timeZone
  type: <string>
  supportedOS: {macOS: {introduced: '10.7'}}`,
		})
		assert.JSONEq(t, `{"source": "apple",
			"platforms": {"iOS": {"introduced": "n/a"}, "macOS": {"introduced": "10.7"}},
			"keys": {
				"cachedaccounts": {"type": "boolean"},
				"timeServer": {"type": "string", "platforms": {"macOS": {"introduced": "10.12.4"}}},
				"timeZone": {"type": "string"}}}`, convert(t, src.Profiles["com.apple.MCX"]))
	})

	t.Run("single file", func(t *testing.T) {
		src := load(t, nil)
		assert.JSONEq(t, `{"source": "apple", "keys": {"allowCamera": {"type": "string"}}}`,
			convert(t, src.Profiles["com.apple.applicationaccess"]))
	})

	t.Run("payload without payloadkeys is not open", func(t *testing.T) {
		// Unlike a nested dictionary without subkeys, which accepts anything.
		src := load(t, map[string]string{"mdm/profiles/com.example.empty.yaml": "payload:\n  payloadtype: com.example.empty\n"})
		assert.JSONEq(t, `{"source": "apple"}`, convert(t, src.Profiles["com.example.empty"]))
	})

	t.Run("conversion error names the file", func(t *testing.T) {
		src := load(t, map[string]string{
			"mdm/profiles/bad.yaml": "payload:\n  payloadtype: com.example.bad\npayloadkeys:\n- {key: L, type: <array>, subkeys: [{key: A, type: <string>}, {key: B, type: <string>}]}\n",
		})
		_, err := convertAppleGroup(t.Context(), slog.New(slog.DiscardHandler), src.Profiles["com.example.bad"])
		require.ErrorContains(t, err, "mdm/profiles/bad.yaml: array L has 2 subkeys")
	})
}
