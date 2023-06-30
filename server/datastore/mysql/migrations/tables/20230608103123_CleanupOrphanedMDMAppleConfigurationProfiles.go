package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20230608103123, Down_20230608103123)
}

func Up_20230608103123(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	r, err := txx.Exec(`
		DELETE macp
		FROM mdm_apple_configuration_profiles macp
		LEFT JOIN teams t
		ON t.id = macp.team_id
		WHERE t.id IS NULL AND macp.team_id != 0
	`)
	if err != nil {
		return fmt.Errorf("deleting orphaned configuration profiles: %w", err)
	}

	i, _ := r.RowsAffected()
	logger.Info.Printf("deleted %d rows without a matching team from mdm_apple_configuration_profiles\n", i)

	return nil
}

func Down_20230608103123(tx *sql.Tx) error {
	return nil
}
