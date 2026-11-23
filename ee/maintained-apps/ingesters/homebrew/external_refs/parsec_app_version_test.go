package externalrefs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsecPackageAppVersion(t *testing.T) {
	t.Run("real-shaped PackageInfo file", func(t *testing.T) {
		// Mirrors parsec-macos.pkg 150.101.1: the bundle id is repeated in empty
		// <bundle> elements under <bundle-version>, <upgrade-bundle>, and others.
		const pkgInfoXML = `<?xml version="1.0" encoding="utf-8"?>
<pkg-info overwrite-permissions="true" relocatable="false" identifier="tv.parsec.www" postinstall-action="none" version="0" format-version="2" install-location="/Applications" auth="root">
    <payload numberOfFiles="13" installKBytes="6526"/>
    <bundle path="./Parsec.app" id="tv.parsec.www" CFBundleShortVersionString="150.101.1" CFBundleVersion="150.101.1"/>
    <bundle-version>
        <bundle id="tv.parsec.www"/>
    </bundle-version>
    <upgrade-bundle>
        <bundle id="tv.parsec.www"/>
    </upgrade-bundle>
    <scripts>
        <preinstall file="./preinstall" timeout="600"/>
    </scripts>
</pkg-info>`

		url := serveRangedPkg(t, buildFlatPkgWithFile(t, "PackageInfo", pkgInfoXML))

		got, err := ParsecPackageAppVersion(t.Context(), url)
		require.NoError(t, err)
		require.Equal(t, "150.101.1", got)
	})

	t.Run("no short version falls back to CFBundleVersion", func(t *testing.T) {
		const pkgInfoXML = `<pkg-info identifier="tv.parsec.www">
    <bundle path="./Parsec.app" id="tv.parsec.www" CFBundleVersion="150.102.0"/>
</pkg-info>`

		url := serveRangedPkg(t, buildFlatPkgWithFile(t, "PackageInfo", pkgInfoXML))

		got, err := ParsecPackageAppVersion(t.Context(), url)
		require.NoError(t, err)
		require.Equal(t, "150.102.0", got)
	})

	t.Run("no versioned Parsec bundle", func(t *testing.T) {
		const pkgInfoXML = `<pkg-info identifier="tv.parsec.www">
    <bundle path="./Other.app" id="com.example.other" CFBundleShortVersionString="1.0"/>
    <bundle-version>
        <bundle id="tv.parsec.www"/>
    </bundle-version>
</pkg-info>`

		url := serveRangedPkg(t, buildFlatPkgWithFile(t, "PackageInfo", pkgInfoXML))

		_, err := ParsecPackageAppVersion(t.Context(), url)
		require.ErrorContains(t, err, parsecAppBundleID)
	})

	t.Run("no PackageInfo file", func(t *testing.T) {
		url := serveRangedPkg(t, buildFlatPkgWithFile(t, "Distribution", `<installer-script/>`))

		_, err := ParsecPackageAppVersion(t.Context(), url)
		require.ErrorContains(t, err, "PackageInfo")
	})
}
