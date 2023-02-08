//go:build darwin
// +build darwin

package dscl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDSCLOutput(t *testing.T) {
	// NOTE(lucas): I've seen the following behavior when running the command as non-root.
	const noKeySample = `No such key: Foobar`
	value, err := parseDSCLReadOutput([]byte(noKeySample))
	require.NoError(t, err)
	require.Nil(t, value)

	// NOTE(lucas): I've seen the following behavior when running the command as root.
	const noKeySample2 = ``
	value, err = parseDSCLReadOutput([]byte(noKeySample2))
	require.NoError(t, err)
	require.Nil(t, value)

	const keySample0 = `PrimaryGroupID: 20`
	value, err = parseDSCLReadOutput([]byte(keySample0))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, "20", *value)

	const keySample1 = `Picture:
 /Library/User Pictures/Animals/Penguin.tif`
	value, err = parseDSCLReadOutput([]byte(keySample1))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, "/Library/User Pictures/Animals/Penguin.tif", *value)

	const keySample2 = `RecordName: foo com.apple.idms.appleid.prd.0A771AC1-B614-4A18-9FA5-0ADFA8EED4BC`
	value, err = parseDSCLReadOutput([]byte(keySample2))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, "foo com.apple.idms.appleid.prd.0A771AC1-B614-4A18-9FA5-0ADFA8EED4BC", *value)

	const keySample3 = `dsAttrTypeNative:accountPolicyData:
 <?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>creationTime</key>
        <real>1634811726.5620289</real>
        <key>failedLoginCount</key>
        <integer>0</integer>
        <key>failedLoginTimestamp</key>
        <integer>0</integer>
        <key>passwordLastSetTime</key>
        <real>1636975330.6275649</real>
</dict>
</plist>`
	value, err = parseDSCLReadOutput([]byte(keySample3))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>creationTime</key>
        <real>1634811726.5620289</real>
        <key>failedLoginCount</key>
        <integer>0</integer>
        <key>failedLoginTimestamp</key>
        <integer>0</integer>
        <key>passwordLastSetTime</key>
        <real>1636975330.6275649</real>
</dict>
</plist>`, *value)

	const keySample4 = `RecordType: dsRecTypeStandard:Users`
	value, err = parseDSCLReadOutput([]byte(keySample4))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, "dsRecTypeStandard:Users", *value)

	const keySample5 = `RecordName:
 root
 BUILTIN\Local System`
	value, err = parseDSCLReadOutput([]byte(keySample5))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, `root
 BUILTIN\Local System`, *value)

	const keySample6 = `NFSHomeDirectory: /var/root /private/var/root`
	value, err = parseDSCLReadOutput([]byte(keySample6))
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, `/var/root /private/var/root`, *value)
}
