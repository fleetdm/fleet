// Package tables holds fleet table migrations.
//
// Migrations can be tested with tests following the following format:
//
//	$ cat 20220208144831_AddSoftwareReleaseArchVendorColumns_test.go
//
//	[...]
//	func TestUp_20220208144831(t *testing.T) {
//		// Apply all migrations up to 20220208144831 (name of test), not included.
//		db := applyUpToPrev(t)
//
//		// insert testing data, etc.
//
//		// The following will apply migration 20220208144831.
//		applyNext(t, db)
//
//		// insert testing data, verify migration.
//	}
package tables

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(lucas): I'm copy pasting some of the mysql functionality methods here
// otherwise we have import cycle errors.
//
// We need to decouple the server/datastore/mysql package,
// it contains both the implementation of the fleet.Datastore and
// MySQL functionality, and MySQL test functionality.
const (
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

func newDBConnForTests(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8mb4&parseTime=true&loc=UTC&multiStatements=true", testUsername, testPassword, testAddress),
	)
	require.NoError(t, err)

	name := strings.ReplaceAll(strings.ReplaceAll(t.Name(), "/", "_"), " ", "_")
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s;", name, name, name))
	require.NoError(t, err)
	return db
}

func getMigrationVersion(t *testing.T) int64 {
	// Migration test functions look like this:
	//   func TestUp_20231109115838(t *testing.T)
	//
	// and multiple unit tests for the same migration version can be done by
	// following this naming pattern:
	//   func TestUp_20231109115838_scenario1(t *testing.T)
	//   func TestUp_20231109115838_scenario2(t *testing.T)
	//
	// Note that sub-tests can also be used, so:
	//   func TestUp_20231109115838(t *testing.T) {
	//     t.Run("scenario1", func(t *testing.T) {...}
	//   }
	// also works (calling applyUpToPrev in each sub-test to create a new test
	// database).
	//
	// This extracts the migration version (timestamp) from the test name.

	baseName, _, _ := strings.Cut(t.Name(), "/")
	withoutPrefix := strings.TrimPrefix(baseName, "TestUp_")
	timestampPart, _, _ := strings.Cut(withoutPrefix, "_")
	v, err := strconv.Atoi(timestampPart)
	require.NoError(t, err)
	return int64(v)
}

// applyUpToPrev will allocate a testing DB connection and apply
// migrations up to, not including, the migration specified in the test name.
//
// It returns the database connection to perform additional queries and migrations.
func applyUpToPrev(t *testing.T) *sqlx.DB {
	// Run migration tests up to 2 months old. Our releases are on a 3-week
	// cadence so this safely catches every migration in the release with a bit
	// of buffer in case of delayed releases.
	const maxMigrationTestAge = 60 * 24 * time.Hour

	v := getMigrationVersion(t)
	testDateTime, err := time.Parse("20060102150405", strconv.FormatInt(v, 10))
	if err == nil && time.Since(testDateTime) > maxMigrationTestAge {
		t.Skip("Skipping migration test for old migration, DB migrations are immutable so once tested for a release they don't need to be tested again.")
	}

	db := newDBConnForTests(t)
	for {
		current, err := MigrationClient.GetDBVersion(db.DB)
		require.NoError(t, err)
		next, err := MigrationClient.Migrations.Next(current)
		require.NoError(t, err)
		if next.Version == v {
			return db
		}
		applyNext(t, db)
	}
}

func execNoErrLastID(t *testing.T, db *sqlx.DB, query string, args ...any) int64 {
	res, err := db.Exec(query, args...)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return id
}

func execNoErr(t *testing.T, db *sqlx.DB, query string, args ...any) {
	execNoErrLastID(t, db, query, args...)
}

// applyNext performs the next migration in the chain.
func applyNext(t *testing.T, db *sqlx.DB) {
	// gooseNoDir is the value to not parse local files and instead use
	// the migrations that were added manually via Add().
	const gooseNoDir = ""
	err := MigrationClient.UpByOne(db.DB, gooseNoDir)
	require.NoError(t, err)
}

func insertQuery(t *testing.T, db *sqlx.DB) uint {
	// Insert a record into queries table
	insertQueryStmt := `
		INSERT INTO queries (
			name, description, query, observer_can_run, platform, logging_type
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	queryName := "Test Query"
	queryDescription := "A test query for the test suite"
	queryValue := "SELECT * FROM apps;"
	observerCanRun := 0
	platform := "mac" // Just a placeholder, adjust as needed
	loggingType := "snapshot"

	res, err := db.Exec(insertQueryStmt, queryName, queryDescription, queryValue, observerCanRun, platform, loggingType)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return uint(id) //nolint:gosec // dismiss G115
}

func checkCollation(t *testing.T, db *sqlx.DB) {
	type collationData struct {
		CollationName    string `db:"COLLATION_NAME"`
		TableName        string `db:"TABLE_NAME"`
		ColumnName       string `db:"COLUMN_NAME"`
		CharacterSetName string `db:"CHARACTER_SET_NAME"`
	}

	stmt := `
SELECT
	TABLE_NAME, COLLATION_NAME, COLUMN_NAME, CHARACTER_SET_NAME 
FROM information_schema.columns
WHERE
	TABLE_SCHEMA = (SELECT DATABASE()) 
	AND (CHARACTER_SET_NAME != ? OR COLLATION_NAME != ?)`

	var nonStandardCollations []collationData
	err := db.Select(&nonStandardCollations, stmt, "utf8mb4", "utf8mb4_unicode_ci")
	require.NoError(t, err)

	exceptions := []collationData{
		{"utf8mb4_bin", "enroll_secrets", "secret", "utf8mb4"},
		{"utf8mb4_bin", "hosts", "node_key", "utf8mb4"},
		{"utf8mb4_bin", "hosts", "orbit_node_key", "utf8mb4"},
		{"utf8mb4_bin", "teams", "name_bin", "utf8mb4"},
	}

	require.ElementsMatch(t, exceptions, nonStandardCollations)
}

func insertHost(t *testing.T, db *sqlx.DB, teamID *uint) uint {
	// Insert a minimal record into hosts table
	insertHostStmt := `
		INSERT INTO hosts (
			hostname, uuid, platform, osquery_version, os_version, build, platform_like, code_name,
			cpu_type, cpu_subtype, cpu_brand, hardware_vendor, hardware_model, hardware_version,
			hardware_serial, computer_name, team_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	hostName := "Dummy Hostname"
	hostUUID := "12345678-1234-1234-1234-123456789012"
	hostPlatform := "windows"
	osqueryVer := "5.9.1"
	osVersion := "Windows 10"
	buildVersion := "10.0.19042.1234"
	platformLike := "windows"
	codeName := "20H2"
	cpuType := "x86_64"
	cpuSubtype := "x86_64"
	cpuBrand := "Intel"
	hwVendor := "Dell Inc."
	hwModel := "OptiPlex 7090"
	hwVersion := "1.0"
	hwSerial := "ABCDEFGHIJ"
	computerName := "DESKTOP-TEST"

	res, err := db.Exec(insertHostStmt, hostName, hostUUID, hostPlatform, osqueryVer, osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand, hwVendor, hwModel, hwVersion, hwSerial, computerName, teamID)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return uint(id) //nolint:gosec // dismiss G115
}

// insertHosts inserts the specified number of hosts per platform. Note that
// macOS hosts will have their enrollment information inserted as well in the
// nano tables. It returns the host IDs of each platform, and the map of IDs to
// host UUIDs.
func insertHosts(t *testing.T, db *sqlx.DB, numMacOS, numWin, numLinux int) (macIDs, winIDs, linuxIDs []uint, idsToUUIDs map[uint]string) {
	const insertHostStmt = `
		INSERT INTO hosts (
			hostname, uuid, platform, osquery_version, os_version, build, platform_like, code_name,
			cpu_type, cpu_subtype, cpu_brand, hardware_vendor, hardware_model, hardware_version,
			hardware_serial, computer_name, team_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	perPlatformCounts := map[string]int{"darwin": numMacOS, "windows": numWin, "linux": numLinux}
	perPlatformOS := map[string]string{"darwin": "macOS 15.1", "windows": "Windows 11", "linux": "Ubuntu 24.04"}
	perPlatformIDs := map[string][]uint{"darwin": macIDs, "windows": winIDs, "linux": linuxIDs}
	perPlatformUUIDs := make(map[string][]string)
	idsToUUIDs = make(map[uint]string, numMacOS+numWin+numLinux)

	for platform, count := range perPlatformCounts {
		for i := 0; i < count; i++ {
			// Insert a minimal record into hosts table
			hostName := fmt.Sprintf("host-%s-%d", platform, i)
			hostUUID := uuid.NewString()
			hostPlatform := platform
			osqueryVer := "5.9.1"
			osVersion := perPlatformOS[platform]
			buildVersion := "10.0.19042.1234"
			platformLike := platform
			codeName := "20H2"
			cpuType := "x86_64"
			cpuSubtype := "x86_64"
			cpuBrand := "Intel"
			hwVendor := "Dell Inc."
			hwModel := "OptiPlex 7090"
			hwVersion := "1.0"
			hwSerial := uuid.NewString()
			computerName := fmt.Sprintf("DESKTOP-%s-%d", platform, i)

			id := execNoErrLastID(t, db, insertHostStmt, hostName, hostUUID, hostPlatform, osqueryVer,
				osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand,
				hwVendor, hwModel, hwVersion, hwSerial, computerName, nil)

			perPlatformIDs[platform] = append(perPlatformIDs[platform], uint(id)) // nolint:gosec
			perPlatformUUIDs[platform] = append(perPlatformUUIDs[platform], hostUUID)
			idsToUUIDs[uint(id)] = hostUUID // nolint:gosec
		}
	}

	for _, uid := range perPlatformUUIDs["darwin"] {
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate)
			VALUES (?, ?)`, uid, "auth")
		execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, uid, uid, "device", "topic", "magic", "hex", time.Now())
	}

	return perPlatformIDs["darwin"], perPlatformIDs["windows"], perPlatformIDs["linux"], idsToUUIDs
}

func insertScriptContents(t *testing.T, db *sqlx.DB, count int) []uint {
	ids := make([]uint, 0, count)
	for i := 0; i < count; i++ {
		content := fmt.Sprintf(`echo %d`, i)
		csum := md5ChecksumScriptContent(content)
		id := execNoErrLastID(t, db, `INSERT INTO script_contents
			(md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum, content)
		ids = append(ids, uint(id)) //nolint:gosec
	}
	return ids
}

// returns the installer IDs and the title IDs
func insertSoftwareInstallers(t *testing.T, db *sqlx.DB, count int) (installerIDs, titleIDs []uint) {
	installerIDs = make([]uint, 0, count)
	titleIDs = make([]uint, 0, count)

	for i := 0; i < count; i++ {
		content := fmt.Sprintf(`install %d`, i)
		csum := md5ChecksumScriptContent(content)
		installID := execNoErrLastID(t, db, `INSERT INTO script_contents
			(md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum, content)

		content = fmt.Sprintf(`uninstall %d`, i)
		csum = md5ChecksumScriptContent(content)
		uninstallID := execNoErrLastID(t, db, `INSERT INTO script_contents
			(md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum, content)

		titleID := execNoErrLastID(t, db, `INSERT INTO software_titles
			(name, source, browser) VALUES (?, 'apps', '')`, fmt.Sprintf("Foo%d.app", i))
		installerID := execNoErrLastID(t, db, `INSERT INTO software_installers
			(title_id, filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id)
			VALUES (?, ?, '1.1', 'darwin', ?, ?, '', ?)`, titleID, fmt.Sprintf("foo-%d.pkg", i), installID, fmt.Sprintf("storage-%d", i), uninstallID)

		installerIDs = append(installerIDs, uint(installerID)) //nolint:gosec
		titleIDs = append(titleIDs, uint(titleID))             //nolint:gosec
	}

	return installerIDs, titleIDs
}

func insertVPPApps(t *testing.T, db *sqlx.DB, count int, platform string) (adamIDs []string, titleIDs []uint) {
	adamIDs = make([]string, 0, count)
	titleIDs = make([]uint, 0, count)

	for i := 0; i < count; i++ {
		titleID := execNoErrLastID(t, db, `INSERT INTO software_titles
			(name, source, browser) VALUES (?, 'apps', '')`, fmt.Sprintf("Bar%d.app", i))
		adamID := fmt.Sprintf("adam-%d", i)
		execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id)
			VALUES (?, ?, ?)`, adamID, platform, titleID)

		adamIDs = append(adamIDs, adamID)
		titleIDs = append(titleIDs, uint(titleID)) //nolint:gosec
	}

	return adamIDs, titleIDs
}

func assertRowCount(t *testing.T, db *sqlx.DB, table string, count int) {
	var n int
	err := db.Get(&n, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	require.NoError(t, err)
	assert.Equal(t, count, n)
}

func insertAppleConfigProfile(t *testing.T, db *sqlx.DB, name, identifier string, vars ...string) string {
	contents := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>PayloadContent</key>
        <array>
            <dict>
							%s
            </dict>
        </array>
        <key>PayloadDisplayName</key>
        <string>%s</string>
        <key>PayloadIdentifier</key>
        <string>%s</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>%s</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
    </dict>
</plist>`
	var varDict strings.Builder
	for i, v := range vars {
		// add some vars with "${...}" and some with "$..."
		if i%2 == 0 {
			v = fmt.Sprintf("{%s}", v)
		}
		varDict.WriteString(fmt.Sprintf(`
						<key>Var%d</key>
						<string>$%s</string>`, i, v))
	}
	profileUUID := uuid.NewString()
	contents = fmt.Sprintf(contents, varDict.String(), name, identifier, profileUUID)

	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, identifier, name, mobileconfig, checksum)
		VALUES (?, ?, ?, ?, UNHEX(MD5(?)))`, profileUUID, identifier, name, contents, contents)
	return profileUUID
}
