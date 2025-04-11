package mobileconfig

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsFDEVileVaultOptionsPayload(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		contains bool
	}{
		{
			name:     "no-op",
			in:       "",
			contains: false,
		},
		{
			name: "not com.apple.MCX payload",
			in: getFileVaultOptionsPayload(FDEFileVaultOptionsPayload{
				PayloadType:         "com.apple.security.scep",
				DontAllowFDEDisable: ptr.Bool(true),
			}),
			contains: false,
		},
		{
			name: "com.apple.MCX payload, no FDE options",
			in: getFileVaultOptionsPayload(FDEFileVaultOptionsPayload{
				PayloadType: FleetCustomSettingsPayloadType,
			}),
			contains: false,
		},
		{
			// Add all the FileVault options to the custom settings payload
			name: "com.apple.MCX payload with all FDE options",
			in: getFileVaultOptionsPayload(FDEFileVaultOptionsPayload{
				PayloadType:           FleetCustomSettingsPayloadType,
				DestroyFVKeyOnStandby: ptr.Bool(true),
				DontAllowFDEDisable:   ptr.Bool(true),
				DontAllowFDEEnable:    ptr.Bool(true),
			}),
			contains: true,
		},
		{
			// Only add the dontAllowFDEDisable property to the custom settings payload
			name: "contains dontAllowFDEDisable",
			in: getFileVaultOptionsPayload(FDEFileVaultOptionsPayload{
				PayloadType:         FleetCustomSettingsPayloadType,
				DontAllowFDEDisable: ptr.Bool(false),
			}),
			contains: true,
		},
		{
			// Only add the dontAllowFDEEnable property to the custom settings payload
			name: "contains dontAllowFDEEnable",
			in: getFileVaultOptionsPayload(FDEFileVaultOptionsPayload{
				PayloadType:        FleetCustomSettingsPayloadType,
				DontAllowFDEEnable: ptr.Bool(false),
			}),
			contains: true,
		},
		{
			// Only add the DestroyFVKeyOnStandby property to the custom settings payload
			name: "contains DestroyFVKeyOnStandby",
			in: getFileVaultOptionsPayload(FDEFileVaultOptionsPayload{
				PayloadType:           FleetCustomSettingsPayloadType,
				DestroyFVKeyOnStandby: ptr.Bool(false),
			}),
			contains: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ContainsFDEFileVaultOptionsPayload([]byte(tc.in))
			require.NoError(t, err)
			assert.Equal(t, tc.contains, result)
		})
	}
}

func getFileVaultOptionsPayload(payload FDEFileVaultOptionsPayload) string {
	var (
		DestroyFVKeyOnStandby string
		dontAllowFDEDisable   string
		dontAllowFDEEnable    string
	)
	if payload.DestroyFVKeyOnStandby != nil {
		DestroyFVKeyOnStandby = fmt.Sprintf("<key>DestroyFVKeyOnStandby</key><%t/>", *payload.DestroyFVKeyOnStandby)
	}
	if payload.DontAllowFDEDisable != nil {
		dontAllowFDEDisable = fmt.Sprintf("<key>dontAllowFDEDisable</key><%t/>", *payload.DontAllowFDEDisable)
	}
	if payload.DontAllowFDEEnable != nil {
		dontAllowFDEEnable = fmt.Sprintf("<key>dontAllowFDEEnable</key><%t/>", *payload.DontAllowFDEEnable)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            %s%s%s
            <key>PayloadIdentifier</key>
            <string>com.example.fdefvoptionspayload</string>
            <key>PayloadType</key>
            <string>%s</string>
            <key>PayloadUUID</key>
            <string>0a8f4102-0fbf-4d8c-b1e1-3d916f89d927</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
        <dict>
            <key>dontAllowFDEDisable</key>
            <true/>
            <key>PayloadIdentifier</key>
            <string>com.example.pkcs12</string>
            <key>PayloadType</key>
            <string>com.apple.security.pkcs12</string>
            <key>PayloadContent</key>
            <data>bozo</data>
            <key>PayloadUUID</key>
            <string>0a8f4102-0fbf-4d8c-b1e1-3d916f89d927</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>FileVault 2 Options</string>
    <key>PayloadIdentifier</key>
    <string>com.example.myprofile</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>92821df0-7c04-4366-b805-eb51ed87541b</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`, DestroyFVKeyOnStandby, dontAllowFDEDisable, dontAllowFDEEnable, payload.PayloadType)
}
