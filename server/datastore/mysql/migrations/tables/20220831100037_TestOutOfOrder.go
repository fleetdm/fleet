
package tables

import (
    "database/sql"
)

func init() {
    MigrationClient.AddMigration(Up_20220916145156, Down_20220916145156)
}

func Up_20220916145156(tx *sql.Tx) error {
    return nil
}

func Down_20220916145156(tx *sql.Tx) error {
    return nil
}
