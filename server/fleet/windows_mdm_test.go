package fleet

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/stretchr/testify/require"
)

func TestValidateUserProvided(t *testing.T) {
	tests := []struct {
		name                 string
		profile              MDMWindowsConfigProfile
		allowCustomOSUpdates bool
		wantErr              string
	}{
		{
			name: "Valid XML with Replace",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: "",
		},
		{
			name: "Invalid Platform",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<SyncML xmlns="SYNCML:SYNCML1.2">
  <Replace>
    <Item>
      <Target><LocURI>./Device/Custom/URI</LocURI></Target>
    </Item>
  </Replace>
</SyncML>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "Add top level element",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Add>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
</Add>
`),
			},
			wantErr: "",
		},
		{
			name: "Reserved LocURI",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Vendor/MSFT/BitLocker/Foo</LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: syncml.DiskEncryptionProfileRestrictionErrMsg,
		},
		{
			name: "Reserved LocURI with implicit ./Device prefix",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Vendor/MSFT/BitLocker/Foo</LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: syncml.DiskEncryptionProfileRestrictionErrMsg,
		},
		{
			name: "XML with Multiple Replace Elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI1</LocURI></Target>
  </Item>
</Replace>
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI2</LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: "",
		},
		{
			name: "Empty XML",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(``),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "XML with Multiple Replace Elements, One with Reserved LocURI",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
</Replace>
<Replace>
  <Item>
    <Target><LocURI>./Device/Vendor/MSFT/BitLocker/Bar</LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: syncml.DiskEncryptionProfileRestrictionErrMsg,
		},
		{
			name: "XML with Mixed Replace and Add",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
</Replace>
<Add>
  <Item>
    <Target><LocURI>./Device/Another/URI</LocURI></Target>
  </Item>
</Add>
`),
			},
			wantErr: "",
		},
		{
			name: "XML with Replace and Alert",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
 <Item>
   <Target><LocURI>./Device/Replace/URI</LocURI></Target>
 </Item>
</Replace>
<Alert>
 <Item>
   <Target><LocURI>./Device/Alert/URI</LocURI></Target>
 </Item>
</Alert>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with Replace and Atomic",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Atomic>
  <Item>
    <Target><LocURI>./Device/Atomic/URI</LocURI></Target>
  </Item>
</Atomic>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with Replace and Delete",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Delete>
  <Item>
    <Target><LocURI>./Device/Delete/URI</LocURI></Target>
  </Item>
</Delete>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with Replace and Exec",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Exec>
  <Item>
    <Target><LocURI>./Device/Exec/URI</LocURI></Target>
  </Item>
</Exec>
`),
			},
			wantErr: "Only SCEP profiles can include <Exec> elements.",
		},
		{
			name: "XML with Replace and Get",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Get>
  <Item>
    <Target><LocURI>./Device/Get/URI</LocURI></Target>
  </Item>
</Get>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with Replace and Results",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Results>
  <Item>
    <Target><LocURI>./Device/Results/URI</LocURI></Target>
  </Item>
</Results>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with Replace and Status",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Status>
  <Item>
    <Target><LocURI>./Device/Status/URI</LocURI></Target>
  </Item>
</Status>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with elements not defined in the protocol",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
</Replace>
<Foo>
  <Item>
    <Target><LocURI>./Device/Another/URI</LocURI></Target>
  </Item>
</Foo>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "invalid XML with mismatched tags",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
</Add>
`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "invalid XML with unclosed root tag",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
  </Item>
`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "invalid XML with unclosed nested tag",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</LocURI></Target>
</Replace>
`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "invalid XML with overlapping elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</Target></LocURI>
  </Item>
</Replace>
`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "invalid XML with duplicate attributes",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</Target></LocURI>
    <Data attr="1" attr="2"></Data>
  </Item>
</Replace>
`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "invalid XML with special chars",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>./Device/Custom/URI</Target></LocURI>
    <Data>Invalid & Data</Data>
  </Item>
</Replace>
`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "empty LocURI",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI></LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: "",
		},
		{
			name: "item without target",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
  </Item>
</Replace>
`),
			},
			wantErr: "",
		},
		{
			name: "no items in Replace",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
</Replace>
`),
			},
			wantErr: "",
		},
		{
			name: "Valid XML with reserved name",
			profile: MDMWindowsConfigProfile{
				Name:   mdm.FleetWindowsOSUpdatesProfileName,
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Custom/URI</LocURI></Target></Replace>`),
			},
			wantErr: `Profile name "Windows OS Updates" is not allowed`,
		},
		{
			name: "Valid XML with reserved name but experimental allow custom OS updates flag enabled is still not allowed",
			profile: MDMWindowsConfigProfile{
				Name:   mdm.FleetWindowsOSUpdatesProfileName,
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Custom/URI</LocURI></Target></Replace>`),
			},
			allowCustomOSUpdates: true,
			wantErr:              `Profile name "Windows OS Updates" is not allowed`,
		},
		{
			name: "Valid XML with Windows Update LocURI without experimental allow custom OS updates flag enabled is blocked",
			profile: MDMWindowsConfigProfile{
				Name:   "FleetieUpdater",
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Update/something</LocURI></Target></Replace>`),
			},
			allowCustomOSUpdates: false,
			wantErr:              "Custom configuration profiles can't include Windows updates settings. To control these settings, use the mdm.windows_updates option.",
		},
		{
			name: "Valid XML with Windows Update LocURI but experimental allow custom OS updates flag enabled is allowed",
			profile: MDMWindowsConfigProfile{
				Name:   "FleetieUpdater",
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Update/something</LocURI></Target></Replace>`),
			},
			allowCustomOSUpdates: true,
			wantErr:              "",
		},
		{
			name: "Valid XML with Bitlocker LocURI without experimental allow custom OS updates flag enabled is blocked",
			profile: MDMWindowsConfigProfile{
				Name:   "FleetieUpdater",
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Vendor/MSFT/BitLocker/something</LocURI></Target></Replace>`),
			},
			allowCustomOSUpdates: false,
			wantErr:              "Couldn't add. The configuration profile can't include BitLocker settings.",
		},
		{
			name: "Valid XML with Bitlocker LocURI without experimental allow custom OS updates flag enabled is blocked",
			profile: MDMWindowsConfigProfile{
				Name:   "FleetieUpdater",
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Vendor/MSFT/BitLocker/something</LocURI></Target></Replace>`),
			},
			allowCustomOSUpdates: true,
			wantErr:              "Couldn't add. The configuration profile can't include BitLocker settings.",
		},
		{
			name: "XML with top level comment",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <!-- this is a comment -->
				  <!-- this is another comment -->
				  <Replace>
				  <!-- this is a comment inside replace -->
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				    </Target>
				  </Replace>
				`),
			},
			wantErr: "",
		},
		{
			name: "XML with nested root element in data",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <Item>
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				      <Data>
				        <?xml version="1.0"?>
					<Foo></Foo>
				      </Data>
				    </Target>
				    </Item>
				  </Replace>
				`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "XML with nested root element under Replace",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <?xml version="1.0"?>
				    <Item>
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				      <Data>
					<Foo></Foo>
				      </Data>
				    </Target>
				    </Item>
				  </Replace>
				`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "XML with root element above Replace",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <?xml version="1.0"?>
				  <Replace>
				  <Item>
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				      <Data>
					<Foo></Foo>
				      </Data>
				    </Target>
				    </Item>
				  </Replace>
				`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "XML with root element inside Target",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <Target>
				      <?xml version="1.0"?>
				      <LocURI>./Device/Custom/URI</LocURI>
				      <Data>
					<Foo></Foo>
				      </Data>
				    </Target>
				  </Replace>
				`),
			},
			wantErr: "The file should include valid XML",
		},
		{
			name: "XML with CDATA used to embed xml",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				      <Data>
				      <![CDATA[
				        <?xml version="1.0"?>
					<Foo></Foo>
				      ]]>
				      </Data>
				    </Target>
				  </Replace>
				`),
			},
			wantErr: "",
		},
		{
			name: "XML escaped with nested root element",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				      <Data>
				        &lt;?xml version=&quot;1.0&quot;?&gt;
                                        &lt;name&gt;Wireless Network&lt;/name&gt;
				      </Data>
				    </Target>
				  </Replace>
				`),
			},
			wantErr: "",
		},
		{
			name: "SCEP profile with other LocURIs",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <Target>
				      <LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI>
				    </Target>
				  </Replace>
				  <Replace>
				    <Target>
				      <LocURI>./Device/Custom/URI</LocURI>
				    </Target>
				  </Replace>
				  <Exec>
				    <Item>
				      <Target>
				        <LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI>
				      </Target>
				    </Item>
				  </Exec>
				`),
			},
			wantErr: "Only options that have <LocURI> starting with \"ClientCertificateInstall/SCEP/\" can be added to SCEP profile.",
		},
		{
			name: "SCEP profile without Exec block",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <Replace>
				    <Target>
				      <LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI>
				    </Target>
				  </Replace>
				`),
			},
			wantErr: "\"ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll\" must be included within <Exec>. Please add and try again.",
		},
		{
			name: "SCEP profile with Exec block, but wrong LocURI ",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Replace>
					<Target>
						<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI>
					</Target>
				</Replace>
				<Exec>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Random/Scep/LocURI</LocURI>
						</Target>
					</Item>
				</Exec>
				`),
			},
			wantErr: "\"ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll\" must be included within <Exec>. Please add and try again.",
		},
		{
			name: "SCEP profile with multiple Exec blocks, but one has wrong loc URI",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Replace>
					<Target>
						<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI>
					</Target>
				</Replace>
				<Exec>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI>
						</Target>
					</Item>
				</Exec>
				<Exec>
					<Item>
						<Target>
							<LocURI>./Device/Test</LocURI>
						</Target>
					</Item>
				</Exec>
				`),
			},
			wantErr: "SCEP profiles must include exactly one <Exec> element.",
		},
		{
			name: fmt.Sprintf("SCEP profile with missing $FLEET_VAR_%s after SCEP LocURI", FleetVarSCEPWindowsCertificateID),
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Add>
					<CmdID>12</CmdID>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/bogus-id-that-is-not-fleet-var/Install/CAThumbprint</LocURI>
						</Target>
						<Meta>
							<Format xmlns="syncml:metinf">chr</Format>
						</Meta>
						<Data>0DE4135C02E5E3C040FE1353E204D8B6F331F47A</Data>
					</Item>
				</Add>
				`),
			},
			wantErr: fmt.Sprintf("You must use \"$FLEET_VAR_%s\" after \"ClientCertificateInstall/SCEP/\".", FleetVarSCEPWindowsCertificateID),
		},
		{
			name: "SCEP Profile with missing required LocURI",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Add>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI>
						</Target>
					</Item>
				</Add>
				<Exec>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI>
						</Target>
					</Item>
				</Exec>
				`),
			},
			wantErr: fmt.Sprintf("\"ClientCertificateInstall/SCEP/$FLEET_VAR_%s/Install/CAThumbprint\" is missing. Please add and try again.", FleetVarSCEPWindowsCertificateID),
		},
		{
			name: "Only SCEP profiles can have Exec elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Exec>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/CustomExecTargetLocURI</LocURI>
						</Target>
					</Item>
				</Exec>
				`),
			},
			wantErr: "Only SCEP profiles can include <Exec> elements.",
		},
		{
			name: "Either device or user SCEP profiles, not both",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Replace>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI>
						</Target>
					</Item>
				</Replace>
				<Exec>
					<Item>
						<Target>
							<LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI>
						</Target>
					</Item>
				</Exec>
				`),
			},
			wantErr: "All <LocURI> elements in the SCEP profile must start either with \"./Device\" or \"./User\".",
		},
		{
			name: fmt.Sprintf("SCEP profile with ${FLEET_VAR_%s} after SCEP LocURI", FleetVarSCEPWindowsCertificateID),
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Add>
					<CmdID>12</CmdID>
					<Item>
						<Target>
							<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/${FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID}/Install/CAThumbprint</LocURI>
						</Target>
						<Meta>
							<Format xmlns="syncml:metinf">chr</Format>
						</Meta>
						<Data>0DE4135C02E5E3C040FE1353E204D8B6F331F47A</Data>
					</Item>
				</Add>
				`),
			},
			wantErr: fmt.Sprintf("\"ClientCertificateInstall/SCEP/%s/Install/Enroll\" must be included within <Exec>. Please add and try again.", FleetVarSCEPWindowsCertificateID.WithPrefix()),
		},
		{
			name: "Atomic profile with other top-level elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Atomic>
				</Atomic>
				<Add>
				</Add>
				`),
			},
			wantErr: "<Atomic> element must wrap all the elements in a Windows configuration profile.",
		},
		{
			name: "non Atomic profile with other <Atomic> top-level elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Add>
				</Add>
				<Atomic>
				</Atomic>
				`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "disallow top-level Delete element",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Delete>
				</Delete>
				`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "disallow top-level Get element",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Get>
				</Get>
				`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "disallow Delete element inside Atomic profile",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Atomic>
					<Delete>
					</Delete>
				</Atomic>
				`),
			},
			wantErr: "Windows configuration profiles can only include <Replace> or <Add> within the <Atomic> element.",
		},
		{
			name: "disallow top-level Get element inside Atomic profile",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Atomic>
					<Get>
					</Get>
				</Atomic>
				`),
			},
			wantErr: "Windows configuration profiles can only include <Replace> or <Add> within the <Atomic> element.",
		},
		{
			name: "valid Atomic profile",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				<Atomic>
					<Add>
						<LocURI>./Device/Custom/URI</LocURI>
					</Add>
					<Replace>
						<LocURI>./Device/Another/URI</LocURI>
					</Replace>
				</Atomic>
				`),
			},
			wantErr: "",
		},
		{
			name: "plain text non-XML content is rejected (#42219)",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte("this is not xml"),
			},
			wantErr: "The file should include valid SyncML XML with at least one supported element.",
		},
		{
			name: "XML with only comments is rejected",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<!-- just a comment --><!-- another -->`),
			},
			wantErr: "The file should include valid SyncML XML with at least one supported element.",
		},
		{
			name: "LocURI missing ./ prefix is rejected (#42224)",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <CmdID>1</CmdID>
  <Item>
    <Target><LocURI>Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock</LocURI></Target>
    <Data>5</Data>
  </Item>
</Replace>
`),
			},
			wantErr: `<LocURI> must start with "./Device/", "./User/", or "./Vendor/".`,
		},
		{
			name: "LocURI with leading single slash is rejected",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>/Vendor/MSFT/BitLocker/Foo</LocURI></Target></Replace>`),
			},
			wantErr: `<LocURI> must start with "./Device/", "./User/", or "./Vendor/".`,
		},
		{
			name: "LocURI with unknown root is rejected",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>./Custom/Foo</LocURI></Target></Replace>`),
			},
			wantErr: `<LocURI> must start with "./Device/", "./User/", or "./Vendor/".`,
		},
		{
			name: "LocURI with ../ path traversal is rejected (#42224)",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <CmdID>1</CmdID>
  <Item>
    <Target><LocURI>./Device/Vendor/../../etc/passwd</LocURI></Target>
    <Data>test</Data>
  </Item>
</Replace>
`),
			},
			wantErr: `<LocURI> can't contain ".." path traversal segments.`,
		},
		{
			name: "LocURI with trailing .. segment is rejected",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Vendor/..</LocURI></Target></Replace>`),
			},
			wantErr: `<LocURI> can't contain ".." path traversal segments.`,
		},
		{
			name: "LocURI with implicit ./Vendor prefix is allowed",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>./Vendor/MSFT/Foo/Bar</LocURI></Target></Replace>`),
			},
			wantErr: "",
		},
		{
			name: "LocURI with ./User prefix is allowed",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>./User/Vendor/MSFT/Foo</LocURI></Target></Replace>`),
			},
			wantErr: "",
		},
		{
			name: "LocURI with surrounding whitespace is allowed",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>  ./Device/Custom/URI  </LocURI></Target></Replace>`),
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.ValidateUserProvided(tt.allowCustomOSUpdates)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
