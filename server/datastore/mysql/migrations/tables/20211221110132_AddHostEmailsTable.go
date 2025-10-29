package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211221110132, Down_20211221110132)
}

func Up_20211221110132(tx *sql.Tx) error {
	hostEmailsTable := `
		CREATE TABLE IF NOT EXISTS host_emails (
			id         int(10) unsigned NOT NULL AUTO_INCREMENT,
			host_id    int(10) UNSIGNED NOT NULL,
			email      varchar(255) NOT NULL,
			source     varchar(255) NOT NULL,
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NOT NULL NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			PRIMARY KEY (id),
			INDEX idx_host_emails_host_id_email (host_id, email),
			INDEX idx_host_emails_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	if _, err := tx.Exec(hostEmailsTable); err != nil {
		return errors.Wrap(err, "create host_emails table")
	}
	return nil
}

func Down_20211221110132(tx *sql.Tx) error {
	return nil
}
