package profilevalidation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeWindowsProfile(t *testing.T) {
	items, err := decodeWindowsProfile([]byte(`
<Replace>
	<CmdID>1</CmdID>
	<Item>
		<Meta><Format xmlns="syncml:metinf">int</Format></Meta>
		<Target><LocURI> ./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength </LocURI></Target>
		<Data>8</Data>
	</Item>
</Replace>
<Add>
	<Item>
		<Meta><Format>chr</Format></Meta>
		<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Chrome~Policy~googlechrome/HomepageLocation</LocURI></Target>
		<Data><![CDATA[<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>]]></Data>
	</Item>
</Add>
<Atomic>
	<Exec>
		<Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI></Target></Item>
	</Exec>
	<Replace>
		<Item>
			<Target><LocURI>./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall</LocURI></Target>
			<Data></Data>
		</Item>
		<Item><Source><LocURI>./Vendor/MSFT/NoTarget</LocURI></Source><Data>x</Data></Item>
		<Item>
			<Target><LocURI>./Vendor/MSFT/Example/Xml</LocURI></Target>
			<Data><Rules><Rule id="1"/></Rules></Data>
		</Item>
	</Replace>
</Atomic>`))
	require.NoError(t, err)
	assert.Equal(t, []windowsItem{
		{LocURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength", MetaFormat: "int", Data: new("8")},
		{LocURI: "./Device/Vendor/MSFT/Policy/Config/Chrome~Policy~googlechrome/HomepageLocation", MetaFormat: "chr", Data: new(`<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>`)},
		{LocURI: "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll"},
		{LocURI: "./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", Data: new("")},
		// Raw child elements are already well-formed; only the element's own text is kept.
		{LocURI: "./Vendor/MSFT/Example/Xml", Data: new("")},
	}, items)
}

func TestDecodeWindowsProfileEdges(t *testing.T) {
	// A body that's only a secret placeholder expands at apply time; it has nothing to check.
	items, err := decodeWindowsProfile([]byte(`$FLEET_SECRET_WINDOWS_PROFILE`))
	require.NoError(t, err)
	assert.Empty(t, items)

	for name, data := range map[string]string{
		"unclosed":   `<Replace><Item><Target><LocURI>./Vendor/MSFT/X</LocURI></Target></Item>`,
		"mismatched": `<Replace><Item></Replace></Item>`,
		"bad entity": `<Replace><Item><Data>&nope;</Data></Item></Replace>`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decodeWindowsProfile([]byte(data))
			require.Error(t, err)
		})
	}
}
