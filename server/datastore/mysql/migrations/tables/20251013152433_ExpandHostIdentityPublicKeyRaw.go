package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251013152433, Down_20251013152433)
}

func Up_20251013152433(tx *sql.Tx) error {
	// Expand public_key_raw column to support RSA keys in addition to ECC keys.
	// ECC keys (P-256/P-384) use uncompressed point format:
	//   - P-256: 65 bytes (1 byte prefix + 32 bytes X + 32 bytes Y)
	//   - P-384: 97 bytes (1 byte prefix + 48 bytes X + 48 bytes Y)
	// RSA keys use PKIX ASN.1 DER encoding:
	//   - RSA 2048: ~294 bytes
	//   - RSA 4096: ~550 bytes
	// Setting to 600 bytes to accommodate current and future key types.
	_, err := tx.Exec(`
		ALTER TABLE host_identity_scep_certificates
		MODIFY COLUMN public_key_raw VARBINARY(600) NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to expand public_key_raw column: %w", err)
	}

	return nil
}

func Down_20251013152433(tx *sql.Tx) error {
	// Note: Downgrade may fail if there are RSA keys in the database.
	// This is intentional as we cannot safely downgrade without data loss.
	_, err := tx.Exec(`
		ALTER TABLE host_identity_scep_certificates
		MODIFY COLUMN public_key_raw VARBINARY(100) NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to shrink public_key_raw column: %w", err)
	}

	return nil
}
