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
			name: "WiFi EAP with ClientCertKeyPairAlias",
			oncJSON: `{
				"NetworkConfigurations": [{
					"WiFi": {
						"EAP": {
							"ClientCertKeyPairAlias": "wifi-cert"
						}
					}
				}]
			}`,
			expected: []string{"wifi-cert"},
		},
		{
			name: "VPN with ClientCertKeyPairAlias",
			oncJSON: `{
				"NetworkConfigurations": [{
					"VPN": {
						"ClientCertKeyPairAlias": "vpn-cert"
					}
				}]
			}`,
			expected: []string{"vpn-cert"},
		},
		{
			name: "Ethernet EAP with ClientCertKeyPairAlias",
			oncJSON: `{
				"NetworkConfigurations": [{
					"Ethernet": {
						"EAP": {
							"ClientCertKeyPairAlias": "ethernet-cert"
						}
					}
				}]
			}`,
			expected: []string{"ethernet-cert"},
		},
		{
			name: "multiple network configs with multiple aliases",
			oncJSON: `{
				"NetworkConfigurations": [
					{
						"WiFi": {
							"EAP": {
								"ClientCertKeyPairAlias": "wifi-cert"
							}
						}
					},
					{
						"VPN": {
							"ClientCertKeyPairAlias": "vpn-cert"
						}
					},
					{
						"Ethernet": {
							"EAP": {
								"ClientCertKeyPairAlias": "eth-cert"
							}
						}
					}
				]
			}`,
			expected: []string{"wifi-cert", "vpn-cert", "eth-cert"},
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
			name:     "empty NetworkConfigurations",
			oncJSON:  `{"NetworkConfigurations": []}`,
			expected: nil,
		},
		{
			name: "WiFi EAP without ClientCertKeyPairAlias",
			oncJSON: `{
				"NetworkConfigurations": [{
					"WiFi": {
						"EAP": {
							"Outer": "PEAP"
						}
					}
				}]
			}`,
			expected: nil,
		},
		{
			name:    "invalid JSON",
			oncJSON: `{invalid}`,
			wantErr: true,
		},
		{
			name:     "empty object",
			oncJSON:  `{}`,
			expected: nil,
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
			name: "profile with ONC but no cert refs",
			profileJSON: `{
				"openNetworkConfiguration": {
					"NetworkConfigurations": [{
						"WiFi": {
							"SSID": "Open",
							"Security": "None"
						}
					}]
				}
			}`,
			expected: nil,
		},
		{
			name:        "invalid JSON",
			profileJSON: `not json`,
			wantErr:     true,
		},
		{
			name: "profile with ONC and other fields",
			profileJSON: `{
				"cameraDisabled": true,
				"openNetworkConfiguration": {
					"NetworkConfigurations": [{
						"WiFi": {
							"EAP": {
								"ClientCertKeyPairAlias": "corp-wifi-cert"
							}
						}
					}]
				},
				"maximumTimeToLock": 600
			}`,
			expected: []string{"corp-wifi-cert"},
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
