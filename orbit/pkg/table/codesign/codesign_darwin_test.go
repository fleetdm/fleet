package codesign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCodesignOutput(t *testing.T) {
	output := []byte(`
Executable=/Applications/Xcode.app/Contents/MacOS/Xcode
Identifier=com.apple.dt.Xcode
Format=app bundle with Mach-O universal (x86_64 arm64)
CodeDirectory v=20400 size=790 flags=0x2000(library-validation) hashes=14+7 location=embedded
Hash type=sha256 size=32
CandidateCDHash sha1=21bbfcedb1ba1ed7078187432cf79234d65e290b
CandidateCDHashFull sha1=21bbfcedb1ba1ed7078187432cf79234d65e290b
CandidateCDHash sha256=cd1f004f0b0cd90c27d72375c7b9546b4c6df361
CandidateCDHashFull sha256=cd1f004f0b0cd90c27d72375c7b9546b4c6df3610868f18ae49ca50c8dfce2d9
Hash choices=sha1,sha256
CMSDigest=e4d43bc2286f60ee818e829f2f72b909c86b2235ec91a44290ec51fdc2f11897
CMSDigestType=2
CDHash=cd1f004f0b0cd90c27d72375c7b9546b4c6df361
Signature size=4797
Authority=Apple Mac OS Application Signing
Authority=Apple Worldwide Developer Relations Certification Authority
Authority=Apple Root CA
Info.plist entries=43
TeamIdentifier=59GAB85EFG
Sealed Resources version=2 rules=13 files=108583
Internal requirements count=1 size=220
`)

	info := parseCodesignOutput(output)

	require.Equal(t, "59GAB85EFG", info.teamIdentifier)
	require.Equal(t, "cd1f004f0b0cd90c27d72375c7b9546b4c6df361", info.cdHash)
}
