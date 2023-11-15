package tables

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20230315104937(t *testing.T) {
	db := applyUpToPrev(t)
	_, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	require.NoError(t, err)

	_, err = db.Exec("DROP TABLE mdm_apple_delivery_status, mdm_apple_operation_types, host_mdm_apple_profiles")
	require.NoError(t, err)

	_, err = db.Exec("CREATE TABLE mdm_apple_delivery_status (status VARCHAR(20) PRIMARY KEY) CHARSET=utf8mb4 COLLATE=utf8mb4_danish_ci")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO mdm_apple_delivery_status (status) VALUES ('failed'), ('applied'), ('pending')")
	require.NoError(t, err)

	_, err = db.Exec("CREATE TABLE mdm_apple_operation_types (operation_type VARCHAR(20) PRIMARY KEY) CHARSET=utf8mb4 COLLATE=utf8mb4_danish_ci")
	require.NoError(t, err)

	_, err = db.Exec("CREATE TABLE host_mdm_apple_profiles (profile_id int(10) UNSIGNED NOT NULL, profile_identifier varchar(255) NOT NULL, host_uuid varchar(255) NOT NULL, status varchar(20) DEFAULT NULL, operation_type varchar(20) DEFAULT NULL, detail text, command_uuid        varchar(127) NOT NULL, PRIMARY KEY (host_uuid, profile_id), FOREIGN KEY (status) REFERENCES mdm_apple_delivery_status (status) ON UPDATE CASCADE, FOREIGN KEY (operation_type) REFERENCES mdm_apple_operation_types (operation_type) ON UPDATE CASCADE) CHARSET=utf8mb4 COLLATE=utf8mb4_danish_ci")
	require.NoError(t, err)

	_, err = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	require.NoError(t, err)

	// force a query with an error
	var c int
	err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM host_mdm_apple_profiles hmap JOIN hosts h WHERE h.uuid = hmap.host_uuid AND hmap.status = 'failed'")
	require.ErrorContains(t, err, "Error 1267")

	var mysqlVersion string
	err = sqlx.Get(db, &mysqlVersion, "SELECT VERSION()")
	require.NoError(t, err)

	// this test can only be replicated in MySQL 8 because for prior
	// versions all collations are padded.
	if strings.HasPrefix(mysqlVersion, "8") {
		// ensure software is using a different collation
		_, err = db.Exec("ALTER TABLE `software` CONVERT TO CHARACTER SET `utf8mb4` COLLATE `utf8mb4_0900_ai_ci`")
		require.NoError(t, err)

		// insert two software records
		insertSoftwareStmt := `INSERT INTO software (name, version, source, bundle_identifier, vendor, arch) VALUES (?, '1.2.1', 'rpm_packages', '', ?, 'x86_64')`
		_, err = db.Exec(insertSoftwareStmt, "zchunk-libs", "vendor")
		require.NoError(t, err)
		_, err = db.Exec(insertSoftwareStmt, "zchunk-libs", "vendor ")
		require.NoError(t, err)
		_, err = db.Exec(insertSoftwareStmt, "vim", "vendor")
		require.NoError(t, err)
		_, err = db.Exec(insertSoftwareStmt, "vim", "vendor ")
		require.NoError(t, err)

		// insert host_users
		_, err = db.Exec("ALTER TABLE `host_users` CONVERT TO CHARACTER SET `utf8mb4` COLLATE `utf8mb4_0900_ai_ci`")
		require.NoError(t, err)
		insertHostUsersStmt := `INSERT INTO host_users (host_id, uid, username) VALUES (?, 1, ?)`
		_, err = db.Exec(insertHostUsersStmt, 1, "username")
		require.NoError(t, err)
		_, err = db.Exec(insertHostUsersStmt, 1, "username ")
		require.NoError(t, err)
		_, err = db.Exec(insertHostUsersStmt, 2, "username")
		require.NoError(t, err)
		_, err = db.Exec(insertHostUsersStmt, 2, "username ")
		require.NoError(t, err)

		// insert operating_systems
		_, err = db.Exec("ALTER TABLE `operating_systems` CONVERT TO CHARACTER SET `utf8mb4` COLLATE `utf8mb4_0900_ai_ci`")
		require.NoError(t, err)
		insertOSStmt := `INSERT INTO operating_systems (name,version,arch,kernel_version,platform) VALUES (?, '12.1', 'arch', 'kernel', ?)`
		_, err = db.Exec(insertOSStmt, "macOS", "darwin")
		require.NoError(t, err)
		_, err = db.Exec(insertOSStmt, "macOS", "darwin ")
		require.NoError(t, err)
		_, err = db.Exec(insertOSStmt, "arch", "linux")
		require.NoError(t, err)
		_, err = db.Exec(insertOSStmt, "arch", "linux ")
		require.NoError(t, err)
	}

	applyNext(t, db)

	err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM host_mdm_apple_profiles hmap JOIN hosts h WHERE h.uuid = hmap.host_uuid AND hmap.status = 'failed'")
	require.NoError(t, err)

	// verify that there are no tables with the wrong collation
	var names []string
	err = sqlx.Select(db, &names, `
          SELECT table_name
          FROM information_schema.TABLES
	  WHERE table_collation != "utf8mb4_unicode_ci" AND table_schema = (SELECT database())`)
	require.NoError(t, err)
	require.Empty(t, names)

	// verify that the collation was maintained for certain columns
	var columns []string
	err = sqlx.Select(db, &columns, `
	  SELECT column_name
	  FROM information_schema.COLUMNS
	  WHERE collation_name != "utf8mb4_unicode_ci" AND table_schema = (SELECT database())`)
	require.NoError(t, err)
	require.Equal(t, []string{"secret", "node_key", "orbit_node_key"}, columns)

	if strings.HasPrefix(mysqlVersion, "8") {
		// verify that duplicate columns have been removed
		c = 0
		err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM software")
		require.NoError(t, err)
		require.Equal(t, 2, c)

		c = 0
		err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM host_users")
		require.NoError(t, err)
		require.Equal(t, 2, c)

		c = 0
		err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM operating_systems")
		require.NoError(t, err)
		require.Equal(t, 2, c)
	}
}
