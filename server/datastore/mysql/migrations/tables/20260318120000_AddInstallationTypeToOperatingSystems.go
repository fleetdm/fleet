package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260318120000, Down_20260318120000)
}

func Up_20260318120000(tx *sql.Tx) error {
	// Add installation_type column to differentiate Windows Server Core from
	// full desktop installations for MSRC vulnerability matching.
	// Values: "" (unknown), "Client", "Server", "Server Core".
	//
	// The unique index key length limit is 3072 bytes for utf8mb4. The current
	// index uses 3060 bytes. To fit installation_type (VARCHAR(20) = 80 bytes),
	// we shrink arch from VARCHAR(150) to VARCHAR(25) — real values are at most
	// ~20 chars (e.g. "ARM 64-bit Processor", "x86_64"). This frees 500 bytes.
	if _, err := tx.Exec(`
		ALTER TABLE operating_systems
			MODIFY COLUMN arch VARCHAR(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			ADD COLUMN installation_type VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			DROP INDEX idx_unique_os,
			ADD UNIQUE INDEX idx_unique_os (name, version, arch, kernel_version, platform, display_version, installation_type)
	`); err != nil {
		return fmt.Errorf("adding installation_type to operating_systems: %w", err)
	}
	return nil
}

func Down_20260318120000(tx *sql.Tx) error {
	return nil
}
