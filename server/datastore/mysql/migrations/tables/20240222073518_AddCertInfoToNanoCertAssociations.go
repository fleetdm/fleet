package tables

import (
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20240222073518, Down_20240222073518)
}

func Up_20240222073518(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE nano_cert_auth_associations
-- used to detect identity certificates that are about to expire. While we have
-- access to the scep_certificates table, nanomdm assumes that you can use any CA
-- to issue your identity certificates, as such we can't add a foreign key here
-- without major changes.
ADD COLUMN cert_not_valid_after TIMESTAMP NULL,
-- used to track the command issued to renew the identity certificate (if one)
ADD COLUMN renew_command_uuid VARCHAR(127) COLLATE utf8mb4_unicode_ci NULL,
ADD CONSTRAINT renew_command_uuid_fk 
    FOREIGN KEY (renew_command_uuid) REFERENCES nano_commands (command_uuid)
`)
	if err != nil {
		return fmt.Errorf("failed to alter nano_cert_auth_associations table: %w", err)
	}

	if err := batchUpdateCertAssociationsTimestamps(tx); err != nil {
		return fmt.Errorf("failed to update associations timestamps: %w", err)
	}

	return nil
}

func batchUpdateCertAssociationsTimestamps(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var totalCount int
	if err := txx.Get(&totalCount, "SELECT COUNT(*) FROM scep_certificates"); err != nil {
		return fmt.Errorf("failed to get total count of scep_certificates: %w", err)
	}

	const batchSize = 100
	for offset := 0; offset < totalCount; offset += batchSize {
		if err := updateCertAssociationTimestamps(&txx, batchSize, offset); err != nil {
			return fmt.Errorf("updating batch with offset %d: %w", offset, err)
		}
	}

	return nil
}

func updateCertAssociationTimestamps(txx *sqlx.Tx, limit, offset int) error {
	var scepCerts []struct {
		Serial         string `db:"serial"`
		CertificatePEM []byte `db:"certificate_pem"`
	}

	if err := txx.Select(
		&scepCerts, `
		SELECT certificate_pem 
		FROM scep_certificates
		ORDER BY serial
		LIMIT ? OFFSET ?
		`,
		limit, offset,
	); err != nil {
		return fmt.Errorf("failed to retrieve scep_certificates: %w", err)
	}

	shas := make([]string, len(scepCerts))
	expiries := make(map[string]time.Time, len(scepCerts))
	for i, rawCert := range scepCerts {
		block, _ := pem.Decode(rawCert.CertificatePEM)
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Printf("failed to parse certificate with serial %s", rawCert.Serial)
			continue
		}

		hashed := hashCert(cert)
		shas[i] = hashed
		expiries[hashed] = cert.NotAfter
	}

	var assocs []struct {
		HostUUID  string    `db:"id"`
		SHA256    string    `db:"sha256"`
		CreatedAt time.Time `db:"created_at"`
	}
	selectAssocStmt, selectAssocArgs, err := sqlx.In(`
		SELECT id, sha256
		FROM nano_cert_auth_associations
		WHERE sha256 IN (?)
		`, shas)
	if err != nil {
		return fmt.Errorf("building sqlx.In for cert associations: %w", err)
	}

	if err := txx.Select(&assocs, selectAssocStmt, selectAssocArgs...); err != nil {
		return fmt.Errorf("failed to retrieve cert associations: %w", err)
	}

	var sb strings.Builder
	updateAssocArgs := make([]any, len(assocs)*3)
	for i, assoc := range assocs {
		sb.WriteString("(?, ?, ?),")
		updateAssocArgs[i*3] = assoc.HostUUID
		updateAssocArgs[i*3+1] = assoc.SHA256
		updateAssocArgs[i*3+2] = expiries[assoc.SHA256]
	}

	updateAssocStmt := fmt.Sprintf(`
		INSERT INTO nano_cert_auth_associations (id, sha256, cert_not_valid_after) VALUES %s
		ON DUPLICATE KEY UPDATE
			cert_not_valid_after = VALUES(cert_not_valid_after),
			updated_at = updated_at
	`, strings.TrimSuffix(sb.String(), ","))
	if _, err := txx.Exec(updateAssocStmt, updateAssocArgs...); err != nil {
		return fmt.Errorf("failed to update cert associations: %w", err)
	}

	return nil
}

func hashCert(cert *x509.Certificate) string {
	hashed := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hashed[:])
}

func Down_20240222073518(tx *sql.Tx) error {
	return nil
}
