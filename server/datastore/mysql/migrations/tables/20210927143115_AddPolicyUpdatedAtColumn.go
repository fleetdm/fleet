
package tables

import (
    "database/sql"
)

func init() {
    MigrationClient.AddMigration(Up_20210927143115, Down_20210927143115)
}

func Up_20210927143115(tx *sql.Tx) error {
    return nil
}

func Down_20210927143115(tx *sql.Tx) error {
    return nil
}
