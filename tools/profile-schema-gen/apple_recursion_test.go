package main

import (
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// homeScreenYAML mirrors com.apple.homescreenlayout: the subkeys lists are anchored, folders hold
// pages that hold icons, and the top-level Pages key enters the same cycle one level down.
const homeScreenYAML = `
- key: Dock
  type: <array>
  subkeys: &icons
  - key: IconItem
    type: <dictionary>
    subkeys:
    - key: Type
      type: <string>
      rangelist: [Application, Folder]
    - key: Pages
      type: <array>
      subkeys: &pages
      - key: PagesItem
        type: <array>
        subkeys: *icons
- key: Pages
  type: <array>
  subkeys: *pages`

func TestConvertAppleRecursion(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "homescreenlayout: inner occurrence refs the enclosing definition",
			yaml: homeScreenYAML,
			want: `{"open": false, "values": null, "keys": {
				"Dock": {"type": "array", "items": {"type": "dictionary", "keys": {
					"Type": {"type": "string", "rangelist": ["Application", "Folder"]},
					"Pages": {"type": "array", "items": {"type": "array", "items": {"type": "dictionary", "ref": "Dock[]"}}}}}},
				"Pages": {"type": "array", "items": {"type": "array", "items": {"type": "dictionary", "keys": {
					"Type": {"type": "string", "rangelist": ["Application", "Folder"]},
					"Pages": {"type": "array", "items": {"type": "array", "ref": "Pages[]"}}}}}}}}`,
		},
		{
			// Like safari.bookmarks: the cycle closes on a dictionary key, not an array item.
			name: "cycle through a dictionary key",
			yaml: `
- key: Bookmarks
  type: <array>
  subkeys:
  - key: Item
    type: <dictionary>
    subkeys:
    - &folder
      key: Folder
      type: <dictionary>
      subkeys:
      - key: Title
        type: <string>
      - *folder`,
			want: `{"open": false, "values": null, "keys": {
				"Bookmarks": {"type": "array", "items": {"type": "dictionary", "keys": {
					"Folder": {"type": "dictionary", "keys": {
						"Title": {"type": "string"},
						"Folder": {"type": "dictionary", "ref": "Bookmarks[].Folder"}}}}}}}}`,
		},
		{
			name: "definition reused side by side is expanded, not a ref",
			yaml: `
- key: A
  type: <dictionary>
  subkeys: &shared
  - key: Name
    type: <string>
- key: B
  type: <dictionary>
  subkeys:
  - key: Inner
    type: <dictionary>
    subkeys: *shared`,
			want: `{"open": false, "values": null, "keys": {
				"A": {"type": "dictionary", "keys": {"Name": {"type": "string"}}},
				"B": {"type": "dictionary", "keys": {"Inner": {"type": "dictionary", "keys": {"Name": {"type": "string"}}}}}}}`,
		},
		{
			name: "ref goes through the platform trim like any key",
			yaml: `
- key: Tree
  type: <array>
  subkeys: &nodes
  - key: Node
    type: <dictionary>
    supportedOS: {iOS: {introduced: '17.0'}}
    subkeys:
    - key: Children
      type: <array>
      subkeys: *nodes`,
			want: `{"open": false, "values": null, "keys": {
				"Tree": {"type": "array", "items": {"type": "dictionary", "platforms": {"iOS": {"introduced": "17.0"}}, "keys": {
					"Children": {"type": "array", "items": {"type": "dictionary", "ref": "Tree[]"}}}}}}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := convertYAMLWithPlatforms(t, c.yaml, platforms{profilevalidation.AppleOSiOS: {Introduced: "9.3"}})
			require.NoError(t, err)
			assert.JSONEq(t, c.want, got)
		})
	}
}

func TestConvertAppleRecursionMerged(t *testing.T) {
	// Two files of one type with the same recursive definition merge without touching the ref.
	files := baseFixture()
	files["mdm/profiles/com.example.tree(a).yaml"] = "payload:\n  payloadtype: com.example.tree\npayloadkeys:" + homeScreenYAML
	files["mdm/profiles/com.example.tree(b).yaml"] = "payload:\n  payloadtype: com.example.tree\npayloadkeys:" + homeScreenYAML
	src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
	require.NoError(t, err)

	pt, err := convertAppleGroup(t.Context(), slog.New(slog.DiscardHandler), src.Profiles["com.example.tree"])
	require.NoError(t, err)
	assert.Equal(t, "Dock[]", pt.Keys["Dock"].Items.Keys["Pages"].Items.Items.Ref)
	assert.Equal(t, "Pages[]", pt.Keys["Pages"].Items.Items.Keys["Pages"].Items.Ref)
}

func TestConvertApple(t *testing.T) {
	files := baseFixture()
	files["mdm/profiles/com.apple.homescreenlayout.yaml"] = "payload:\n  payloadtype: com.apple.homescreenlayout\n  supportedOS: {iOS: {introduced: '9.3', supervised: true}}\npayloadkeys:" + homeScreenYAML
	files["declarative/declarations/management/properties.yaml"] = "payload:\n  declarationtype: com.apple.management.properties\npayloadkeys:\n- {key: ANY, type: <any>}\n"
	src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
	require.NoError(t, err)

	var s profilevalidation.ProfileValidationSchema
	require.NoError(t, convertApple(t.Context(), slog.New(slog.DiscardHandler), src, &s))

	asJSON := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}
	assert.JSONEq(t, `{"source": "apple", "keys": {"PayloadVersion": {"type": "string"}}}`, asJSON(s.AppleEnvelope.TopLevel))
	assert.JSONEq(t, `{"source": "apple", "keys": {"PayloadUUID": {"type": "string"}}}`, asJSON(s.AppleEnvelope.CommonPayloadKeys))
	assert.JSONEq(t, `{"source": "apple", "keys": {"Type": {"type": "string"}, "Identifier": {"type": "string"}}}`, asJSON(s.AppleDeclarationEnvelope))

	assert.ElementsMatch(t, []string{"com.apple.applicationaccess", "com.apple.homescreenlayout"}, slices.Collect(maps.Keys(s.AppleProfiles)))
	assert.ElementsMatch(t, []string{"com.apple.configuration.passcode.settings", "com.apple.management.properties"}, slices.Collect(maps.Keys(s.AppleDeclarations)))

	home := s.AppleProfiles["com.apple.homescreenlayout"]
	assert.Equal(t, platforms{profilevalidation.AppleOSiOS: {Introduced: "9.3", Supervised: true}}, home.Platforms)
	assert.Equal(t, "Dock[]", home.Keys["Dock"].Items.Keys["Pages"].Items.Items.Ref)
	// Keys inherit the payload platforms, so none record their own.
	assert.Nil(t, home.Keys["Dock"].Platforms)

	props := s.AppleDeclarations["com.apple.management.properties"]
	assert.True(t, props.Open)
	assert.Nil(t, props.Keys)

	// Nothing about the Windows sections is touched.
	assert.Nil(t, s.Windows)
}

func TestConvertAppleError(t *testing.T) {
	files := baseFixture()
	files["declarative/declarations/configurations/bad.yaml"] = "payload:\n  declarationtype: com.apple.configuration.bad\npayloadkeys:\n- {type: <string>}\n"
	src, err := loadApple(t.Context(), slog.New(slog.DiscardHandler), writeAppleFixture(t, files))
	require.NoError(t, err)

	var s profilevalidation.ProfileValidationSchema
	require.ErrorContains(t, convertApple(t.Context(), slog.New(slog.DiscardHandler), src, &s), "declarative/declarations/configurations/bad.yaml: key without a name")
}
