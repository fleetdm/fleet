package tables

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260414185606, Down_20260414185606)
}

// certificateAssetNames is a snapshot of the asset names that hold certificates
// at the time this migration was written. This list is intentionally duplicated
// here (rather than referencing fleet.MDMAssetName constants) so the migration
// remains stable even if those constants change in the future.
var certificateAssetNames_AddRenewalToMDMConfigAssets = []string{
	"ca_cert",
	"apns_cert",
	"abm_cert",
	"host_identity_ca_cert",
	"conditional_access_ca_cert",
	"conditional_access_idp_cert",
}

func Up_20260414185606(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE mdm_config_assets ADD COLUMN renew_at TIMESTAMP NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding renew_at to mdm_config_assets: %w", err)
	}

	// Attempt to backfill renew_at for existing certificate assets.
	// This requires the server private key to decrypt asset values.
	privateKey := os.Getenv("FLEET_SERVER_PRIVATE_KEY")
	if privateKey == "" {
		// Without the key we cannot decrypt, skip the backfill. The values will
		// be populated the next time assets are replaced via the normal write path.
		return nil
	}

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	type row struct {
		ID    int64  `db:"id"`
		Value []byte `db:"value"`
	}

	query, args, err := sqlx.In(
		`SELECT id, value FROM mdm_config_assets WHERE name IN (?) AND deletion_uuid = ''`,
		certificateAssetNames_AddRenewalToMDMConfigAssets,
	)
	if err != nil {
		return fmt.Errorf("building IN query for mdm_config_assets backfill: %w", err)
	}

	var rows []row
	if err := txx.Select(&rows, query, args...); err != nil {
		return fmt.Errorf("selecting certificate assets for backfill: %w", err)
	}

	for _, r := range rows {
		decrypted, err := decrypt_AddRenewalToMDMConfigAssets(r.Value, privateKey)
		if err != nil {
			// If decryption fails (e.g. key mismatch), skip this row rather
			// than failing the entire migration.
			continue
		}

		renewAt := extractCertRenewAt_AddRenewalToMDMConfigAssets(decrypted)
		if renewAt == nil {
			continue
		}

		if _, err := tx.Exec(`UPDATE mdm_config_assets SET renew_at = ? WHERE id = ?`, renewAt, r.ID); err != nil {
			return fmt.Errorf("updating renew_at for mdm_config_assets id %d: %w", r.ID, err)
		}
	}

	return nil
}

func Down_20260414185606(tx *sql.Tx) error {
	return nil
}

// decrypt_AddRenewalToMDMConfigAssets is a migration-local copy of the AES-GCM decryption
// routine used by the datastore. It is duplicated here so the migration is
// self-contained and immune to future refactors of the main decrypt function.
func decrypt_AddRenewalToMDMConfigAssets(encrypted []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return decrypted, nil
}

// extractCertRenewAt_AddRenewalToMDMConfigAssets is a migration-local copy of
// fleet.ExtractCertRenewAt. It parses PEM-encoded certificate bytes and
// returns the expiration time (NotAfter) of the first certificate found.
func extractCertRenewAt_AddRenewalToMDMConfigAssets(pemBytes []byte) *time.Time {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	notAfter := cert.NotAfter.UTC()
	return &notAfter
}
