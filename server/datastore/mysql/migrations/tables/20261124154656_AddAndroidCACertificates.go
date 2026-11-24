package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261124154656, Down_20261124154656)
}

func Up_20261124154656(tx *sql.Tx) error {
	createCAsTable := `
CREATE TABLE IF NOT EXISTS android_ca_certificates (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  team_id INT UNSIGNED NOT NULL DEFAULT 0,
  name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  certificate_pem TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  sha256_fingerprint CHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  subject TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  issuer TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  not_valid_before TIMESTAMP NULL DEFAULT NULL,
  not_valid_after TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_android_ca_certificates_team_name (team_id, name),
  KEY idx_android_ca_certificates_team_id (team_id),
  KEY idx_android_ca_certificates_sha256 (sha256_fingerprint)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createCAsTable); err != nil {
		return fmt.Errorf("create android_ca_certificates table: %w", err)
	}

	createLabelsTable := `
CREATE TABLE IF NOT EXISTS android_ca_certificate_labels (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  ca_certificate_id INT UNSIGNED NOT NULL,
  label_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  label_id INT UNSIGNED DEFAULT NULL,
  exclude TINYINT(1) NOT NULL DEFAULT 0,
  require_all TINYINT(1) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_android_ca_cert_labels_cert_label (ca_certificate_id, label_name),
  KEY idx_android_ca_cert_labels_label_id (label_id),
  CONSTRAINT fk_android_ca_cert_labels_cert_id FOREIGN KEY (ca_certificate_id) REFERENCES android_ca_certificates (id) ON DELETE CASCADE,
  CONSTRAINT fk_android_ca_cert_labels_label_id FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createLabelsTable); err != nil {
		return fmt.Errorf("create android_ca_certificate_labels table: %w", err)
	}

	createHostCAsTable := `
CREATE TABLE IF NOT EXISTS host_android_ca_certificates (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  host_id INT UNSIGNED NOT NULL,
  host_uuid VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  ca_certificate_id INT UNSIGNED NOT NULL,
  status VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  detail TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_host_android_ca_certs_host_cert (host_id, ca_certificate_id),
  KEY idx_host_android_ca_certs_host_uuid (host_uuid),
  KEY idx_host_android_ca_certs_ca_cert_id (ca_certificate_id),
  CONSTRAINT fk_host_android_ca_certs_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE,
  CONSTRAINT fk_host_android_ca_certs_ca_cert_id FOREIGN KEY (ca_certificate_id) REFERENCES android_ca_certificates (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createHostCAsTable); err != nil {
		return fmt.Errorf("create host_android_ca_certificates table: %w", err)
	}

	return nil
}

func Down_20261124154656(tx *sql.Tx) error {
	return nil
}
