package fleet

import (
	"testing"

	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
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
			name: "Valid XML with reserved name",
			profile: MDMWindowsConfigProfile{
				Name:   microsoft_mdm.FleetWindowsOSUpdatesProfileName,
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
