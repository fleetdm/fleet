package msi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The expected values in this file are taken verbatim from a WiX
// 3.14-generated fleetd MSI (fleetctl v4.89.2, --arch=arm64), so these tests
// pin the WiX compatibility of the identifier and short-name generators.

func TestWixIdentifier(t *testing.T) {
	root := wixIdentifier("dir", "ORBITROOT")

	binDir := wixIdentifier("dir", root, "bin")
	assert.Equal(t, "dirC7B0FE4712C41A16BEE94120136AC5BB", binDir)

	orbitDir := wixIdentifier("dir", binDir, "orbit")
	assert.Equal(t, "dir4C397B31F65DCA7A6A8966FA40599A6D", orbitDir)

	stagingDir := wixIdentifier("dir", root, "staging")
	assert.Equal(t, "dir8EA49721AE3FA2E0689208CD7A33ACD1", stagingDir)

	certsFile := wixIdentifier("fil", root, "certs.pem")
	assert.Equal(t, "filBD4431EBD09EAC47887B6474705D94B7", certsFile)

	certsComp := wixIdentifier("cmp", root, certsFile)
	assert.Equal(t, "cmpE28C2CD72ADB7E04540B944C4D9F5660", certsComp)

	// empty directories hash the directory id alone
	stagingComp := wixIdentifier("cmp", stagingDir)
	assert.Equal(t, "cmp3721680D11CEA9BEE8173E0F0A00BD49", stagingComp)
}

func TestWixShortName(t *testing.T) {
	// directories hash ("Directory", parentDirID), no extension
	assert.Equal(t, "2-lee-0b",
		wixShortName("windows-arm64", false, "Directory", "dir4C397B31F65DCA7A6A8966FA40599A6D"))
	assert.Equal(t, "7bdpo0-r",
		wixShortName("windows-arm64", false, "Directory", "dir73AA72A1EB32252D1814932E0A5D4B4B"))

	// files hash ("File", componentID) and keep up to a 4-char extension
	assert.Equal(t, "ula4zql4.ps1",
		wixShortName("installer_utils.ps1", true, "File", "cmpBA828A0C0B65B16C932B89D73E670818"))
	assert.Equal(t, "nmt5v-wl.fla",
		wixShortName("osquery.flags", true, "File", "cmpDDD8E061FF54F90449C7949D823A7011"))
	assert.Equal(t, "0_mfcvnu.jso",
		wixShortName("tuf-metadata.json", true, "File", "cmp49406D45DA2F4A02BCEC42D62BB4AB79"))
}

func TestNeedsShortName(t *testing.T) {
	for _, valid := range []string{"orbit.exe", "certs.pem", "secret.txt", "bin", "staging", "osquery.man"} {
		assert.False(t, needsShortName(valid), valid)
	}
	for _, invalid := range []string{
		"windows-arm64",         // basename longer than 8
		"osquery.flags",         // extension longer than 3
		"installer_utils.ps1",   // basename longer than 8
		"updates-metadata.json", // both
		"a b.txt",               // space
	} {
		assert.True(t, needsShortName(invalid), invalid)
	}
}

func baseOptions() Options {
	return Options{
		Architecture:        "arm64",
		Version:             "1.58.0",
		FleetURL:            "https://fleet.example.com",
		EnrollSecret:        true,
		DesktopChannel:      "stable",
		OrbitChannel:        "stable",
		OsquerydChannel:     "stable",
		UpdateURL:           "https://updates.fleetdm.com",
		OrbitUpdateInterval: "15m0s",
		NativePlatform:      "windows-arm64",
	}
}

func TestServiceArguments(t *testing.T) {
	t.Run("default arguments match the WiX template output", func(t *testing.T) {
		opt := baseOptions()
		opt.EnableEndUserEmailProperty = true
		opt.EnableEUATokenProperty = true
		// Reference string extracted from the ServiceInstall table of a WiX
		// 3.14-built fleetd MSI (byte for byte, including the double space
		// after --enable-scripts when no host identifier is set).
		want := `--root-dir "[ORBITROOT]." --log-file "[System64Folder]config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log" --fleet-url "[FLEET_URL]" --enroll-secret-path "[ORBITROOT]secret.txt" --update-url "https://updates.fleetdm.com" --fleet-desktop="[FLEET_DESKTOP]" --desktop-channel stable --orbit-channel "stable" --osqueryd-channel "stable" --enable-scripts="[ENABLE_SCRIPTS]"  --end-user-email="[END_USER_EMAIL]" --eua-token="[EUA_TOKEN]"`
		assert.Equal(t, want, serviceArguments(opt))
	})

	t.Run("eua token flag toggles with the property", func(t *testing.T) {
		opt := baseOptions()
		opt.EnableEUATokenProperty = true
		assert.Contains(t, serviceArguments(opt), ` --eua-token="[EUA_TOKEN]"`)
		opt.EnableEUATokenProperty = false
		assert.NotContains(t, serviceArguments(opt), "--eua-token")
	})

	t.Run("bypass end user auth", func(t *testing.T) {
		opt := baseOptions()
		opt.BypassEndUserAuth = true
		assert.Contains(t, serviceArguments(opt), " --bypass-end-user-auth")
		opt.BypassEndUserAuth = false
		assert.NotContains(t, serviceArguments(opt), "--bypass-end-user-auth")
	})

	t.Run("host identifier uuid is omitted", func(t *testing.T) {
		opt := baseOptions()
		opt.HostIdentifier = "uuid"
		assert.NotContains(t, serviceArguments(opt), "--host-identifier")
		opt.HostIdentifier = "instance"
		assert.Contains(t, serviceArguments(opt), `--host-identifier=instance`)
	})

	t.Run("optional flags", func(t *testing.T) {
		opt := baseOptions()
		opt.Insecure = true
		opt.Debug = true
		opt.DisableUpdates = true
		opt.FleetCertificate = true
		opt.UpdateTLSServerCertificate = true
		opt.OsqueryDB = `D:\osq`
		opt.DisableSetupExperience = true
		args := serviceArguments(opt)
		for _, want := range []string{
			` --insecure`, ` --debug`, ` --disable-updates`,
			` --fleet-certificate "[ORBITROOT]fleet.pem"`,
			` --update-tls-certificate "[ORBITROOT]update.pem"`,
			` --osquery-db="D:\osq"`, ` --disable-setup-experience`,
		} {
			assert.Contains(t, args, want)
		}
	})
}

func TestEUATokenProperty(t *testing.T) {
	// The EUA_TOKEN property row must exist exactly when the option is set
	// (formerly tested against the main.wxs template).
	build := func(enable bool) *database {
		opt := baseOptions()
		opt.EnableEUATokenProperty = enable
		db := newDatabase()
		require.NoError(t, db.addValidationRows())
		h := &harvest{dirByID: map[string]*harvestedDir{}}
		// populate requires the authored orbit.exe to exist in the harvest;
		// fake a minimal component for it.
		h.Components = []harvestedComponent{{File: &harvestedFile{
			ComponentID:   "cmpX",
			ComponentGUID: "{00000000-0000-4000-8000-000000000000}",
			FileID:        "filX",
			DirID:         orbitRootRef,
			FileName:      "orbit.exe",
			Path:          "root/bin/orbit/windows-arm64/stable/orbit.exe",
		}}}
		_, err := populate(db, opt, h, "{11111111-2222-4333-8444-555555555555}")
		require.NoError(t, err)
		return db
	}

	props := func(db *database) []string {
		var out []string
		for _, row := range db.tables["Property"].rows {
			out = append(out, row[0].(string))
		}
		return out
	}

	assert.Contains(t, props(build(true)), "EUA_TOKEN")
	assert.NotContains(t, props(build(false)), "EUA_TOKEN")
}

func TestSummaryTemplate(t *testing.T) {
	got, err := summaryTemplate("arm64")
	require.NoError(t, err)
	assert.Equal(t, "Arm64;1033", got)
	got, err = summaryTemplate("amd64")
	require.NoError(t, err)
	assert.Equal(t, "x64;1033", got)
	_, err = summaryTemplate("386")
	assert.Error(t, err)
}

func TestEncodeStreamName(t *testing.T) {
	// Round-trip sanity: encoded names use the packed MSI alphabet.
	for _, name := range []string{"!_StringPool", "!_Tables", "Binary.WixCA", "cab1.cab"} {
		enc := encodeStreamName(name)
		for _, r := range enc {
			assert.True(t, (r >= 0x3800 && r <= 0x4840) || r < 0x80, "unexpected rune %U in %q", r, enc)
		}
		if strings.HasPrefix(name, "!") {
			assert.Equal(t, rune(0x4840), []rune(enc)[0])
		}
	}
}
