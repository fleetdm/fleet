package apple_mdm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDeviceInformationResponse_Basic(t *testing.T) {
	plistData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>some-uuid</string>
	<key>QueryResponses</key>
	<dict>
		<key>OSVersion</key>
		<string>17.4</string>
		<key>DeviceName</key>
		<string>Fleet-iPad</string>
		<key>DeviceCapacity</key>
		<real>64</real>
		<key>IsMDMLostModeEnabled</key>
		<false/>
	</dict>
	<key>Status</key>
	<string>Acknowledged</string>
	<key>UDID</key>
	<string>some-udid</string>
</dict>
</plist>`)

	result, err := ParseDeviceInformationResponse(plistData)
	require.NoError(t, err)

	assert.Equal(t, "17.4", result["DeviceInformation.OSVersion"])
	assert.Equal(t, "Fleet-iPad", result["DeviceInformation.DeviceName"])
	assert.Equal(t, "64", result["DeviceInformation.DeviceCapacity"])
	assert.Equal(t, "false", result["DeviceInformation.IsMDMLostModeEnabled"])
	assert.Len(t, result, 4)
}

func TestParseDeviceInformationResponse_MalformedPlist(t *testing.T) {
	_, err := ParseDeviceInformationResponse([]byte("not valid plist data at all"))
	require.Error(t, err)
}

func TestParseDeviceInformationResponse_EmptyQueryResponses(t *testing.T) {
	plistData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>some-uuid</string>
	<key>QueryResponses</key>
	<dict>
	</dict>
	<key>Status</key>
	<string>Acknowledged</string>
</dict>
</plist>`)

	result, err := ParseDeviceInformationResponse(plistData)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseSecurityInfoResponse_Basic(t *testing.T) {
	plistData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>SecurityInfo</key>
	<dict>
		<key>PasscodePresent</key>
		<true/>
		<key>HardwareEncryptionCaps</key>
		<integer>3</integer>
		<key>PasscodeCompliant</key>
		<true/>
	</dict>
	<key>Status</key>
	<string>Acknowledged</string>
</dict>
</plist>`)

	result, err := ParseSecurityInfoResponse(plistData)
	require.NoError(t, err)

	assert.Equal(t, "true", result["SecurityInfo.PasscodePresent"])
	assert.Equal(t, "3", result["SecurityInfo.HardwareEncryptionCaps"])
	assert.Equal(t, "true", result["SecurityInfo.PasscodeCompliant"])
	assert.Len(t, result, 3)
}

func TestParseSecurityInfoResponse_MalformedPlist(t *testing.T) {
	_, err := ParseSecurityInfoResponse([]byte("garbage data here"))
	require.Error(t, err)
}

func TestParseInstalledApplicationListResponse_Basic(t *testing.T) {
	plistData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>InstalledApplicationList</key>
	<array>
		<dict>
			<key>Identifier</key>
			<string>com.apple.mobilesafari</string>
			<key>Name</key>
			<string>Safari</string>
			<key>ShortVersion</key>
			<string>17.4</string>
			<key>Version</key>
			<string>17.4</string>
		</dict>
		<dict>
			<key>Identifier</key>
			<string>com.slack.Slack</string>
			<key>Name</key>
			<string>Slack</string>
			<key>ShortVersion</key>
			<string>4.0</string>
		</dict>
	</array>
	<key>Status</key>
	<string>Acknowledged</string>
</dict>
</plist>`)

	result, err := ParseInstalledApplicationListResponse(plistData)
	require.NoError(t, err)

	assert.Equal(t, "com.apple.mobilesafari", result["InstalledApplicationList.com.apple.mobilesafari.Identifier"])
	assert.Equal(t, "Safari", result["InstalledApplicationList.com.apple.mobilesafari.Name"])
	assert.Equal(t, "17.4", result["InstalledApplicationList.com.apple.mobilesafari.ShortVersion"])
	assert.Equal(t, "17.4", result["InstalledApplicationList.com.apple.mobilesafari.Version"])
	assert.Equal(t, "com.slack.Slack", result["InstalledApplicationList.com.slack.Slack.Identifier"])
	assert.Equal(t, "Slack", result["InstalledApplicationList.com.slack.Slack.Name"])
	assert.Equal(t, "4.0", result["InstalledApplicationList.com.slack.Slack.ShortVersion"])
	assert.Len(t, result, 7) // 4 Safari fields + 3 Slack fields (no Version for Slack)
}

func TestParseInstalledApplicationListResponse_Empty(t *testing.T) {
	plistData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>InstalledApplicationList</key>
	<array>
	</array>
	<key>Status</key>
	<string>Acknowledged</string>
</dict>
</plist>`)

	result, err := ParseInstalledApplicationListResponse(plistData)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseInstalledApplicationListResponse_MalformedPlist(t *testing.T) {
	_, err := ParseInstalledApplicationListResponse([]byte("garbage"))
	require.Error(t, err)
}

func TestParseInstalledApplicationListResponse_MissingIdentifier(t *testing.T) {
	plistData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>InstalledApplicationList</key>
	<array>
		<dict>
			<key>Identifier</key>
			<string>com.apple.mobilesafari</string>
			<key>Name</key>
			<string>Safari</string>
		</dict>
		<dict>
			<key>Name</key>
			<string>SomeApp</string>
			<key>ShortVersion</key>
			<string>1.0</string>
		</dict>
	</array>
	<key>Status</key>
	<string>Acknowledged</string>
</dict>
</plist>`)

	result, err := ParseInstalledApplicationListResponse(plistData)
	require.NoError(t, err)

	// Safari should be present
	assert.Equal(t, "Safari", result["InstalledApplicationList.com.apple.mobilesafari.Name"])
	// The app without Identifier should be skipped
	_, hasNoIdApp := result["InstalledApplicationList..Name"]
	assert.False(t, hasNoIdApp, "app without Identifier should be skipped")
	assert.Len(t, result, 2) // Only Safari's 2 fields
}
