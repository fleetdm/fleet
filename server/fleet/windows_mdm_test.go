package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/stretchr/testify/require"
)

func TestValidateUserProvided(t *testing.T) {
	tests := []struct {
		name    string
		profile MDMWindowsConfigProfile
		wantErr bool
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
			wantErr: false,
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
			wantErr: true,
		},
		{
			name: "Invalid XML Structure",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Add>
  <Item>
    <Target><LocURI>Custom/URI</LocURI></Target>
  </Item>
</Add>
`),
			},
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: false,
		},
		{
			name: "Empty XML",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(``),
			},
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
		},
		{
			name: "invalid XML",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
  <Item>
    <Target><LocURI>Custom/URI</LocURI></Target>
  </Item>
</Add>
`),
			},
			wantErr: true,
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
			wantErr: false,
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
			wantErr: false,
		},
		{
			name: "no items in Replace",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`
<Replace>
</Replace>
`),
			},
			wantErr: false,
		},
		{
			name: "Valid XML with reserved name",
			profile: MDMWindowsConfigProfile{
				Name:   syncml.FleetWindowsOSUpdatesProfileName,
				SyncML: []byte(`<Replace><Target><LocURI>Custom/URI</LocURI></Target></Replace>`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.ValidateUserProvided()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
