package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20261013045312, Down_20261013045312)
}

// Account-driven user enrollments were stored with full access rights (8191)
// because the personal check only looked for byod=1 in the ServerURL. Apple
// never grants lock (4) or erase (8) on a user enrollment, so clear those bits.
func Up_20261013045312(tx *sql.Tx) error {
	// Bounding every index column keeps this a range read of the
	// (small) account-driven slice instead of a scan of host_mdm.
	rows, err := tx.Query(`
		SELECT h.uuid
		FROM host_mdm hm FORCE INDEX (host_mdm_enrolled_dep_personal_type_idx)
		JOIN hosts h ON h.id = hm.host_id
		WHERE hm.enrolled IN (0, 1)
			AND hm.installed_from_dep IN (0, 1)
			AND hm.is_personal_enrollment = 1
			AND hm.personal_enrollment_type = 'account_driven'
	`)
	if err != nil {
		return fmt.Errorf("select account-driven hosts: %w", err)
	}
	defer rows.Close()
	var uuids []any
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return fmt.Errorf("scan account-driven host: %w", err)
		}
		uuids = append(uuids, uuid)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate account-driven hosts: %w", err)
	}

	const batchSize = 1000
	for start := 0; start < len(uuids); start += batchSize {
		stmt, args, err := sqlx.In(`
			UPDATE host_mdm_apple_enrollment_permissions
			SET access_rights = access_rights & ~12, delivered_at = delivered_at
			WHERE host_uuid IN (?) AND access_rights & 12 != 0`, uuids[start:min(start+batchSize, len(uuids))])
		if err != nil {
			return fmt.Errorf("build access rights update: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("narrow account-driven enrollment access rights: %w", err)
		}
	}
	return nil
}

func Down_20261013045312(tx *sql.Tx) error {
	return nil
}
