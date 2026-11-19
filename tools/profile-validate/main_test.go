package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func profile(payload string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>PayloadType</key><string>Configuration</string>
	<key>PayloadVersion</key><integer>1</integer>
	<key>PayloadIdentifier</key><string>com.example.profile</string>
	<key>PayloadUUID</key><string>0F1E2D3C-4B5A-6978-8796-A5B4C3D2E1F0</string>
	<key>PayloadDisplayName</key><string>Example</string>
	<key>PayloadContent</key><array><dict>
		<key>PayloadType</key><string>com.apple.applicationaccess</string>
		<key>PayloadIdentifier</key><string>com.example.restrictions</string>
		<key>PayloadUUID</key><string>a1b2c3d4-e5f6-a7b8-c9d0-e1f2a3b4c5d6</string>` + payload + `
	</dict></array>
</dict></plist>`
}

func TestRun(t *testing.T) {
	// Parsing twice errors; the schema is the same either way.
	_ = profilevalidation.ParseProfileValidationSchema()
	s := profilevalidation.Current()
	require.NotNil(t, s)

	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		return path
	}
	write("good.mobileconfig", profile(`<key>allowAirDrop</key><false/>`))
	bad := write("nested/bad.mobileconfig", profile(`<key>allowAirDrop</key><string>no</string>`))
	write("unreadable.mobileconfig", `<?xml version="1.0"?><plist><dict><key>k</key><data>$NOT_EXPANDED</data></dict></plist>`)
	write("declaration.json", `{"Type": "com.apple.configuration.passcode.settings", "Identifier": "x", "Payload": {"MinimumLength": "8"}}`)
	write("android.json", `{"passwordPolicies": []}`)
	write("broken.json", `{"Type": "com.apple.configuration.passcode.settings",}`)
	write("notes.txt", "not a profile")
	const minLength = "./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength"
	syncML := func(data string) string {
		return `<Replace><Item><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Target><LocURI>` + minLength + `</LocURI></Target><Data>` + data + `</Data></Item></Replace>`
	}
	write("windows/good.xml", syncML("8"))
	badWindows := write("windows/bad.xml", syncML("eight"))
	write("windows/other.xml", `<?xml version="1.0"?><config><setting>not SyncML</setting></config>`)

	var out, logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	findings, err := run(t.Context(), logger, &out, s, profilevalidation.ModeLenient, []string{dir})
	require.NoError(t, err)
	assert.Equal(t, 3, findings)
	// Unparseable files are reported as skipped, not silently ignored as non-Apple JSON.
	assert.Contains(t, logs.String(), "files=7 findings=3 skipped=2")
	assert.Contains(t, logs.String(), "broken.json")
	assert.Equal(t,
		filepath.Join(dir, "declaration.json")+": TYPE_MISMATCH com.apple.configuration.passcode.settings MinimumLength expected=integer actual=string\n"+
			bad+": TYPE_MISMATCH com.apple.applicationaccess allowAirDrop expected=boolean actual=string\n"+
			badWindows+": TYPE_MISMATCH DeviceLock "+minLength+" expected=int actual=\n",
		out.String())

	// Strict adds the restrictions payload's missing PayloadVersion in both profiles.
	out.Reset()
	findings, err = run(t.Context(), logger, &out, s, profilevalidation.ModeStrict, []string{dir})
	require.NoError(t, err)
	assert.Equal(t, 5, findings)

	// A single file works as a path.
	out.Reset()
	findings, err = run(t.Context(), logger, &out, s, profilevalidation.ModeLenient, []string{bad})
	require.NoError(t, err)
	assert.Equal(t, 1, findings)

	_, err = run(t.Context(), logger, &out, s, profilevalidation.ModeLenient, []string{filepath.Join(dir, "missing")})
	require.Error(t, err)
}
