package sigverify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const pkgutilSignedOutput = `Package "BoxDrive-2.52.312.pkg":
   Status: signed by a developer certificate issued by Apple for distribution
   Notarization: trusted by the Apple notary service
   Signed with a trusted timestamp on: 2026-06-25 20:36:22 +0000
   Certificate Chain:
    1. Developer ID Installer: Box, Inc. (M683GB7CPW)
       Expires: 2027-02-01 22:12:15 +0000
       SHA256 Fingerprint:
           AA BB CC DD
       ------------------------------------------------------------------------
    2. Developer ID Certification Authority
       Expires: 2031-09-16 00:00:00 +0000
       SHA256 Fingerprint:
           EE FF 00 11
       ------------------------------------------------------------------------
    3. Apple Root CA
       Expires: 2035-02-09 21:40:36 +0000
`

const pkgutilUnsignedOutput = `Package "7z2602-x64.msi":
   Status: no signature
`

const pkgutilUntrustedOutput = `Package "shady.pkg":
   Status: signed by untrusted certificate
   Certificate Chain:
    1. Developer ID Installer: Shady Corp (ABCDE12345)
`

func TestParsePkgutilOutput(t *testing.T) {
	t.Run("signed and trusted", func(t *testing.T) {
		res := ParsePkgutilOutput(pkgutilSignedOutput)
		require.True(t, res.Verified)
		require.False(t, res.NoSignature)
		require.Equal(t, "M683GB7CPW", res.TeamID)
		require.Equal(t, "Developer ID Installer: Box, Inc. (M683GB7CPW)", res.Identity)
	})

	t.Run("unsigned", func(t *testing.T) {
		res := ParsePkgutilOutput(pkgutilUnsignedOutput)
		require.False(t, res.Verified)
		require.True(t, res.NoSignature)
	})

	t.Run("untrusted", func(t *testing.T) {
		res := ParsePkgutilOutput(pkgutilUntrustedOutput)
		require.False(t, res.Verified)
		require.False(t, res.NoSignature)
		require.Contains(t, res.Detail, "untrusted")
		require.Equal(t, "ABCDE12345", res.TeamID)
	})
}

const codesignDvvOutput = `Executable=/Volumes/Rectangle0.98/Rectangle.app/Contents/MacOS/Rectangle
Identifier=com.knollsoft.Rectangle
Format=app bundle with Mach-O universal (x86_64 arm64)
CodeDirectory v=20500 size=25698 flags=0x10000(runtime) hashes=792+7 location=embedded
Signature size=9000
Authority=Developer ID Application: Ryan Hanson (XSYZ3E4B7D)
Authority=Developer ID Certification Authority
Authority=Apple Root CA
Timestamp=Jun 25, 2026 at 1:23:45 PM
Info.plist entries=32
TeamIdentifier=XSYZ3E4B7D
Runtime Version=15.0.0
Sealed Resources version=2 rules=13 files=145
Internal requirements count=1 size=216
`

func TestParseCodesignInfo(t *testing.T) {
	identity, teamID := ParseCodesignInfo(codesignDvvOutput)
	require.Equal(t, "XSYZ3E4B7D", teamID)
	require.Equal(t, "Developer ID Application: Ryan Hanson (XSYZ3E4B7D)", identity)

	identity, teamID = ParseCodesignInfo("some unrelated output")
	require.Empty(t, teamID)
	require.Empty(t, identity)
}

const spctlAcceptedOutput = `/tmp/BoxDrive-2.52.312.pkg: accepted
source=Notarized Developer ID
origin=Developer ID Installer: Box, Inc. (M683GB7CPW)
`

const spctlRejectedOutput = `/tmp/shady.pkg: rejected
source=no usable signature
`

func TestParseSpctlOutput(t *testing.T) {
	t.Run("accepted and notarized", func(t *testing.T) {
		assess := ParseSpctlOutput(spctlAcceptedOutput)
		require.True(t, assess.Accepted)
		require.Equal(t, "Notarized Developer ID", assess.Source)
		require.Equal(t, "Developer ID Installer: Box, Inc. (M683GB7CPW)", assess.Origin)
		require.Equal(t, "accepted; source=Notarized Developer ID", assess.Summary())
	})

	t.Run("rejected", func(t *testing.T) {
		assess := ParseSpctlOutput(spctlRejectedOutput)
		require.False(t, assess.Accepted)
		require.Equal(t, "no usable signature", assess.Source)
		require.Equal(t, "rejected; source=no usable signature", assess.Summary())
	})
}

const hdiutilPlistOutput = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>system-entities</key>
	<array>
		<dict>
			<key>content-hint</key>
			<string>GUID_partition_scheme</string>
			<key>dev-entry</key>
			<string>/dev/disk4</string>
		</dict>
		<dict>
			<key>content-hint</key>
			<string>Apple_HFS</string>
			<key>dev-entry</key>
			<string>/dev/disk4s1</string>
			<key>mount-point</key>
			<string>/Volumes/Rectangle0.98</string>
		</dict>
	</array>
</dict>
</plist>
`

func TestParseHdiutilMountPoint(t *testing.T) {
	require.Equal(t, "/Volumes/Rectangle0.98", parseHdiutilMountPoint(hdiutilPlistOutput))
	require.Empty(t, parseHdiutilMountPoint("<plist></plist>"))
}
