package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// Test is for collation fix; uniQ migration didn't have a test before
func TestUp_20250127162751(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, "SET FOREIGN_KEY_CHECKS = 0")
	execNoErr(t, db, "DROP TABLE mdm_apple_bootstrap_packages")
	execNoErr(t, db, "CREATE TABLE mdm_apple_bootstrap_packages (team_id int(10) unsigned NOT NULL PRIMARY KEY, name varchar(255)) CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci")
	execNoErr(t, db, "INSERT INTO mdm_apple_bootstrap_packages (team_id, name) VALUES (1, 'Care Package')")
	execNoErr(t, db, "DROP TABLE host_mdm_apple_bootstrap_packages")
	execNoErr(t, db, "CREATE TABLE host_mdm_apple_bootstrap_packages (host_uuid VARCHAR(127) NOT NULL PRIMARY KEY) CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci")
	execNoErr(t, db, "INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid) VALUES ('a123b123')")
	execNoErr(t, db, "SET FOREIGN_KEY_CHECKS = 1")

	// force a query with an error
	var c int
	err := sqlx.Get(db, &c, "SELECT COUNT(*) FROM host_mdm_apple_bootstrap_packages hmabp JOIN hosts h WHERE h.uuid = hmabp.host_uuid")
	require.ErrorContains(t, err, "Error 1267")

	applyNext(t, db)

	err = sqlx.Get(db, &c, "SELECT COUNT(*) FROM host_mdm_apple_bootstrap_packages hmabp JOIN hosts h WHERE h.uuid = hmabp.host_uuid")
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
	require.ElementsMatch(t, []string{"secret", "node_key", "orbit_node_key", "name_bin"}, columns)
}
