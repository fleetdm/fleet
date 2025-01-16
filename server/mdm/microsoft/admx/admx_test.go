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
	assert.True(t, IsADMX(`<![CDATA[<enabled/>]]>`))
	assert.True(t, IsADMX(`<![CDATA[<disabled/>]]>`))
	assert.True(t, IsADMX(`<![CDATA[<data id="id" value="value"/>]]>`))
	assert.True(t, IsADMX(
		`	<![CDATA[
				<enabled/>
				]]>`))
	assert.True(t,
		IsADMX("&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;"))
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
			name: "disabled policies with data",
			a: `<![CDATA[<disabled/>
				<data id="ExecutionPolicy" value="AllSigned"/>
				<data id="SourcePathForUpdateHelp" value="false"/>]]>`,
			b:             "&lt;Disabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			equal:         true,
			errorContains: "",
		},
		{
			name:          "unparsable policy",
			a:             "",
			b:             "<bozo",
			equal:         false,
			errorContains: "unmarshalling ADMX policy",
		},
		{
			name:          "multiple CDATA",
			a:             "<![CDATA[<enabled/>]]><![CDATA[<bozo>]]>",
			b:             "",
			equal:         false,
			errorContains: "multiple CDATA matches found",
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
