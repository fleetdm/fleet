//go:build darwin
// +build darwin

package authdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAuthDBReadOutput(t *testing.T) {
	const systemLoginScreensaver = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>class</key>
        <string>rule</string>
        <key>created</key>
        <real>656503622.12447298</real>
        <key>modified</key>
        <real>697495406.285501</real>
        <key>rule</key>
        <array>
                <string>authenticate-session-owner-or-admin</string>
        </array>
        <key>version</key>
        <integer>0</integer>
</dict>
</plist>`
	m, err := parseAuthDBReadOutput([]byte(systemLoginScreensaver))
	require.NoError(t, err)
	require.NotNil(t, m["rule"])
	rule, ok := m["rule"].([]interface{})
	require.True(t, ok)
	require.Len(t, rule, 1)
	require.Equal(t, "authenticate-session-owner-or-admin", rule[0])
	require.Equal(t, "rule", m["class"])
}
