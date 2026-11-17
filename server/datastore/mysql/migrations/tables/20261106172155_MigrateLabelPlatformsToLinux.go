package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261106172155, Down_20261106172155)
}

// Up_20261106172155 moves dynamic labels targeting the retired "ubuntu" and "centos" platforms to
// "linux". Memberships are kept: hosts that matched on Ubuntu/CentOS still match under "linux", and
// clearing them would briefly unscope profiles and software that use these labels.
func Up_20261106172155(tx *sql.Tx) error {
	if _, err := tx.Exec(`UPDATE labels SET platform = 'linux' WHERE platform IN ('ubuntu', 'centos')`); err != nil {
		return fmt.Errorf("migrate ubuntu and centos label platforms to linux: %w", err)
	}
	return nil
}

func Down_20261106172155(tx *sql.Tx) error {
	return nil
}
