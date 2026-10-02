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

func TestServiceEnvironment(t *testing.T) {
	t.Run("minimal options", func(t *testing.T) {
		assert.Equal(t, []string{
			"ORBIT_ROOT_DIR=[ORBITROOT].",
			`ORBIT_LOG_FILE=[System64Folder]config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log`,
			"ORBIT_FLEET_URL=[FLEET_URL]",
			"ORBIT_ENROLL_SECRET_PATH=[ORBITROOT]secret.txt",
			"ORBIT_UPDATE_URL=https://updates.fleetdm.com",
			"ORBIT_UPDATE_INTERVAL=15m0s",
			"ORBIT_FLEET_DESKTOP=[FLEET_DESKTOP]",
			"ORBIT_DESKTOP_CHANNEL=stable",
			"ORBIT_ORBIT_CHANNEL=stable",
			"ORBIT_OSQUERYD_CHANNEL=stable",
			"ORBIT_ENABLE_SCRIPTS=[ENABLE_SCRIPTS]",
		}, serviceEnvironment(baseOptions()))
	})

	t.Run("all options", func(t *testing.T) {
		opt := baseOptions()
		opt.FleetCertificate = true
		opt.Insecure = true
		opt.Debug = true
		opt.UpdateTLSServerCertificate = true
		opt.DisableUpdates = true
		opt.FleetDesktopAlternativeBrowserHost = "localhost:8080"
		opt.HostIdentifier = "instance"
		opt.EnableEndUserEmailProperty = true
		opt.EnableEUATokenProperty = true
		opt.OsqueryDB = `C:\osquery.db`
		opt.DisableSetupExperience = true
		opt.BypassEndUserAuth = true

		env := serviceEnvironment(opt)
		for _, want := range []string{
			"ORBIT_FLEET_CERTIFICATE=[ORBITROOT]fleet.pem",
			"ORBIT_INSECURE=true",
			"ORBIT_DEBUG=true",
			"ORBIT_UPDATE_TLS_CERTIFICATE=[ORBITROOT]update.pem",
			"ORBIT_DISABLE_UPDATES=true",
			"ORBIT_FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST=localhost:8080",
			"ORBIT_HOST_IDENTIFIER=instance",
			"ORBIT_END_USER_EMAIL=[END_USER_EMAIL]",
			"ORBIT_EUA_TOKEN=[EUA_TOKEN]",
			`ORBIT_OSQUERY_DB=C:\osquery.db`,
			"ORBIT_DISABLE_SETUP_EXPERIENCE=true",
			"ORBIT_BYPASS_END_USER_AUTH=true",
		} {
			assert.Contains(t, env, want)
		}
		for _, e := range env {
			assert.Regexp(t, `^ORBIT_[A-Z_]+=`, e)
			assert.NotContains(t, e, "[~]", "entries must not contain the multi-string separator")
		}
	})

	t.Run("uuid host identifier is orbit's default and omitted", func(t *testing.T) {
		opt := baseOptions()
		opt.HostIdentifier = "uuid"
		for _, e := range serviceEnvironment(opt) {
			assert.NotContains(t, e, "ORBIT_HOST_IDENTIFIER")
		}
	})

	t.Run("literal end user email without the MSI property", func(t *testing.T) {
		opt := baseOptions()
		opt.EndUserEmail = "user@example.com"
		assert.Contains(t, serviceEnvironment(opt), "ORBIT_END_USER_EMAIL=user@example.com")
	})

	t.Run("eua token and bypass end user auth toggle", func(t *testing.T) {
		opt := baseOptions()
		assert.NotContains(t, serviceEnvironment(opt), "ORBIT_EUA_TOKEN=[EUA_TOKEN]")
		assert.NotContains(t, serviceEnvironment(opt), "ORBIT_BYPASS_END_USER_AUTH=true")
		opt.EnableEUATokenProperty = true
		opt.BypassEndUserAuth = true
		assert.Contains(t, serviceEnvironment(opt), "ORBIT_EUA_TOKEN=[EUA_TOKEN]")
		assert.Contains(t, serviceEnvironment(opt), "ORBIT_BYPASS_END_USER_AUTH=true")
	})
}

// populateForTest runs populate with a minimal harvest containing only the
// authored orbit.exe.
func populateForTest(t *testing.T, opt Options) *database {
	t.Helper()
	db := newDatabase()
	require.NoError(t, db.addValidationRows())
	h := &harvest{dirByID: map[string]*harvestedDir{}}
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

func tableRows(t *testing.T, db *database, name string) [][]cell {
	t.Helper()
	tbl, ok := db.tables[name]
	require.True(t, ok, "table %s not found", name)
	require.NotNil(t, tbl)
	return tbl.rows
}

func propertyValues(t *testing.T, db *database) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, row := range tableRows(t, db, "Property") {
		out[row[0].(string)] = row[1].(string)
	}
	return out
}

func TestEUATokenProperty(t *testing.T) {
	opt := baseOptions()
	opt.EnableEUATokenProperty = true
	assert.Contains(t, propertyValues(t, populateForTest(t, opt)), "EUA_TOKEN")
	opt.EnableEUATokenProperty = false
	assert.NotContains(t, propertyValues(t, populateForTest(t, opt)), "EUA_TOKEN")
}

func TestServiceConfiguredThroughEnvironment(t *testing.T) {
	opt := baseOptions()
	opt.BypassEndUserAuth = true
	db := populateForTest(t, opt)

	// Expected id and value layout taken from a WiX 3.14-built fleetd MSI.
	rows := tableRows(t, db, "Registry")
	require.Len(t, rows, 1)
	assert.Equal(t, []cell{
		"regB26658E6B8B0ACF031C982078F375B22", 2, `SYSTEM\CurrentControlSet\Services\Fleet osquery`, "Environment",
		strings.Join(serviceEnvironment(opt), "[~]"), "C_ORBITBIN",
	}, rows[0])
	assert.Contains(t, rows[0][4], "[~]ORBIT_BYPASS_END_USER_AUTH=true")

	svc := tableRows(t, db, "ServiceInstall")
	require.Len(t, svc, 1)
	assert.Nil(t, svc[0][10], "orbit must be configured via environment, not ServiceInstall Arguments")

	assert.NotContains(t, db.tables, "Environment", "the system-wide Environment table is no longer used")

	var actions []string
	for _, row := range tableRows(t, db, "InstallExecuteSequence") {
		actions = append(actions, row[0].(string))
	}
	assert.Contains(t, actions, "WriteRegistryValues")
	assert.Contains(t, actions, "RemoveRegistryValues")
	assert.NotContains(t, actions, "WriteEnvironmentStrings")
}

func TestEnrollSecretHidden(t *testing.T) {
	db := populateForTest(t, baseOptions())
	assert.Equal(t, "CA_UpdateSecret;FLEET_SECRET", propertyValues(t, db)["MsiHiddenProperties"])
	for _, row := range tableRows(t, db, "CustomAction") {
		if row[0] == "CA_UpdateSecret" {
			assert.Equal(t, 3073|customActionTypeHideTarget, row[1])
			return
		}
	}
	t.Fatal("CA_UpdateSecret custom action not found")
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
