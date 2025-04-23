package wlanxml

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const admxPolicy = `&lt;Enabled/&gt;
      <![CDATA[<data id="ExecutionPolicy" value="AllSigned"/>]]>
      <![CDATA[<data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`

func GenerateSingleSSIDTestWLANXMLProfiles(t *testing.T, omitHex, omitName, nonBroadcast bool) string {
	if omitHex && omitName {
		assert.Fail(t, "cannot omit hex and name")
	}
	ssidConfig := WlanXmlProfileSSIDConfig{
		SSID: []WlanXmlProfileSSID{
			{
				Name: "Test",
				Hex:  "54657374",
			},
		},
		NonBroadcast: nonBroadcast,
	}
	if omitHex {
		ssidConfig.SSID[0].Hex = ""
	}
	if omitName {
		ssidConfig.SSID[0].Name = ""
	}
	profile, err := GenerateWLANXMLProfileForTests("SSIDOne", ssidConfig)
	require.NoError(t, err, "Error generating WLAN XML profile")
	return profile
}

func GenerateMultipleSSIDTestWLANXMLProfileVariants(t *testing.T, prefix string, omitName, omitHex, reverseSSIDs, nonBroadcast bool) string {
	if omitHex && omitName {
		assert.Fail(t, "cannot omit hex and name")
	}
	ssidConfig := WlanXmlProfileSSIDConfig{
		SSID: []WlanXmlProfileSSID{
			{
				Name: "SSIDOne",
				Hex:  "535349444F6E65",
			},
			{
				Name: "SSIDTwo",
				Hex:  "5353494454776F",
			},
		},
		SSIDPrefix: WlanXmlProfileSSID{
			Name: prefix,
			Hex:  fmt.Sprintf("%X", prefix),
		},
		NonBroadcast: nonBroadcast,
	}
	if omitHex {
		ssidConfig.SSID[0].Hex = ""
		ssidConfig.SSID[1].Hex = ""
		ssidConfig.SSIDPrefix.Hex = ""
	}
	if omitName {
		ssidConfig.SSID[0].Name = ""
		ssidConfig.SSID[1].Name = ""
		ssidConfig.SSIDPrefix.Name = ""
	}
	if reverseSSIDs {
		slices.Reverse(ssidConfig.SSID)
	}
	profile, err := GenerateWLANXMLProfileForTests("SSIDOne", ssidConfig)
	require.NoError(t, err, "Error generating WLAN XML profile")
	return profile
}

func TestIsWLANXML(t *testing.T) {
	simpleBroadcastingProfile := GenerateSingleSSIDTestWLANXMLProfiles(t, false, false, false)
	simpleNonBroadcastingProfile := GenerateSingleSSIDTestWLANXMLProfiles(t, false, false, true)
	singleNonBroadcastingProfileHexOnly := GenerateSingleSSIDTestWLANXMLProfiles(t, false, true, true)
	singleNonBroadcastingProfileNameOnly := GenerateSingleSSIDTestWLANXMLProfiles(t, false, true, true)

	baseMultiSSIDBroadcastingProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, false, false, false)
	onlyHexSSIDsBroadcastingMultiSSIDProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", true, false, false, false)
	onlyNameSSIDsBroadcastingMultiSSIDProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, true, false, false)

	baseMultiSSIDNonBroadcastingProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, false, false, false)

	t.Parallel()
	tests := map[string]struct {
		input    string
		expected bool
	}{
		"empty string":                     {"", false},
		"not a profile":                    {"not a profile", false},
		"CDATA with invalid content":       {`<![CDATA[bozo]]>`, false},
		"CDATA with invalid XML":           {`<![CDATA[<bozo/>]]>`, false},
		"CDATA with incomplete XML":        {`<![CDATA[<bozo]]>`, false},
		"valid ADMX policy - enabled":      {`<![CDATA[<enabled/>]]>`, false},
		"valid ADMX policy - disabled":     {`<![CDATA[<disabled/>]]>`, false},
		"valid ADMX policy - data element": {`<![CDATA[<data id="id" value="value"/>]]>`, false},
		"valid ADMX policy - multiline": {`	<![CDATA[
				<enabled/>
				]]>`, false},
		"valid ADMX policy - encoded":                        {"&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;", false},
		"valid ADMX policy - constant":                       {admxPolicy, false},
		"valid WLAN XML profile broadcasting SSID":           {simpleBroadcastingProfile, true},
		"valid WLAN XML profile non-broadcasting SSID":       {simpleNonBroadcastingProfile, true},
		"valid WLAN XML profile with only hex SSID":          {singleNonBroadcastingProfileHexOnly, true},
		"valid WLAN XML profile with only name SSID":         {singleNonBroadcastingProfileNameOnly, true},
		"valid WLAN XML profile with multiple SSIDs":         {baseMultiSSIDBroadcastingProfile, true},
		"valid WLAN XML profile with only hex SSIDs":         {onlyHexSSIDsBroadcastingMultiSSIDProfile, true},
		"valid WLAN XML profile with only name SSIDs":        {onlyNameSSIDsBroadcastingMultiSSIDProfile, true},
		"valid WLAN XML profile with non-broadcasting SSIDs": {baseMultiSSIDNonBroadcastingProfile, true},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, IsWLANXML(testCase.input))
		})
	}
}

func TestEqual(t *testing.T) {
	// Each newline separted group below should be equivalent to others in the group
	simpleBroadcastingProfile := GenerateSingleSSIDTestWLANXMLProfiles(t, false, false, false)
	simpleBroadcastingProfileHexOnly := GenerateSingleSSIDTestWLANXMLProfiles(t, false, true, false)
	simpleBroadcastingProfileNameOnly := GenerateSingleSSIDTestWLANXMLProfiles(t, false, true, false)

	simpleNonBroadcastingProfile := GenerateSingleSSIDTestWLANXMLProfiles(t, false, false, true)
	simpleNonBroadcastingProfileHexOnly := GenerateSingleSSIDTestWLANXMLProfiles(t, false, true, true)
	simpleNonBroadcastingProfileNameOnly := GenerateSingleSSIDTestWLANXMLProfiles(t, false, true, true)

	baseMultiSSIDBroadcastingProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, false, false, false)
	baseMultiSSIDBroadcastingProfileReverseSort := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, false, true, false)
	onlyHexSSIDsBroadcastingMultiSSIDProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", true, false, false, false)
	onlyNameSSIDsBroadcastingMultiSSIDProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, true, false, false)

	baseMultiSSIDNonBroadcastingProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "MySSIDPrefix", false, false, false, true)

	alternatePrefixMultiSSIDBroadcastingProfile := GenerateMultipleSSIDTestWLANXMLProfileVariants(t, "BozoSSIDPrefix", false, false, false, false)

	t.Parallel()
	testCases := []struct {
		name, a, b, errorContains string
		equal                     bool
	}{
		{
			name:          "empty profiles",
			a:             "",
			b:             "",
			equal:         false,
			errorContains: "unmarshalling WLAN XML profile",
		},
		{
			name:          "a is an ADMX policy",
			a:             admxPolicy,
			b:             simpleBroadcastingProfile,
			equal:         false,
			errorContains: "unmarshalling WLAN XML profile",
		},
		{
			name:          "b is an ADMX policy",
			a:             simpleBroadcastingProfile,
			b:             admxPolicy,
			equal:         false,
			errorContains: "unmarshalling WLAN XML profile",
		},
		{
			name:          "equal profiles",
			a:             simpleBroadcastingProfile,
			b:             simpleBroadcastingProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal single-SSID profiles but one only includes Hex",
			a:             simpleBroadcastingProfileHexOnly,
			b:             simpleBroadcastingProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal single-SSID profiles but one only includes Name",
			a:             simpleBroadcastingProfileHexOnly,
			b:             simpleBroadcastingProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal single-SSID profiles but one only includes Name and one only includes Hex",
			a:             simpleBroadcastingProfileHexOnly,
			b:             simpleBroadcastingProfileNameOnly,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal non-broadcasting profiles",
			a:             simpleNonBroadcastingProfile,
			b:             simpleNonBroadcastingProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal single-SSID non-broadcast profiles but one only includes Hex",
			a:             simpleNonBroadcastingProfileHexOnly,
			b:             simpleNonBroadcastingProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal single-SSID non-broadcast profiles but one only includes Name",
			a:             simpleNonBroadcastingProfileHexOnly,
			b:             simpleNonBroadcastingProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal single-SSID non-broadcast profiles but one only includes Name and one only includes Hex",
			a:             simpleNonBroadcastingProfileHexOnly,
			b:             simpleNonBroadcastingProfileNameOnly,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "single same-SSID profiles with different non-broadcasting settings",
			a:             simpleBroadcastingProfile,
			b:             simpleNonBroadcastingProfile,
			equal:         false,
			errorContains: "",
		},
		{
			name:          "equal multi-SSID profiles but different SSID order",
			a:             baseMultiSSIDBroadcastingProfile,
			b:             baseMultiSSIDBroadcastingProfileReverseSort,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal multi-SSID profiles but SSIDs as hex for one",
			a:             baseMultiSSIDBroadcastingProfile,
			b:             onlyHexSSIDsBroadcastingMultiSSIDProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal multi-SSID profiles but SSIDs as names for one",
			a:             baseMultiSSIDBroadcastingProfile,
			b:             onlyNameSSIDsBroadcastingMultiSSIDProfile,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "similar profiles with different SSID prefix settings",
			a:             baseMultiSSIDBroadcastingProfile,
			b:             alternatePrefixMultiSSIDBroadcastingProfile,
			equal:         false,
			errorContains: "",
		},
		{
			name:          "similar multi-SSID profiles with different non-broadcasting settings",
			a:             baseMultiSSIDBroadcastingProfile,
			b:             baseMultiSSIDNonBroadcastingProfile,
			equal:         false,
			errorContains: "",
		},
		{
			name:          "single broadcasting SSID compared to multi-SSID broadcasting profile",
			a:             simpleBroadcastingProfile,
			b:             baseMultiSSIDBroadcastingProfile,
			equal:         false,
			errorContains: "",
		},
		{
			name:          "single non-broadcasting SSID compared to multi-SSID non-broadcasting profile",
			a:             simpleNonBroadcastingProfile,
			b:             baseMultiSSIDNonBroadcastingProfile,
			equal:         false,
			errorContains: "",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			equal, err := Equal(tt.a, tt.b)
			if tt.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.errorContains)
			}
			assert.Equal(t, tt.equal, equal)

			// Swap the order of a and b - should be equivalent output
			equal, err = Equal(tt.b, tt.a)
			if tt.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.errorContains)
			}
			assert.Equal(t, tt.equal, equal)
		})
	}
}
