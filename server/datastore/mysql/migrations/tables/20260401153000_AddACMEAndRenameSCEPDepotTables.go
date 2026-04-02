package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260401153000, Down_20260401153000)
}

func Up_20260401153000(tx *sql.Tx) error {
	_, err := tx.Exec(`
        ALTER TABLE nano_enrollments
        ADD COLUMN hardware_attested TINYINT(1) NOT NULL DEFAULT 0
    `)
	if err != nil {
		return errors.Wrap(err, "add nano_enrollments hardware_attested column")
	}

	_, err = tx.Exec(`
        ALTER TABLE scep_serials RENAME TO identity_serials
    `)
	if err != nil {
		return errors.Wrap(err, "rename scep_serials to identity_serials")
	}

	_, err = tx.Exec(`
        ALTER TABLE scep_certificates RENAME TO identity_certificates
    `)
	if err != nil {
		return errors.Wrap(err, "rename scep_certificates to identity_certificates")
	}

	_, err = tx.Exec(`
        CREATE TABLE acme_enrollments (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  path_identifier VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  host_identifier VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  not_valid_after DATETIME DEFAULT NULL,
  revoked TINYINT(1) NOT NULL DEFAULT '0',
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_path_identifier (path_identifier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    `)
	if err != nil {
		return errors.Wrap(err, "create acme_enrollments table")
	}

	_, err = tx.Exec(`
        CREATE TABLE acme_accounts (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  acme_enrollment_id INT UNSIGNED NOT NULL,
  json_web_key json NOT NULL,
  json_web_key_thumbprint VARCHAR(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  revoked TINYINT(1) NOT NULL DEFAULT '0',
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (acme_enrollment_id) REFERENCES acme_enrollments(id) ON DELETE CASCADE ON UPDATE CASCADE,
  UNIQUE KEY idx_enrollment_id_thumbprint (acme_enrollment_id, json_web_key_thumbprint)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    `)
	if err != nil {
		return errors.Wrap(err, "create acme_accounts table")
	}

	_, err = tx.Exec(`
        CREATE TABLE acme_orders (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  acme_account_id INT UNSIGNED NOT NULL,
  finalized TINYINT(1) NOT NULL DEFAULT '0',
  certificate_signing_request TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  identifiers json NOT NULL,
  status enum('pending', 'ready', 'processing', 'valid', 'invalid') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  issued_certificate_serial BIGINT DEFAULT NULL,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (acme_account_id) REFERENCES acme_accounts(id) ON DELETE CASCADE ON UPDATE CASCADE,
  UNIQUE KEY idx_issued_certificate_serial (issued_certificate_serial)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    `)
	if err != nil {
		return errors.Wrap(err, "create acme_orders table")
	}

	_, err = tx.Exec(`
        CREATE TABLE acme_authorizations (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    identifier_type varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    identifier_value varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    acme_order_id INT UNSIGNED NOT NULL,
    status enum('pending', 'valid', 'invalid', 'deactivated', 'expired', 'revoked') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
	created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	FOREIGN KEY (acme_order_id) REFERENCES acme_orders(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    `)
	if err != nil {
		return errors.Wrap(err, "create acme_authorizations table")
	}

	_, err = tx.Exec(`
        CREATE TABLE acme_challenges (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    challenge_type varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    token varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    acme_authorization_id INT UNSIGNED NOT NULL,
    status enum('pending', 'valid', 'invalid', 'processing') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
	created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	FOREIGN KEY (acme_authorization_id) REFERENCES acme_authorizations(id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    `)
	if err != nil {
		return errors.Wrap(err, "create acme_challenges table")
	}
	return nil
}

func Down_20260401153000(tx *sql.Tx) error {
	return nil
}
