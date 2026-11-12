package tables

import (
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// Not named TestUp_<version>, so applyUpToPrev's age-based skip never applies.
func TestFleet5Baseline(t *testing.T) {
	db := newDBConnForTests(t)
	applyNext(t, db)
	checkBaseline(t, db)
}

func TestFleet5BaselineRetry(t *testing.T) {
	db := newDBConnForTests(t)
	_, err := MigrationClient.GetDBVersion(db.DB)
	require.NoError(t, err)

	// Simulate a previous run that failed part way, leaving some tables and
	// seed rows behind without recording the migration.
	require.NoError(t, runBaseline(t, db))
	execNoErr(t, db, `DROP TABLE aggregated_stats`)
	execNoErr(t, db, `DELETE FROM labels WHERE id > 10`)

	applyNext(t, db)
	checkBaseline(t, db)
}

func TestFleet5BaselineFleet4Database(t *testing.T) {
	newFleet4DB := func(t *testing.T, tableVersions, dataVersions []int64) *sqlx.DB {
		db := newDBConnForTests(t)
		_, err := MigrationClient.GetDBVersion(db.DB)
		require.NoError(t, err)
		execNoErr(t, db, `CREATE TABLE migration_status_data (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL,
			tstamp TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP
		)`)
		for _, v := range tableVersions {
			execNoErr(t, db, `INSERT INTO migration_status_tables (version_id, is_applied) VALUES (?, 1)`, v)
		}
		for _, v := range dataVersions {
			execNoErr(t, db, `INSERT INTO migration_status_data (version_id, is_applied) VALUES (?, 1)`, v)
		}
		return db
	}
	last := len(Fleet4Migrations) - 1

	t.Run("fleet 4_95", func(t *testing.T) {
		db := newFleet4DB(t, Fleet4Migrations, fleet4DataMigrations)
		// A 4.95 database already has the schema, so the baseline only records itself.
		require.NoError(t, runBaseline(t, db))
		requireNoBaselineTables(t, db)
	})

	t.Run("fleet 4_95 legacy extras", func(t *testing.T) {
		db := newFleet4DB(t,
			slices.Concat(Fleet4Migrations, Fleet4OptionalMigrations),
			// A data migration removed in an old Fleet 4 release.
			append(slices.Clone(fleet4DataMigrations), 20171212182459),
		)
		require.NoError(t, runBaseline(t, db))
		requireNoBaselineTables(t, db)
	})

	t.Run("older than fleet 4_95", func(t *testing.T) {
		db := newFleet4DB(t, Fleet4Migrations[:last], fleet4DataMigrations)
		requireBaselineRefused(t, db, "1 of 690 Fleet 4.95.0 migrations not applied (first missing 20261102142138, latest applied 20261102134303)")
	})

	t.Run("fleet 4_94_0", func(t *testing.T) {
		// 4.94.0 shipped the first 666 Fleet 4.95 migrations.
		db := newFleet4DB(t, Fleet4Migrations[:666], fleet4DataMigrations)
		requireBaselineRefused(t, db, "24 of 690 Fleet 4.95.0 migrations not applied (first missing 20261014132040, latest applied 20261014074522)")
	})

	t.Run("missing a fleet 4_95 migration", func(t *testing.T) {
		db := newFleet4DB(t, slices.Concat(Fleet4Migrations[:10], Fleet4Migrations[11:]), fleet4DataMigrations)
		requireBaselineRefused(t, db, "1 of 690 Fleet 4.95.0 migrations not applied (first missing 20161118212621, latest applied 20261102142138)")
	})

	t.Run("missing a data migration", func(t *testing.T) {
		db := newFleet4DB(t, Fleet4Migrations, fleet4DataMigrations[:len(fleet4DataMigrations)-1])
		requireBaselineRefused(t, db, "Fleet 4 data migrations not applied (20230525175650)")
	})

	t.Run("newer than fleet 4_95", func(t *testing.T) {
		db := newFleet4DB(t, append(slices.Clone(Fleet4Migrations), Fleet4Migrations[last]+1), fleet4DataMigrations)
		requireBaselineRefused(t, db, "applied migrations not part of Fleet 4.95.0 (20261102142139)")
	})

	t.Run("many unknown migrations", func(t *testing.T) {
		extra := []int64{20261102142143, 20261102142139, 20261102142142, 20261102142140, 20261102142141}
		db := newFleet4DB(t, slices.Concat(Fleet4Migrations, extra), fleet4DataMigrations)
		requireBaselineRefused(t, db, "applied migrations not part of Fleet 4.95.0 (20261102142139, 20261102142140, 20261102142141, and 2 more)")
	})

	t.Run("every mismatch", func(t *testing.T) {
		db := newFleet4DB(t, append(slices.Clone(Fleet4Migrations[:last]), 20250101000000), nil)
		requireBaselineRefused(t, db, "1 of 690 Fleet 4.95.0 migrations not applied (first missing 20261102142138, latest applied 20261102134303); "+
			"applied migrations not part of Fleet 4.95.0 (20250101000000); "+
			"Fleet 4 data migrations not applied (20161229171615, 20170223171234, 20170301093653, and 6 more)")
	})

	t.Run("data migrations only", func(t *testing.T) {
		db := newFleet4DB(t, nil, fleet4DataMigrations)
		requireBaselineRefused(t, db, "690 of 690 Fleet 4.95.0 migrations not applied (first missing 20161118193812)")
	})
}

func TestFleet4MigrationsSorted(t *testing.T) {
	require.True(t, slices.IsSorted(Fleet4Migrations))
	require.Len(t, slices.Compact(slices.Clone(Fleet4Migrations)), len(Fleet4Migrations))
	require.Less(t, Fleet4Migrations[len(Fleet4Migrations)-1], int64(Fleet5BaselineVersion))
}

// runBaseline calls Up directly because goose exits the process when a migration fails.
func runBaseline(t *testing.T, db *sqlx.DB) error {
	tx, err := db.Begin()
	require.NoError(t, err)
	err = Up_20261103120000(tx)
	require.NoError(t, tx.Commit())
	return err
}

func requireBaselineRefused(t *testing.T, db *sqlx.DB, reason string) {
	err := runBaseline(t, db)
	require.EqualError(t, err, "this database must be upgraded to Fleet 4.95.0 or later before upgrading to Fleet 5: "+reason)
	requireNoBaselineTables(t, db)
}

func requireNoBaselineTables(t *testing.T, db *sqlx.DB) {
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'labels'`))
	require.Zero(t, count)
}

// The baseline is frozen, so these counts must never change.
func checkBaseline(t *testing.T, db *sqlx.DB) {
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'`))
	require.Equal(t, 243, count) // includes migration_status_tables

	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM labels WHERE label_type = ?`, fleet.LabelTypeBuiltIn))
	require.Equal(t, 13, count)
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM app_config_json`))
	require.Equal(t, 1, count)

	var collation string
	require.NoError(t, db.Get(&collation, `SELECT DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = DATABASE()`))
	require.Equal(t, "utf8mb4_unicode_ci", collation)

	// The view must not be tied to the user that ran the migration.
	var security string
	require.NoError(t, db.Get(&security, `SELECT SECURITY_TYPE FROM information_schema.VIEWS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'nano_view_queue'`))
	require.Equal(t, "INVOKER", security)

	var fkChecks int
	require.NoError(t, db.Get(&fkChecks, `SELECT @@FOREIGN_KEY_CHECKS`))
	require.Equal(t, 1, fkChecks)
}
