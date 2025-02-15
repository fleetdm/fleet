package testing_utils

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	TestUsername              = "root"
	TestPassword              = "toor"
	TestAddress               = "localhost:3307"
	TestReplicaDatabaseSuffix = "_replica"
	TestReplicaAddress        = "localhost:3310"
)

// TruncateTables truncates the specified tables, in order, using ds.writer.
// Note that the order is typically not important because FK checks are
// disabled while truncating. If no table is provided, all tables (except
// those that are seeded by the SQL schema file) are truncated.
func TruncateTables(t testing.TB, db *sqlx.DB, logger log.Logger, nonEmptyTables map[string]bool, tables ...string) {
	// By setting DISABLE_TRUNCATE_TABLES a developer can troubleshoot tests
	// by inspecting mysql tables.
	if os.Getenv("DISABLE_TRUNCATE_TABLES") != "" {
		return
	}

	ctx := context.Background()

	require.NoError(t, common_mysql.WithTxx(ctx, db, func(tx sqlx.ExtContext) error {
		var skipSeeded bool

		if len(tables) == 0 {
			skipSeeded = true
			sql := `
      SELECT
        table_name
      FROM
        information_schema.tables
      WHERE
        table_schema = database() AND
        table_type = 'BASE TABLE'
    `
			if err := sqlx.SelectContext(ctx, tx, &tables, sql); err != nil {
				return err
			}
		}

		if _, err := tx.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS=0`); err != nil {
			return err
		}
		for _, tbl := range tables {
			if nonEmptyTables[tbl] {
				if skipSeeded {
					continue
				}
				return fmt.Errorf("cannot truncate table %s, it contains seed data from schema.sql", tbl)
			}
			if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE "+tbl); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS=1`); err != nil {
			return err
		}
		return nil
	}, logger))
}
