package fleet

import (
	"testing"

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
				SyncML: []byte(`<Replace><Target><LocURI>Custom/URI</LocURI></Target></Replace>`),
			},
			wantErr: false,
		},
		{
			name: "Invalid Platform",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<SyncML xmlns="SYNCML:SYNCML1.2"><Replace><Target><LocURI>Custom/URI</LocURI></Target></Replace></SyncML>`),
			},
			wantErr: true,
		},
		{
			name: "Invalid XML Structure",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Add><Target><LocURI>Custom/URI</LocURI></Target></Add>`),
			},
			wantErr: true,
		},
		{
			name: "Reserved LocURI",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>./Device/Vendor/MSFT/BitLocker/Foo</LocURI></Target></Replace>`),
			},
			wantErr: true,
		},
		{
			name: "XML with Multiple Replace Elements",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>Custom/URI1</LocURI></Target></Replace><Replace><Target><LocURI>Custom/URI2</LocURI></Target></Replace>`),
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
				SyncML: []byte(`<Replace><Target><LocURI>Custom/URI</LocURI></Target></Replace><Replace><Target><LocURI>./Device/Vendor/MSFT/BitLocker/Bar</LocURI></Target></Replace>`),
			},
			wantErr: true,
		},
		{
			name: "XML with Mixed Replace and Add",
			profile: MDMWindowsConfigProfile{
				SyncML: []byte(`<Replace><Target><LocURI>Custom/URI</LocURI></Target></Replace><Add><Target><LocURI>Another/URI</LocURI></Target></Add>`),
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
