package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20240802113716, Down_20240802113716)
}

func Up_20240802113716(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	type row struct {
		Config json.RawMessage `db:"config"`
		ID     uint            `db:"id"`
	}

	var rows []row
	if err := txx.Select(&rows, "SELECT config, id FROM teams"); err != nil {
		return fmt.Errorf("selecting team configs: %w", err)
	}

	for _, r := range rows {

		config := make(map[string]any)
		if err := json.Unmarshal(r.Config, &config); err != nil {
			return fmt.Errorf("unmarshal team config: %w", err)
		}
		softwareData, ok := config["software"]
		if !ok {
			continue
		}

		rt := reflect.TypeOf(config["software"])
		if rt == nil {
			continue
		}

		if rt.Kind() == reflect.Slice {
			// then we have an older config without the new fields
			// Note: we are setting the new key to be whatever the old key was (if it was null, then
			// it's set to null, if it was empty array, then it's set to empty array)
			config["software"] = map[string]any{"packages": softwareData}
			b, err := json.Marshal(config)
			if err != nil {
				return fmt.Errorf("marshal updated team config: %w", err)
			}
			if _, err := tx.Exec(`UPDATE teams SET config = ? WHERE id = ?`, b, r.ID); err != nil {
				return fmt.Errorf("updating config for team %d: %w", r.ID, err)
			}
		}

	}

	return nil
}

func Down_20240802113716(tx *sql.Tx) error {
	return nil
}
