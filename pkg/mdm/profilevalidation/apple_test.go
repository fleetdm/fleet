package profilevalidation

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractExampleDecodes(t *testing.T) {
	raw, err := os.ReadFile("testdata/contract_example.json")
	require.NoError(t, err)

	// Unknown fields (version.future_platform, a Windows node's future_field) must be ignored.
	var s ProfileValidationSchema
	require.NoError(t, json.Unmarshal(raw, &s))

	assert.Equal(t, 1, s.FormatVersion)
	assert.Equal(t, AppleVersion{Release: "Release-v27.0", Commit: "0123456789abcdef0123456789abcdef01234567"}, s.Version.Apple)
	assert.Equal(t, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), s.Version.GeneratedAt)

	require.NotNil(t, s.AppleEnvelope.TopLevel)
	require.NotNil(t, s.AppleEnvelope.CommonPayloadKeys)
	assert.True(t, notNil(t, s.AppleEnvelope.TopLevel.Keys["PayloadVersion"]).Required)
	assert.Equal(t, AppleTypes{AppleTypeString}, notNil(t, s.AppleEnvelope.CommonPayloadKeys.Keys["PayloadUUID"]).Type)
	assert.NotContains(t, s.AppleProfiles, "TopLevel")
	require.NotNil(t, s.AppleDeclarationEnvelope)
	assert.True(t, notNil(t, s.AppleDeclarationEnvelope.Keys["Identifier"]).Required)
	assert.Equal(t, "16.0", s.AppleDeclarationEnvelope.Platforms[AppleOStvOS].Introduced)

	access := notNil(t, s.AppleProfiles["com.apple.applicationaccess"])
	assert.Equal(t, "apple", access.Source)
	assert.Equal(t, "10.7", access.Platforms[AppleOSmacOS].Introduced)
	assert.Equal(t, ApplePlatform{Introduced: "11.0", Supervised: true}, notNil(t, access.Keys["allowAirPrint"]).Platforms[AppleOSiOS])
	assert.Equal(t, "n/a", notNil(t, access.Keys["allowAirPrint"]).Platforms[AppleOSmacOS].Introduced)
	assert.Nil(t, notNil(t, access.Keys["allowCamera"]).Platforms)
	assert.Equal(t, AppleTypes{AppleTypeString}, notNil(t, access.Keys["autonomousSingleAppModePermittedAppIDs"]).Items.Type)
	delay := notNil(t, access.Keys["enforcedSoftwareUpdateDelay"])
	assert.Equal(t, &Range{Min: new(1.0), Max: new(90.0)}, delay.Range)
	assert.Equal(t, ApplePlatform{Introduced: "10.13.4", Deprecated: "26.0", Removed: "27.0"}, delay.Platforms[AppleOSmacOS])

	assert.Equal(t, AppleTypes{AppleTypeInteger, AppleTypeString}, notNil(t, notNil(t, s.AppleProfiles["com.apple.example.override"]).Keys["SomeKey"]).Type)
	assert.NotEmpty(t, notNil(t, notNil(t, s.AppleProfiles["com.apple.eas.account"]).Keys["PayloadCertificateUUID"]).Format)
	assert.True(t, notNil(t, notNil(t, s.AppleProfiles["com.apple.ManagedClient.preferences"]).Keys["PayloadContent"]).Open)

	dock := notNil(t, notNil(t, s.AppleProfiles["com.apple.homescreenlayout"]).Keys["Dock"])
	assert.Equal(t, []any{"Application", "Folder", "WebClip"}, notNil(t, dock.Items.Keys["Type"]).RangeList)
	assert.Equal(t, "Dock[]", notNil(t, dock.Items.Keys["Pages"]).Items.Items.Ref)

	passcode := notNil(t, s.AppleDeclarations["com.apple.configuration.passcode.settings"])
	// A 0 bound must survive, not be confused with "no bound".
	require.NotNil(t, notNil(t, passcode.Keys["MinimumLength"]).Range.Min)
	assert.Zero(t, *notNil(t, passcode.Keys["MinimumLength"]).Range.Min)
	assert.True(t, notNil(t, notNil(t, passcode.Keys["CustomRegex"]).Keys["Regex"]).Required)
	// Windows sections are checked in TestWindowsContractExample.

	// Re-encoding and decoding again is lossless for everything modeled.
	out, err := json.Marshal(s)
	require.NoError(t, err)
	var again ProfileValidationSchema
	require.NoError(t, json.Unmarshal(out, &again))
	assert.Equal(t, s, again)
}

func TestAppleKeyEncoding(t *testing.T) {
	// Optional fields are omitted, a single type is a string, and a 0 bound is kept.
	out, err := json.Marshal(&AppleKey{Type: AppleTypes{AppleTypeInteger}, Range: &Range{Min: new(0.0)}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type": "integer", "range": {"min": 0}}`, string(out))

	out, err = json.Marshal(&AppleKey{Type: AppleTypes{AppleTypeInteger, AppleTypeString}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type": ["integer", "string"]}`, string(out))
}

func TestAppleTypesUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    AppleTypes
		wantErr bool
	}{
		{name: "single", in: `"boolean"`, want: AppleTypes{AppleTypeBoolean}},
		{name: "list", in: `["integer","string"]`, want: AppleTypes{AppleTypeInteger, AppleTypeString}},
		// Newer upstream data must never fail an older parser.
		{name: "unknown value carried", in: `"<future>"`, want: AppleTypes{"<future>"}},
		{name: "number rejected", in: `1`, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got AppleTypes
			err := json.Unmarshal([]byte(c.in), &got)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestUnknownFieldsIgnored(t *testing.T) {
	var k AppleKey
	require.NoError(t, json.Unmarshal([]byte(`{"type": "string", "valuetype": "<future>", "someNewField": {"x": 1}}`), &k))
	assert.Equal(t, AppleTypes{AppleTypeString}, k.Type)
	assert.Equal(t, AppleValueType("<future>"), k.ValueType)
}

// notNil fails the test if v is nil and returns it, so a missing schema entry fails cleanly
// instead of panicking (and nilaway can see the lookup is guarded).
func notNil[T any](t *testing.T, v *T) *T {
	t.Helper()
	require.NotNil(t, v)
	return v
}
