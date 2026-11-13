package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20261112145417, Down_20261112145417)
}

func Up_20261112145417(tx *sql.Tx) error {
	if tableExists(tx, "packs") {
		// Since 4.35 every remaining row is a user-created 2017 pack. Log what is
		// dropped so the upgrade log records it.
		rows, err := tx.Query(`SELECT name FROM packs ORDER BY name`)
		if err != nil {
			return fmt.Errorf("listing 2017 packs: %w", err)
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return fmt.Errorf("scanning 2017 pack name: %w", err)
			}
			names = append(names, name)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("listing 2017 packs: %w", err)
		}
		if len(names) > 0 {
			logger.Info.Printf("dropping %d 2017 packs: %s", len(names), strings.Join(names, ", "))
		}
	}

	// scheduled_queries and pack_targets reference packs, so drop them first.
	for _, table := range []string{"scheduled_queries", "pack_targets", "packs"} {
		if _, err := tx.Exec("DROP TABLE IF EXISTS `" + table + "`"); err != nil {
			return fmt.Errorf("dropping %s: %w", table, err)
		}
	}
	return nil
}

func Down_20261112145417(tx *sql.Tx) error {
	return nil
}
