package main

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mobileconfigPayload(identifier, displayName string) []byte {
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
</dict>
</plist>`, identifier, displayName)
}

func installProfileCommandWithPayload(payload []byte) *mdm.Command {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>cmd-uuid</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>InstallProfile</string>
		<key>Payload</key>
		<data>%s</data>
	</dict>
</dict>
</plist>`, base64.StdEncoding.EncodeToString(payload))
	return &mdm.Command{Raw: []byte(raw)}
}

func installProfileCommand(t *testing.T, identifier, displayName string) *mdm.Command {
	t.Helper()
	return installProfileCommandWithPayload(mobileconfigPayload(identifier, displayName))
}

func removeProfileCommand(identifier string) *mdm.Command {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>cmd-uuid</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>RemoveProfile</string>
		<key>Identifier</key>
		<string>%s</string>
	</dict>
</dict>
</plist>`, identifier)
	return &mdm.Command{Raw: []byte(raw)}
}

func TestParseInstallProfileCommand(t *testing.T) {
	identifier, displayName, ok := parseInstallProfileCommand(installProfileCommand(t, "com.example.test", "Test Profile"))
	require.True(t, ok)
	assert.Equal(t, "com.example.test", identifier)
	assert.Equal(t, "Test Profile", displayName)

	// display name falls back to the identifier when absent
	identifier, displayName, ok = parseInstallProfileCommand(installProfileCommand(t, "com.example.noname", ""))
	require.True(t, ok)
	assert.Equal(t, "com.example.noname", identifier)
	assert.Equal(t, "com.example.noname", displayName)

	// not an InstallProfile command
	_, _, ok = parseInstallProfileCommand(removeProfileCommand("com.example.test"))
	assert.False(t, ok)

	// garbage payload
	_, _, ok = parseInstallProfileCommand(&mdm.Command{Raw: []byte("not a plist")})
	assert.False(t, ok)
}

// TestParseInstallProfileCommandSigned covers the PKCS7 branch of
// installProfilePayload. Fleet always signs profiles before sending them (see
// MDMAppleCommander.SignAndEncodeInstallProfile), so this is the path every
// real InstallProfile command takes.
func TestParseInstallProfileCommandSigned(t *testing.T) {
	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "osquery-perf test signer"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	signedData, err := pkcs7.NewSignedData(mobileconfigPayload("com.example.signed", "Signed Profile"))
	require.NoError(t, err)
	require.NoError(t, signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}))
	signed, err := signedData.Finish()
	require.NoError(t, err)

	identifier, displayName, ok := parseInstallProfileCommand(installProfileCommandWithPayload(signed))
	require.True(t, ok)
	assert.Equal(t, "com.example.signed", identifier)
	assert.Equal(t, "Signed Profile", displayName)
}

func TestRemoveProfileIdentifier(t *testing.T) {
	assert.Equal(t, "com.example.test", removeProfileIdentifier(removeProfileCommand("com.example.test")))
	assert.Empty(t, removeProfileIdentifier(installProfileCommand(t, "com.example.test", "Test Profile")))
	assert.Empty(t, removeProfileIdentifier(&mdm.Command{Raw: []byte("not a plist")}))
}

func TestInstalledProfileTracking(t *testing.T) {
	a := &agent{}

	// nothing installed yet
	assert.Empty(t, a.mdmConfigProfilesMac())
	assert.False(t, a.removeInstalledProfile("device", "com.example.test"))

	a.seedEnrollmentProfile()
	a.recordInstalledProfile("device", installProfileCommand(t, "com.example.device", "Device Profile"))
	a.recordInstalledProfile("user", installProfileCommand(t, "com.example.user", "User Profile"))

	results := a.mdmConfigProfilesMac()
	require.Len(t, results, 3)
	byIdentifier := make(map[string]map[string]string, len(results))
	for _, row := range results {
		byIdentifier[row["identifier"]] = row
	}
	require.Contains(t, byIdentifier, apple_mdm.FleetPayloadIdentifier)
	require.Contains(t, byIdentifier, "com.example.device")
	require.Contains(t, byIdentifier, "com.example.user")
	assert.Equal(t, "Device Profile", byIdentifier["com.example.device"]["display_name"])

	// install_date must be in the format the server's
	// parseMacOSProfileInstallDate expects, and recent
	installDate, err := time.Parse("2006-01-02 15:04:05 -0700", byIdentifier["com.example.device"]["install_date"])
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), installDate, time.Minute)

	// removal is channel-scoped
	assert.False(t, a.removeInstalledProfile("user", "com.example.device"))
	assert.True(t, a.removeInstalledProfile("device", "com.example.device"))
	assert.False(t, a.removeInstalledProfile("device", "com.example.device"))
	assert.True(t, a.removeInstalledProfile("user", "com.example.user"))
	assert.True(t, a.removeInstalledProfile("device", apple_mdm.FleetPayloadIdentifier))
	assert.Empty(t, a.mdmConfigProfilesMac())
}

func TestProcessQueryMDMConfigProfiles(t *testing.T) {
	a := &agent{}
	a.seedEnrollmentProfile()
	var cached cachedResults

	// The legacy device-only query simulates a failed osquery discovery:
	// handled, but nothing submitted for it.
	handled, results, status, message, stats := a.processQuery("fleet_detail_query_mdm_config_profiles_darwin", "SELECT 1;", &cached)
	assert.True(t, handled)
	assert.Nil(t, results)
	assert.Nil(t, status)
	assert.Nil(t, message)
	assert.Nil(t, stats)

	// The with-user query reports the tracked profiles. It simulates a failed
	// query ~10% of the time, so retry until a success is observed.
	sawSuccess := false
	for i := 0; i < 100 && !sawSuccess; i++ {
		handled, results, status, _, _ := a.processQuery("fleet_detail_query_mdm_config_profiles_darwin_with_user", "SELECT 1;", &cached)
		require.True(t, handled)
		require.NotNil(t, status)
		if *status == fleet.StatusOK {
			require.Len(t, results, 1)
			assert.Equal(t, apple_mdm.FleetPayloadIdentifier, results[0]["identifier"])
			sawSuccess = true
		} else {
			assert.Empty(t, results)
		}
	}
	assert.True(t, sawSuccess)
}

func TestDelayCancelableMDMAck(t *testing.T) {
	mkCmd := func(requestType string) *mdm.Command {
		cmd := &mdm.Command{CommandUUID: "test-uuid"}
		cmd.Command.RequestType = requestType
		return cmd
	}
	elapsed := func(delay time.Duration, cmd *mdm.Command) time.Duration {
		start := time.Now()
		delayCancelableMDMAck(delay, "test device", cmd)
		return time.Since(start)
	}

	// non-cancelable commands never wait, whatever the delay
	assert.Less(t, elapsed(time.Hour, mkCmd("InstallProfile")), time.Second)
	assert.Less(t, elapsed(time.Hour, nil), time.Second)

	// a zero delay never waits, even for cancelable commands
	for requestType := range fleet.CancelableAppleMDMRequestTypes {
		assert.Less(t, elapsed(0, mkCmd(requestType)), time.Second)
	}

	// cancelable commands wait out the delay
	assert.GreaterOrEqual(t, elapsed(20*time.Millisecond, mkCmd("DeviceLock")), 20*time.Millisecond)
}
