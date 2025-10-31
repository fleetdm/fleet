package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251016100000, Down_20251016100000)
}

func Up_20251016100000(tx *sql.Tx) error {
	// Find hosts with mdm_idp_accounts emails but no corresponding host_scim_user records
	const findUnmappedHostsQuery = `
		SELECT he.host_id, he.email
		FROM host_emails he
		LEFT JOIN host_scim_user hsu ON he.host_id = hsu.host_id
		WHERE he.source = 'mdm_idp_accounts'
		  AND hsu.host_id IS NULL
	`

	rows, err := tx.Query(findUnmappedHostsQuery)
	if err != nil {
		return errors.Wrap(err, "find hosts with mdm_idp_accounts emails but no SCIM user mapping")
	}
	defer rows.Close()

	var unmappedHosts []struct {
		HostID uint
		Email  string
	}

	for rows.Next() {
		var host struct {
			HostID uint
			Email  string
		}
		if err := rows.Scan(&host.HostID, &host.Email); err != nil {
			return errors.Wrap(err, "scan unmapped host")
		}
		unmappedHosts = append(unmappedHosts, host)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterate unmapped hosts")
	}

	if len(unmappedHosts) == 0 {
		return nil // Nothing to reconcile
	}

	var reconciled int
	for _, host := range unmappedHosts {
		// Try to find a SCIM user by email (treating email as user_name)
		const findScimUserQuery = `
			SELECT id FROM scim_users
			WHERE user_name = ?
			LIMIT 1
		`

		var scimUserID uint
		err := tx.QueryRow(findScimUserQuery, host.Email).Scan(&scimUserID)
		if err != nil {
			if err == sql.ErrNoRows {
				// No matching SCIM user found, skip
				continue
			}
			return errors.Wrapf(err, "find SCIM user for host %d email %s", host.HostID, host.Email)
		}

		// Create the host-to-SCIM user mapping
		const insertMappingQuery = `
			INSERT IGNORE INTO host_scim_user (host_id, scim_user_id)
			VALUES (?, ?)
		`

		_, err = tx.Exec(insertMappingQuery, host.HostID, scimUserID)
		if err != nil {
			return errors.Wrapf(err, "create host-SCIM user mapping for host %d scim_user %d", host.HostID, scimUserID)
		}

		reconciled++
	}

	return nil
}

func Down_20251016100000(tx *sql.Tx) error {
	// No down migration needed for data reconciliation
	return nil
}
