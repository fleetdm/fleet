package tables

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250502222222, Down_20250502222222)
}

func Up_20250502222222(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// legacy_host_mdm_enroll_refs captures existing enroll refs from the host_mdm table. Going
	// forward, it is used to ensure the enroll refs is appended to the server URL when legacy Apple
	// devices attempt to re-enroll. This table eventually could be dropped once all legacy devices
	// are retired.
	createStmt := `
CREATE TABLE IF NOT EXISTS legacy_host_mdm_enroll_refs (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	host_uuid VARCHAR(255) NOT NULL,
	enroll_ref VARCHAR(36) NOT NULL,
	INDEX idx_legacy_enroll_refs_host_uuid (host_uuid)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`

	// legacy_host_mdm_idp_accounts captures existing host email information from the host_emails table solely
	// for archival purposes in the event of unexpected issues with the new enrollment flow. This table
	// is not used for any ongoing purpose and could be dropped if no issues arise.
	createStmt += `
CREATE TABLE IF NOT EXISTS legacy_host_mdm_idp_accounts (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	host_uuid VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	account_uuid VARCHAR(36) NULL,
	host_id INT UNSIGNED NULL,
	email_id INT UNSIGNED NULL,
	email_created_at DATETIME NULL,
	email_updated_at DATETIME NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`

	// host_mdm_idp_accounts is used to track the association between hosts and MDM IDP accounts
	// during the enrollment process. Going forward, it is an integral part of the enrollment
	// process for Apple devices. Initially, it is populated with the host emails from the
	// legacy tables.
	createStmt += `
CREATE TABLE IF NOT EXISTS host_mdm_idp_accounts (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	host_uuid VARCHAR(255) NOT NULL,
	account_uuid VARCHAR(36) NOT NULL DEFAULT '',
	created_at DATETIME (6) NOT NULL DEFAULT NOW(6),
	updated_at DATETIME (6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
	UNIQUE KEY idx_host_mdm_idp_accounts (host_uuid)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`

	_, err := txx.Exec(createStmt)
	if err != nil {
		return err
	}

	// capture legacy enroll refs from host_mdm table
	_, err = txx.Exec(`
INSERT INTO legacy_host_mdm_enroll_refs (host_uuid, enroll_ref)
SELECT
	h.uuid,
	hmdm.fleet_enroll_ref
FROM
	host_mdm hmdm
	JOIN hosts h ON hmdm.host_id = h.id
WHERE
	hmdm.fleet_enroll_ref != ''`)
	if err != nil {
		return fmt.Errorf("inserting legacy enroll refs %w", err)
	}

	// capture legacy mdm idp accounts from host_emails table
	_, err = txx.Exec(`
INSERT INTO legacy_host_mdm_idp_accounts (host_uuid, email, email_id, host_id, email_created_at, email_updated_at, account_uuid)
SELECT 
	h.uuid AS host_uuid, 
	he.email,
	he.id AS email_id,
	he.host_id,
	he.created_at AS email_created_at,
	he.updated_at AS email_updated_at,
	mia.uuid AS account_uuid
FROM 
	host_emails he
	JOIN hosts h ON he.host_id = h.id
	LEFT JOIN mdm_idp_accounts mia ON mia.email = he.email
WHERE 
	he.source = ?`, fleet.DeviceMappingMDMIdpAccounts)
	if err != nil {
		return fmt.Errorf("inserting legacy host mdm idp accounts: %w", err)
	}

	// Lack of uniqe contraints for emails in the prior implementation means we may find hosts with missing,
	// duplicate, or conflicting account information. In this migration, we'll resolve to a
	// single account UUID using some best-guess heuristics, but we'll also log any conflicts for
	// investigation and hang on to the legacy tables for archival purposes in case we discover
	// any issues.
	type legacyAccount struct {
		ID             uint      `db:"id"`
		HostUUID       string    `db:"host_uuid"`
		HostID         uint      `db:"host_id"`
		Email          string    `db:"email"`
		EmailID        uint      `db:"email_id"`
		EmailCreatedAt time.Time `db:"email_created_at"`
		EmailUpdatedAt time.Time `db:"email_updated_at"`
		AccountUUID    string    `db:"account_uuid"`
	}
	var hostEmails []legacyAccount
	err = txx.Select(&hostEmails, `
SELECT 
	id,
	host_uuid,
	host_id,
	email,
	email_id,
	email_created_at,
	email_updated_at,
	coalesce(account_uuid, '') AS account_uuid
FROM 
	legacy_host_mdm_idp_accounts
ORDER BY email_created_at DESC, email DESC`) // order by is arbitrary but deterministic; this order means we'll prefer the most recent and alphanumerically largest emails in case of duplicates
	if err != nil {
		return fmt.Errorf("selecting existing host emails: %w", err)
	}

	emailByHostUUID := make(map[string]legacyAccount, len(hostEmails))
	ignored := []legacyAccount{}
	for _, he := range hostEmails {
		if he.AccountUUID == "" {
			ignored = append(ignored, he)
			continue
		}
		if v, ok := emailByHostUUID[he.HostUUID]; ok && (v.Email != he.Email || v.AccountUUID != he.AccountUUID) {
			ignored = append(ignored, he)
			continue
		}
		emailByHostUUID[he.HostUUID] = he
	}

	// If we didn't get a match with email-based join above, we try to match
	// the legacy enroll refs to the mdm_idp_accounts table as a fallback.
	type accountMatch struct {
		HostUUID    string `db:"host_uuid"`
		AccountUUID string `db:"account_uuid"`
		Email       string `db:"email"`
	}
	stmt := `
SELECT
	host_uuid,
	mia.uuid as account_uuid,
	mia.email
FROM
	legacy_host_mdm_enroll_refs hler
	JOIN mdm_idp_accounts mia ON mia.uuid = hler.enroll_ref`
	var matchedRefs []accountMatch
	if err := txx.Select(&matchedRefs, stmt); err != nil {
		return fmt.Errorf("matching mdm idp accounts to legacy refs: %w", err)
	}

	refMatchByHostUUID := make(map[string]accountMatch, len(matchedRefs))
	refConflicts := []interface{}{}
	for _, match := range matchedRefs {
		if m, ok := refMatchByHostUUID[match.HostUUID]; ok {
			if m.Email != match.Email || m.AccountUUID != match.AccountUUID {
				refConflicts = append(refConflicts, m, match)
				continue
			}
		}
		if e, ok := emailByHostUUID[match.HostUUID]; ok {
			if e.Email != match.Email || e.AccountUUID != match.AccountUUID {
				refConflicts = append(refConflicts, e, match)
				continue
			}
		}
		refMatchByHostUUID[match.HostUUID] = match
	}

	// Log any conflicts we found. We don't want to fail the migration, but
	// we do want to surface potential issues for investigation.
	msg := ""
	if len(ignored) > 0 {
		msg += fmt.Sprintf("ignoring %d host email records because no matching account or conflicting acount information\n", len(ignored))
		for _, he := range ignored {
			msg += fmt.Sprintf("  - %+v\n", he)
		}
	}
	if len(refConflicts) > 0 {
		msg += fmt.Sprintf("found %d legacy enroll references with duplicative or conflicting account information\n", len(refConflicts))
		for _, m := range refConflicts {
			msg += fmt.Sprintf("  - %+v\n", m)
		}
	}
	if msg != "" {
		// // TODO: return or log error?
		// return errors.New(msg)
		fmt.Println(msg)
	}

	// Finally, we populate the new host_mdm_idp_accounts table with the deduped results.
	items := make([]accountMatch, 0, len(emailByHostUUID)+len(refMatchByHostUUID))
	for _, he := range emailByHostUUID {
		items = append(items, accountMatch{
			HostUUID:    he.HostUUID,
			AccountUUID: he.AccountUUID,
			Email:       he.Email,
		})
	}
	for _, m := range refMatchByHostUUID {
		items = append(items, accountMatch{
			HostUUID:    m.HostUUID,
			AccountUUID: m.AccountUUID,
			Email:       m.Email,
		})
	}
	executeUpsertBatch := func(itemsThisBatch []accountMatch) error {
		valuesPart := ""
		args := make([]interface{}, 0, len(itemsThisBatch)*2)
		for _, item := range itemsThisBatch {
			valuesPart += "(?, ?),"
			args = append(args, item.HostUUID, item.AccountUUID)
		}

		insStmt := fmt.Sprintf(`
INSERT INTO host_mdm_idp_accounts (host_uuid, account_uuid)
VALUES %s
ON DUPLICATE KEY UPDATE
	updated_at = NOW(6)`, strings.TrimSuffix(valuesPart, ","))

		if _, err := txx.Exec(insStmt, args...); err != nil {
			return fmt.Errorf("upserting host mdm idp accounts: %w", err)
		}
		return nil
	}
	if err := common_mysql.BatchProcessSimple[accountMatch](items, 10000, executeUpsertBatch); err != nil {
		return fmt.Errorf("batch processing host mdm idp accounts: %w", err)
	}

	// TODO: anything else we want to do here?

	return nil
}

func Down_20250502222222(tx *sql.Tx) error {
	return nil
}
