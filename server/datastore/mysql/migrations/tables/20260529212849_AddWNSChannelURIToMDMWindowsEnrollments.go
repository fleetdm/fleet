package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260529212849, Down_20260529212849)
}

func Up_20260529212849(tx *sql.Tx) error {
	// WNS push notification channel for the device. The device reports its ChannelURI (and a status code)
	// in response to a Get on the DMClient Push CSP nodes during a management session. The server uses the
	// ChannelURI to wake the device with a raw WNS push instead of waiting for the next poll. These are
	// null until the device reports a channel, and remain null when WNS push is not configured.
	_, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
			ADD COLUMN wns_channel_uri VARCHAR(2048) NULL DEFAULT NULL,
			ADD COLUMN wns_channel_uri_status INT NULL DEFAULT NULL,
			ADD COLUMN wns_channel_uri_updated_at DATETIME(6) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding WNS channel URI columns to mdm_windows_enrollments: %w", err)
	}
	return nil
}

func Down_20260529212849(tx *sql.Tx) error {
	return nil
}
