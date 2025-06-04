package admx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsADMX(t *testing.T) {
	t.Parallel()
	assert.False(t, IsADMX(""))
	assert.False(t, IsADMX("not an ADMX policy"))
	assert.False(t, IsADMX(`<![CDATA[bozo]]>`))
	assert.False(t, IsADMX(`<![CDATA[<bozo/>]]>`))
	assert.False(t, IsADMX(`<![CDATA[<bozo]]>`))

	// This is a WLAN XML profile
	assert.False(t, IsADMX(
		`&lt;?xml version=&quot;1.0&quot;?&gt;
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
&lt;/WLANProfile&gt;`))

	assert.True(t, IsADMX(`<![CDATA[<enabled/>]]>`))
	assert.True(t, IsADMX(`<![CDATA[<disabled/>]]>`))
	assert.True(t, IsADMX(`<![CDATA[<data id="id" value="value"/>]]>`))
	assert.True(t, IsADMX(
		`	<![CDATA[
				<enabled/>
				]]>`))
	assert.True(t,
		IsADMX("&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;"))
	assert.True(t, IsADMX(
		`&lt;Enabled/&gt;
      <![CDATA[<data id="ExecutionPolicy" value="AllSigned"/>]]>
      <![CDATA[<data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`))
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
			equal:         true,
			errorContains: "",
		},
		{
			name:          "enabled policies",
			a:             "<![CDATA[<enabled/>]]>",
			b:             "&lt;Enabled/&gt;",
			equal:         true,
			errorContains: "",
		},
		{
			name:          "disabled policies",
			a:             "<![CDATA[<disabled/>]]>",
			b:             "&lt;Disabled/&gt;",
			equal:         true,
			errorContains: "",
		},
		{
			name:          "unequal policies",
			a:             "<![CDATA[<disabled/>]]>",
			b:             "&lt;enabled/&gt;",
			equal:         false,
			errorContains: "",
		},
		{
			name: "enabled policies with data",
			a: `<![CDATA[<enabled/>
				<data id="ExecutionPolicy" value="AllSigned"/>
				<data id="Listbox_ModuleNames" value="*"/>
				<data id="OutputDirectory" value="false"/>
				<data id="EnableScriptBlockInvocationLogging" value="true"/>
				<data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			equal:         true,
			errorContains: "",
		},
		{
			name: "enabled policies with data and nonstandard format",
			a: `&lt;Enabled/&gt;
      <![CDATA[<data id="ExecutionPolicy" value="AllSigned"/>]]>
      <![CDATA[<data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			equal:         true,
			errorContains: "",
		},
		{
			name: "disabled policies with data",
			a: `<![CDATA[<disabled/>
				<data id="ExecutionPolicy" value="AllSigned"/>
				<data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Disabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			equal:         true,
			errorContains: "",
		},
		{
			name:          "unparsable policy a 1",
			a:             "<bozo",
			b:             "",
			equal:         false,
			errorContains: "unmarshalling ADMX policy",
		},
		{
			name:          "unparsable policy a 2",
			a:             "&lt;bozo",
			b:             "",
			equal:         false,
			errorContains: "unmarshalling ADMX policy",
		},
		{
			name:          "unparsable policy b 1",
			a:             "",
			b:             "<bozo",
			equal:         false,
			errorContains: "unmarshalling ADMX policy",
		},
		{
			name:          "unparsable policy b 2",
			a:             "",
			b:             "&lt;bozo",
			equal:         false,
			errorContains: "unmarshalling ADMX policy",
		},
		{
			name: "unequal policies with missing enable",
			a: `<![CDATA[<Xenabled/>
				<data id="ExecutionPolicy" value="AllSigned"/>
				<data id="Listbox_ModuleNames" value="*"/>
				<data id="OutputDirectory" value="false"/>
				<data id="EnableScriptBlockInvocationLogging" value="true"/>
				<data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			equal:         false,
			errorContains: "",
		},
		{
			name: "unequal policies with data 1",
			a: `<![CDATA[<enabled/>
				<data id="EnableScriptBlockInvocationLogging" value="true"/>
				<data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			equal:         false,
			errorContains: "",
		},
		{
			name: "unequal policies with data 2",
			a: `<![CDATA[<enabled/>
				<data id="ExecutionPolicy" value="XXXX"/>
				<data id="Listbox_ModuleNames" value="*"/>
				<data id="OutputDirectory" value="false"/>
				<data id="EnableScriptBlockInvocationLogging" value="true"/>
				<data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
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
