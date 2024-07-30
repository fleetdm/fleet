package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/stretchr/testify/require"
)

func TestValidateUserProvided(t *testing.T) {
	tests := []struct {
		name    string
		profile MDMWindowsConfigProfile
		wantErr string
	}{
		{
			name: "Valid XML with Replace",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>Custom/URI</LocURI></Target>
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
      <Target><LocURI>Custom/URI</LocURI></Target>
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
    <Target><LocURI>Custom/URI</LocURI></Target>
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
			wantErr: "Custom configuration profiles can't include BitLocker settings.",
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
			wantErr: "Custom configuration profiles can't include BitLocker settings.",
		},
		{
			name: "XML with Multiple Replace Elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>Custom/URI1</LocURI></Target>
  </Item>
</Replace>
<Replace>
  <Item>
    <Target><LocURI>Custom/URI2</LocURI></Target>
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
    <Target><LocURI>Custom/URI</LocURI></Target>
  </Item>
</Replace>
<Replace>
  <Item>
    <Target><LocURI>./Device/Vendor/MSFT/BitLocker/Bar</LocURI></Target>
  </Item>
</Replace>
`),
			},
			wantErr: "Custom configuration profiles can't include BitLocker settings",
		},
		{
			name: "XML with Mixed Replace and Add",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>Custom/URI</LocURI></Target>
  </Item>
</Replace>
<Add>
  <Item>
    <Target><LocURI>Another/URI</LocURI></Target>
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
   <Target><LocURI>Replace/URI</LocURI></Target>
 </Item>
</Replace>
<Alert>
 <Item>
   <Target><LocURI>Alert/URI</LocURI></Target>
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
    <Target><LocURI>Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Atomic>
  <Item>
    <Target><LocURI>Atomic/URI</LocURI></Target>
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
    <Target><LocURI>Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Delete>
  <Item>
    <Target><LocURI>Delete/URI</LocURI></Target>
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
    <Target><LocURI>Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Exec>
  <Item>
    <Target><LocURI>Exec/URI</LocURI></Target>
  </Item>
</Exec>
`),
			},
			wantErr: "Windows configuration profiles can only have <Replace> or <Add> top level elements.",
		},
		{
			name: "XML with Replace and Get",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Get>
  <Item>
    <Target><LocURI>Get/URI</LocURI></Target>
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
    <Target><LocURI>Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Results>
  <Item>
    <Target><LocURI>Results/URI</LocURI></Target>
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
    <Target><LocURI>Replace/URI</LocURI></Target>
  </Item>
</Replace>
<Status>
  <Item>
    <Target><LocURI>Status/URI</LocURI></Target>
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
    <Target><LocURI>Custom/URI</LocURI></Target>
  </Item>
</Replace>
<Foo>
  <Item>
    <Target><LocURI>Another/URI</LocURI></Target>
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
    <Target><LocURI>Custom/URI</LocURI></Target>
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
    <Target><LocURI>Custom/URI</LocURI></Target>
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
    <Target><LocURI>Custom/URI</LocURI></Target>
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
    <Target><LocURI>Custom/URI</Target></LocURI>
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
    <Target><LocURI>Custom/URI</Target></LocURI>
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
    <Target><LocURI>Custom/URI</Target></LocURI>
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
				SyncML: []byte(`<Replace><Target><LocURI>Custom/URI</LocURI></Target></Replace>`),
			},
			wantErr: `Profile name "Windows OS Updates" is not allowed`,
		},
		{
			name: "XML with top level comment",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
				  <!-- this is a comment -->
				  <Replace>
				    <Target>
				      <LocURI>Custom/URI</LocURI>
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
				      <LocURI>Custom/URI</LocURI>
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
				      <LocURI>Custom/URI</LocURI>
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
				      <LocURI>Custom/URI</LocURI>
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
				      <LocURI>Custom/URI</LocURI>
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
				      <LocURI>Custom/URI</LocURI>
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
				      <LocURI>Custom/URI</LocURI>
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.ValidateUserProvided()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
