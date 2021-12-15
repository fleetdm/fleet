package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211215110132, Down_20211215110132)
}

func Up_20211215110132(tx *sql.Tx) error {
	hostEmailsTable := `
		CREATE TABLE IF NOT EXISTS host_emails (
			id         int(10) unsigned NOT NULL AUTO_INCREMENT,
			host_id    int(10) UNSIGNED NOT NULL,
			email      varchar(255) NOT NULL,
			source     varchar(255) NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			PRIMARY KEY (id),
			INDEX idx_host_emails_host_id_email (host_id, email),
			INDEX idx_host_emails_email (email)
		);
	`
	// TODO(mna): do we want the created/updated timestamps? I usually add them
	// to all tables, but I noticed we don't have them everywhere. Also, I
	// suspect we don't want the FK on hosts, so I haven't added it and will add
	// a cleanup job.
	if _, err := tx.Exec(hostEmailsTable); err != nil {
		return errors.Wrap(err, "create host_emails table")
	}
	return nil
}

func Down_20211215110132(tx *sql.Tx) error {
	return nil
}
