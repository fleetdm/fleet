package tables

import (
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

	applyNext(t, db)

	err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM host_mdm_apple_profiles hmap JOIN hosts h WHERE h.uuid = hmap.host_uuid AND hmap.status = 'failed'")
	require.NoError(t, err)

	// verify that there are no tables with the wrong collation
	var names []string
	err = sqlx.Select(db, &names, `
          SELECT table_name
          FROM information_schema.TABLES
	  WHERE table_collation != ? AND table_schema = (SELECT database())`, "utf8mb4_general_ci")
	require.NoError(t, err)
	require.Empty(t, names)
}
