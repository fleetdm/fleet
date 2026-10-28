package main

import (
	"maps"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type platforms = map[profilevalidation.AppleOS]profilevalidation.ApplePlatform

func rawPlatforms(t *testing.T, src string) appleRawPlatforms {
	t.Helper()
	var p appleRawPlatforms
	require.NoError(t, yaml.Unmarshal([]byte(src), &p))
	return p
}

func TestAppleRawPlatformsResolve(t *testing.T) {
	parent := platforms{
		profilevalidation.AppleOSiOS:   {Introduced: "4.0", Supervised: true},
		profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
	}

	cases := []struct {
		name string
		own  string
		want platforms
	}{
		{name: "no supportedOS inherits parent", own: "{}", want: parent},
		{
			name: "fields not set are inherited",
			own:  "iOS: {introduced: '11.0'}",
			want: platforms{
				profilevalidation.AppleOSiOS:   {Introduced: "11.0", Supervised: true},
				profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
			},
		},
		{
			name: "explicit false overrides inherited true",
			own:  "iOS: {supervised: false}",
			want: platforms{
				profilevalidation.AppleOSiOS:   {Introduced: "4.0"},
				profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
			},
		},
		{
			name: "lifecycle and n/a",
			own:  "macOS: {introduced: n/a}\niOS: {deprecated: '26.0', removed: '27.0'}",
			want: platforms{
				profilevalidation.AppleOSiOS:   {Introduced: "4.0", Deprecated: "26.0", Removed: "27.0", Supervised: true},
				profilevalidation.AppleOSmacOS: {Introduced: "n/a"},
			},
		},
		{
			name: "platform the parent lacks, and an unknown one carried as declared",
			own:  "tvOS: {introduced: '9.0'}\nfutureOS: {introduced: '1.0'}",
			want: platforms{
				profilevalidation.AppleOSiOS:   {Introduced: "4.0", Supervised: true},
				profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
				profilevalidation.AppleOStvOS:  {Introduced: "9.0"},
				"futureOS":                     {Introduced: "1.0"},
			},
		},
		{
			name: "fields Fleet doesn't keep are dropped",
			own:  "macOS: {userenrollment: {mode: forbidden}, devicechannel: true, allowmanualinstall: true}",
			want: parent,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := maps.Clone(parent)
			assert.Equal(t, c.want, rawPlatforms(t, c.own).resolve(parent))
			assert.Equal(t, before, parent, "parent must not be modified")
		})
	}

	// The payload type has no parent.
	assert.Equal(t, platforms{profilevalidation.AppleOSiOS: {Introduced: "15.0"}}, rawPlatforms(t, "iOS: {introduced: '15.0'}").resolve(nil))
	assert.Nil(t, rawPlatforms(t, "{}").resolve(nil))
}

func TestPlatformsDiff(t *testing.T) {
	parent := platforms{
		profilevalidation.AppleOSiOS:   {Introduced: "4.0"},
		profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
	}
	assert.Nil(t, platformsDiff(parent, parent))
	assert.Nil(t, platformsDiff(nil, nil))
	assert.Equal(t,
		platforms{profilevalidation.AppleOSiOS: {Introduced: "11.0", Supervised: true}},
		platformsDiff(platforms{
			profilevalidation.AppleOSiOS:   {Introduced: "11.0", Supervised: true},
			profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
		}, parent))
	// Everything is a difference when there's no parent.
	assert.Equal(t, parent, platformsDiff(parent, nil))
}

func TestConvertApplePlatformsInheritance(t *testing.T) {
	payload := platforms{
		profilevalidation.AppleOSiOS:   {Introduced: "4.0"},
		profilevalidation.AppleOSmacOS: {Introduced: "10.7"},
	}
	got, err := convertYAMLWithPlatforms(t, `
- key: Same
  type: <boolean>
  supportedOS:
    iOS: {introduced: '4.0'}
- key: Outer
  type: <dictionary>
  supportedOS:
    iOS: {introduced: '11.0', supervised: true}
  subkeys:
  - key: InheritsOuter
    type: <string>
  - key: DiffersFromOuter
    type: <string>
    supportedOS:
      macOS: {introduced: n/a}
  - key: List
    type: <array>
    subkeys:
    - key: Item
      type: <dictionary>
      subkeys:
      - key: Deep
        type: <string>
        supportedOS:
          iOS: {introduced: '12.0'}
- key: Map
  type: <dictionary>
  supportedOS:
    macOS: {introduced: '13.0'}
  subkeys:
  - key: ANY
    type: <string>
    supportedOS:
      macOS: {introduced: '13.0'}`, payload)
	require.NoError(t, err)

	// Only differences from the enclosing key are recorded, at every level.
	assert.JSONEq(t, `{"open": false, "values": null, "keys": {
		"Same": {"type": "boolean"},
		"Outer": {"type": "dictionary", "platforms": {"iOS": {"introduced": "11.0", "supervised": true}}, "keys": {
			"InheritsOuter": {"type": "string"},
			"DiffersFromOuter": {"type": "string", "platforms": {"macOS": {"introduced": "n/a"}}},
			"List": {"type": "array", "items": {"type": "dictionary", "keys": {
				"Deep": {"type": "string", "platforms": {"iOS": {"introduced": "12.0", "supervised": true}}}}}}}},
		"Map": {"type": "dictionary", "platforms": {"macOS": {"introduced": "13.0"}}, "open": true,
			"values": {"type": "string"}}}}`, got)
}
