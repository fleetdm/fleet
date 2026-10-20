package main

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// convertYAML converts a YAML payloadkeys list and returns the result as JSON, for comparison
// against the output contract.
func convertYAML(t *testing.T, src string) (string, error) {
	t.Helper()
	return convertYAMLWithPlatforms(t, src, nil)
}

func convertYAMLWithPlatforms(t *testing.T, src string, parent map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) (string, error) {
	t.Helper()
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &doc))
	require.Equal(t, yaml.SequenceNode, doc.Content[0].Kind)

	keys, open, values, err := newAppleConverter(t.Context(), slog.New(slog.DiscardHandler), "test.yaml").convertKeys(doc.Content[0].Content, "", parent)
	if err != nil {
		return "", err
	}
	trimPlatforms(keys, parent)
	trimKeyPlatforms(values, parent)
	out, err := json.Marshal(map[string]any{"keys": keys, "open": open, "values": values})
	require.NoError(t, err)
	return string(out), nil
}

func TestConvertAppleKeys(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "scalar fields carried, descriptions and defaults dropped",
			yaml: `
- key: Delay
  title: Delay
  content: How long to wait.
  type: <integer>
  presence: required
  default: 30
  combinetype: max
  range: {min: 0, max: 90}
  rangelist: [0, 30, 90]
- key: Server
  type: <string>
  presence: optional
  format: ^[a-z]+$
  repetition: {min: 1, max: 2}
  valuetype: hostname
- key: Ratio
  type: <real>
  rangelist: [0.5, 1.5]
- key: Missing presence
  type: <boolean>`,
			want: `{"open": false, "values": null, "keys": {
				"Delay": {"type": "integer", "required": true, "range": {"min": 0, "max": 90}, "rangelist": [0, 30, 90]},
				"Server": {"type": "string", "format": "^[a-z]+$", "repetition": {"min": 1, "max": 2}, "valuetype": "hostname"},
				"Ratio": {"type": "real", "rangelist": [0.5, 1.5]},
				"Missing presence": {"type": "boolean"}}}`,
		},
		{
			name: "deprecated subtype becomes valuetype, valuetype wins when both",
			yaml: `
- key: Old
  type: <string>
  subtype: url
- key: Both
  type: <string>
  subtype: url
  valuetype: email`,
			want: `{"open": false, "values": null, "keys": {
				"Old": {"type": "string", "valuetype": "url"},
				"Both": {"type": "string", "valuetype": "email"}}}`,
		},
		{
			name: "unknown type and field carried as declared",
			yaml: `
- key: Future
  type: <uuid>
  somethingNew: true
- key: NoBrackets
  type: string`,
			want: `{"open": false, "values": null, "keys": {
				"Future": {"type": "uuid"},
				"NoBrackets": {"type": "string"}}}`,
		},
		{
			name: "array items and nested dictionary",
			yaml: `
- key: Dock
  type: <array>
  subkeys:
  - key: IconItem
    type: <dictionary>
    subkeys:
    - key: Type
      type: <string>
      presence: required
      rangelist: [Application, Folder]
- key: Matrix
  type: <array>
  subkeys:
  - key: Row
    type: <array>
    subkeys:
    - key: Cell
      type: <integer>
- key: Untyped
  type: <array>`,
			want: `{"open": false, "values": null, "keys": {
				"Dock": {"type": "array", "items": {"type": "dictionary", "keys": {
					"Type": {"type": "string", "required": true, "rangelist": ["Application", "Folder"]}}}},
				"Matrix": {"type": "array", "items": {"type": "array", "items": {"type": "integer"}}},
				"Untyped": {"type": "array"}}}`,
		},
		{
			name: "ANY subkey makes the dictionary open with typed values",
			yaml: `
- key: AllowedSystemExtensions
  type: <dictionary>
  subkeys:
  - key: ANY
    type: <array>
    subkeys:
    - key: Item
      type: <string>
- key: VendorConfig
  type: <dictionary>
  subkeys:
  - key: ANY
    type: <any>`,
			want: `{"open": false, "values": null, "keys": {
				"AllowedSystemExtensions": {"type": "dictionary", "open": true, "values": {"type": "array", "items": {"type": "string"}}},
				"VendorConfig": {"type": "dictionary", "open": true, "values": {"type": "any", "open": true}}}}`,
		},
		{
			name: "payload-level ANY",
			yaml: `
- key: ANY
  type: <any>`,
			want: `{"open": true, "values": {"type": "any", "open": true}, "keys": null}`,
		},
		{
			name: "dictionary without subkeys is open",
			yaml: `
- key: Payload
  type: <dictionary>
  presence: required`,
			want: `{"open": false, "values": null, "keys": {
				"Payload": {"type": "dictionary", "required": true, "open": true}}}`,
		},
		{
			name: "subkeys on a scalar are ignored",
			yaml: `
- key: Mode
  type: <string>
  subkeys:
  - key: Ignored
    type: <string>`,
			want: `{"open": false, "values": null, "keys": {"Mode": {"type": "string"}}}`,
		},
		{
			name: "aliased key is expanded where used",
			yaml: `
- &shared
  key: Name
  type: <string>
  presence: required
- key: Wrapper
  type: <dictionary>
  subkeys:
  - *shared`,
			want: `{"open": false, "values": null, "keys": {
				"Name": {"type": "string", "required": true},
				"Wrapper": {"type": "dictionary", "keys": {"Name": {"type": "string", "required": true}}}}}`,
		},
		{
			// Apple anchors whole subkeys lists; losing these silently was a real bug.
			name: "aliased subkeys list is expanded where used",
			yaml: `
- key: First
  type: <array>
  subkeys: &items
  - key: Item
    type: <integer>
    range: {min: 1}
- key: Second
  type: <array>
  subkeys: *items`,
			want: `{"open": false, "values": null, "keys": {
				"First": {"type": "array", "items": {"type": "integer", "range": {"min": 1}}},
				"Second": {"type": "array", "items": {"type": "integer", "range": {"min": 1}}}}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := convertYAML(t, c.yaml)
			require.NoError(t, err)
			assert.JSONEq(t, c.want, got)
		})
	}
}

func TestConvertAppleKeysErrors(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "array with two subkeys",
			yaml: `
- key: List
  type: <array>
  subkeys:
  - {key: A, type: <string>}
  - {key: B, type: <string>}`,
			wantErr: "test.yaml: array List has 2 subkeys, want 1",
		},
		{
			name: "duplicate key",
			yaml: `
- key: Box
  type: <dictionary>
  subkeys:
  - {key: Name, type: <string>}
  - {key: Name, type: <integer>}`,
			wantErr: `test.yaml: key "Box.Name" is defined twice`,
		},
		{
			name:    "key without a name",
			yaml:    "\n- type: <string>",
			wantErr: "test.yaml: key without a name",
		},
		{
			name:    "key that isn't a mapping",
			yaml:    "\n- just a string",
			wantErr: "test.yaml: decode key",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := convertYAML(t, c.yaml)
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}

func TestKnownAppleType(t *testing.T) {
	for _, typ := range []string{"string", "integer", "real", "boolean", "date", "data", "array", "dictionary", "any"} {
		assert.True(t, knownAppleType(profilevalidation.AppleType(typ)), typ)
	}
	assert.False(t, knownAppleType("uuid"))
}
