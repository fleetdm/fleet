package wlanxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	// Policy 1 is a pretty simple single SSID policy
	xmlEncodedPolicy1 = `&lt;?xml version=&quot;1.0&quot;?&gt;
&lt;WLANProfile xmlns=&quot;http://www.microsoft.com/networking/WLAN/profile/v1&quot;&gt;
	&lt;name&gt;Test&lt;/name&gt;
	&lt;SSIDConfig&gt;
		&lt;SSID&gt;
                    &lt;hex&gt;54657374&lt;/hex&gt;
                    &lt;name&gt;Test&lt;/name&gt;
                &lt;/SSID&gt;
                &lt;nonBroadcast&gt;false&lt;/nonBroadcast&gt;
	&lt;/SSIDConfig&gt;
	&lt;connectionType&gt;ESS&lt;/connectionType&gt;
	&lt;connectionMode&gt;auto&lt;/connectionMode&gt;
	&lt;MSM&gt;
		&lt;security&gt;
			&lt;authEncryption&gt;
				&lt;authentication&gt;WPA2PSK&lt;/authentication&gt;
				&lt;encryption&gt;AES&lt;/encryption&gt;
				&lt;useOneX&gt;false&lt;/useOneX&gt;
			&lt;/authEncryption&gt;
			&lt;sharedKey&gt;
				&lt;keyType&gt;passPhrase&lt;/keyType&gt;
				&lt;protected&gt;false&lt;/protected&gt;
				&lt;keyMaterial&gt;sup3rs3cr3t&lt;/keyMaterial&gt;
			&lt;/sharedKey&gt;
		&lt;/security&gt;
	&lt;/MSM&gt;
&lt;/WLANProfile&gt;`

	// Policy 2 is a variant of policy 1 with a non-broadcast SSID
	xmlEncodedPolicy2 = `&lt;?xml version=&quot;1.0&quot;?&gt;
&lt;WLANProfile xmlns=&quot;http://www.microsoft.com/networking/WLAN/profile/v1&quot;&gt;
	&lt;name&gt;Test&lt;/name&gt;
	&lt;SSIDConfig&gt;
		&lt;SSID&gt;
                    &lt;hex&gt;54657374&lt;/hex&gt;
                    &lt;name&gt;Test&lt;/name&gt;
                &lt;/SSID&gt;
                &lt;nonBroadcast&gt;true&lt;/nonBroadcast&gt;
	&lt;/SSIDConfig&gt;
	&lt;connectionType&gt;ESS&lt;/connectionType&gt;
	&lt;connectionMode&gt;auto&lt;/connectionMode&gt;
	&lt;MSM&gt;
		&lt;security&gt;
			&lt;authEncryption&gt;
				&lt;authentication&gt;WPA2PSK&lt;/authentication&gt;
				&lt;encryption&gt;AES&lt;/encryption&gt;
				&lt;useOneX&gt;false&lt;/useOneX&gt;
			&lt;/authEncryption&gt;
			&lt;sharedKey&gt;
				&lt;keyType&gt;passPhrase&lt;/keyType&gt;
				&lt;protected&gt;false&lt;/protected&gt;
				&lt;keyMaterial&gt;sup3rs3cr3t&lt;/keyMaterial&gt;
			&lt;/sharedKey&gt;
		&lt;/security&gt;
	&lt;/MSM&gt;
&lt;/WLANProfile&gt;`

	// Policy 3 is a more complex policy with multiple SSIDs
	xmlEncodedPolicy3 = `&lt;WLANProfile xmlns=&quot;http://www.microsoft.com/networking/CarrierControl/WLAN/v1&quot;
             xmlns:v2=&quot;http://www.microsoft.com/networking/CarrierControl/WLAN/v2&quot;&gt;
  &lt;name&gt;SampleProfile&lt;/name&gt;
  &lt;SSIDConfig&gt;
    &lt;SSID&gt;
        &lt;name&gt;MySSID1&lt;/name&gt;
    &lt;/SSID&gt;
    &lt;v2:SSID&gt;
        &lt;v2:name&gt;MySSID2&lt;/v2:name&gt;
    &lt;/v2:SSID&gt;
    &lt;v2:SSIDPrefix&gt;
        &lt;v2:name&gt;MySSIDPrefix&lt;/v2:name&gt;
    &lt;/v2:SSIDPrefix&gt;
  &lt;/SSIDConfig&gt;
  &lt;MSM&gt;
    &lt;security&gt;
        &lt;authEncryption&gt;
            &lt;authentication&gt;open&lt;/authentication&gt;
            &lt;encryption&gt;none&lt;/encryption&gt;
            &lt;useOneX&gt;false&lt;/useOneX&gt;
        &lt;/authEncryption&gt;
    &lt;/security&gt;
  &lt;/MSM&gt;
&lt;/WLANProfile&gt;`

	xmlEncodedPolicy3Variant = `&lt;WLANProfile xmlns=&quot;http://www.microsoft.com/networking/CarrierControl/WLAN/v1&quot;
             xmlns:v2=&quot;http://www.microsoft.com/networking/CarrierControl/WLAN/v2&quot;&gt;
  &lt;name&gt;SampleProfile&lt;/name&gt;
  &lt;SSIDConfig&gt;
    &lt;v2:SSID&gt;
        &lt;v2:name&gt;MySSID2&lt;/v2:name&gt;
    &lt;/v2:SSID&gt;
    &lt;SSID&gt;
        &lt;name&gt;MySSID1&lt;/name&gt;
    &lt;/SSID&gt;
    &lt;v2:SSIDPrefix&gt;
        &lt;v2:name&gt;MySSIDPrefix&lt;/v2:name&gt;
    &lt;/v2:SSIDPrefix&gt;
  &lt;/SSIDConfig&gt;
  &lt;MSM&gt;
    &lt;security&gt;
        &lt;authEncryption&gt;
            &lt;authentication&gt;open&lt;/authentication&gt;
            &lt;encryption&gt;none&lt;/encryption&gt;
            &lt;useOneX&gt;false&lt;/useOneX&gt;
        &lt;/authEncryption&gt;
    &lt;/security&gt;
  &lt;/MSM&gt;
&lt;/WLANProfile&gt;`

	admxPolicy = `&lt;Enabled/&gt;
      <![CDATA[<data id="ExecutionPolicy" value="AllSigned"/>]]>
      <![CDATA[<data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`
)

func TestIsWLANXML(t *testing.T) {
	t.Parallel()
	assert.False(t, IsWLANXML(""))
	assert.False(t, IsWLANXML("not a policy"))
	assert.False(t, IsWLANXML(`<![CDATA[bozo]]>`))
	assert.False(t, IsWLANXML(`<![CDATA[<bozo/>]]>`))
	assert.False(t, IsWLANXML(`<![CDATA[<bozo]]>`))

	// These are all valid ADMX policies but not WLAN XML policies
	assert.False(t, IsWLANXML(`<![CDATA[<enabled/>]]>`))
	assert.False(t, IsWLANXML(`<![CDATA[<disabled/>]]>`))
	assert.False(t, IsWLANXML(`<![CDATA[<data id="id" value="value"/>]]>`))
	assert.False(t, IsWLANXML(
		`	<![CDATA[
				<enabled/>
				]]>`))
	assert.False(t,
		IsWLANXML("&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;"))
	assert.False(t, IsWLANXML(admxPolicy))

	assert.True(t, IsWLANXML(xmlEncodedPolicy1))
}

func TestEqual(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name, a, b, errorContains string
		equal                     bool
	}{
		{
			name:          "empty policies",
			a:             "",
			b:             "",
			equal:         false,
			errorContains: "unmarshalling WLAN XML policy",
		},
		{
			name:          "a is an ADMX policy",
			a:             admxPolicy,
			b:             xmlEncodedPolicy1,
			equal:         false,
			errorContains: "unmarshalling WLAN XML policy",
		},
		{
			name:          "b is an ADMX policy",
			a:             xmlEncodedPolicy1,
			b:             admxPolicy,
			equal:         false,
			errorContains: "unmarshalling WLAN XML policy",
		},
		{
			name:          "equal policies",
			a:             xmlEncodedPolicy1,
			b:             xmlEncodedPolicy1,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal policies but different SSID order",
			a:             xmlEncodedPolicy3,
			b:             xmlEncodedPolicy3Variant,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "equal policies but different SSID order - swapped invocation order",
			a:             xmlEncodedPolicy3Variant,
			b:             xmlEncodedPolicy3,
			equal:         true,
			errorContains: "",
		},
		{
			name:          "different policies",
			a:             xmlEncodedPolicy1,
			b:             xmlEncodedPolicy2,
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
		})
	}
}
