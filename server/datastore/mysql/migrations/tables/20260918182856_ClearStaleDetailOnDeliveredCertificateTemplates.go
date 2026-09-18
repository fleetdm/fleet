package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260918182856, Down_20260918182856)
}

// An automatic retry of a certificate install carries the failure message that caused it through
// pending and delivering so the UI can explain why the certificate is being sent again. The
// transition to delivered did not empty that message, so a certificate that had been delivered
// again still reported its old failure. The write path now empties it, but a row only gets
// rewritten when the host reports on it, which a host that has gone quiet never does.
//
// Scoped to delivered rows: on a pending, delivering or failed row the detail is the current error
// message and has to survive, and on a verified row it was written by the host's own report.
//
// A NULL detail is left alone. NULL is what tells a manual resend apart from an automatic retry
// (see fleet.HostCertificateTemplate.IsRetrying), so it must not be turned into an empty string.
func Up_20260918182856(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		UPDATE host_certificate_templates
		SET detail = '', updated_at = updated_at
		WHERE status = 'delivered' AND detail IS NOT NULL AND detail != ''
	`); err != nil {
		return fmt.Errorf("clearing stale detail on delivered certificate templates: %w", err)
	}

	return nil
}

func Down_20260918182856(tx *sql.Tx) error {
	return nil
}
