package android

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCertAliasesFromONC(t *testing.T) {
	tests := []struct {
		name     string
		oncJSON  string
		expected []string
		wantErr  bool
	}{
		{
			name: "WiFi EAP with KeyPairAlias",
			oncJSON: `{
				"NetworkConfigurations": [{
					"WiFi": {
						"EAP": {
							"ClientCertType": "KeyPairAlias",
							"ClientCertKeyPairAlias": "wifi-cert"
						}
					}
				}]
			}`,
			expected: []string{"wifi-cert"},
		},
		{
			name: "VPN with KeyPairAlias",
			oncJSON: `{
				"NetworkConfigurations": [{
					"VPN": {
						"ClientCertType": "KeyPairAlias",
						"ClientCertKeyPairAlias": "vpn-cert"
					}
				}]
			}`,
			expected: []string{"vpn-cert"},
		},
		{
			name: "WiFi without EAP (WPA-PSK)",
			oncJSON: `{
				"NetworkConfigurations": [{
					"WiFi": {
						"SSID": "GuestNet",
						"Security": "WPA-PSK",
						"Passphrase": "guest123"
					}
				}]
			}`,
			expected: nil,
		},
		{
			name: "alias present but ClientCertType is Ref (ignored per ONC spec)",
			oncJSON: `{
				"NetworkConfigurations": [{
					"WiFi": {
						"EAP": {
							"ClientCertType": "Ref",
							"ClientCertKeyPairAlias": "should-be-ignored"
						}
					}
				}]
			}`,
			expected: nil,
		},
		{
			name: "alias present but ClientCertType is missing (ignored per ONC spec)",
			oncJSON: `{
				"NetworkConfigurations": [{
					"WiFi": {
						"EAP": {
							"ClientCertKeyPairAlias": "should-be-ignored"
						}
					}
				}]
			}`,
			expected: nil,
		},
		{
			name: "mix of KeyPairAlias and other ClientCertTypes",
			oncJSON: `{
				"NetworkConfigurations": [
					{
						"WiFi": {
							"EAP": {
								"ClientCertType": "KeyPairAlias",
								"ClientCertKeyPairAlias": "real-cert"
							}
						}
					},
					{
						"VPN": {
							"ClientCertType": "Ref",
							"ClientCertKeyPairAlias": "ignored-cert"
						}
					}
				]
			}`,
			expected: []string{"real-cert"},
		},
		{
			name:    "invalid JSON",
			oncJSON: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases, err := ExtractCertAliasesFromONC(json.RawMessage(tt.oncJSON))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, aliases)
		})
	}
}

func TestExtractCertAliasesFromProfileJSON(t *testing.T) {
	tests := []struct {
		name        string
		profileJSON string
		expected    []string
		wantErr     bool
	}{
		{
			name: "profile with ONC containing cert ref",
			profileJSON: `{
				"openNetworkConfiguration": {
					"NetworkConfigurations": [{
						"WiFi": {
							"EAP": {
								"ClientCertType": "KeyPairAlias",
								"ClientCertKeyPairAlias": "my-cert"
							}
						}
					}]
				}
			}`,
			expected: []string{"my-cert"},
		},
		{
			name:        "profile without ONC field",
			profileJSON: `{"cameraDisabled": true, "maximumTimeToLock": 300}`,
			expected:    nil,
		},
		{
			name:        "invalid JSON",
			profileJSON: `not json`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases, err := ExtractCertAliasesFromProfileJSON(json.RawMessage(tt.profileJSON))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, aliases)
		})
	}
}
