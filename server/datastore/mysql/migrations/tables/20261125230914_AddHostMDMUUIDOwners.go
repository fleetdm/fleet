package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261125230914, Down_20261125230914)
}

// Up_20261125230914 adds the record of which host owns a hardware UUID's MDM state, and gives every UUID linked to a Windows MDM
// enrollment an owner. Among Windows hosts sharing the UUID, it prefers the one MDM-enrolled, then recorded as enrolled in Fleet,
// then whose osquery host identifier is the UUID, then the newest. Without the Fleet preference, that is the host the Windows
// profile reconciler used before this table existed.
func Up_20261125230914(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_mdm_uuid_owners (
			host_uuid    varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			mdm_platform enum('apple','windows','android') COLLATE utf8mb4_unicode_ci NOT NULL,
			host_id      int unsigned NOT NULL,
			claimed_at   datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			claimed_by   varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
			PRIMARY KEY (host_uuid, mdm_platform),
			UNIQUE KEY idx_host_mdm_uuid_owners_host_id (host_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("creating host_mdm_uuid_owners: %w", err)
	}

	// Driven from the enrollments so it never scans hosts. UUIDs compare case-insensitively, as hosts.uuid does.
	res, err := tx.Exec(`
		INSERT INTO host_mdm_uuid_owners (host_uuid, mdm_platform, host_id, claimed_by)
		SELECT ranked.uuid, 'windows', ranked.id, 'migration'
		FROM (
			SELECT h.uuid, h.id, ROW_NUMBER() OVER (
				PARTITION BY h.uuid
				ORDER BY
					COALESCE(hm.enrolled, 0) DESC,
					COALESCE(mdms.name = 'Fleet', 0) DESC,
					COALESCE(h.osquery_host_id = h.uuid, 0) DESC,
					h.id DESC
			) AS rn
			FROM (SELECT DISTINCT host_uuid FROM mdm_windows_enrollments WHERE host_uuid <> '') e
			JOIN hosts h FORCE INDEX (idx_hosts_uuid) ON h.uuid = e.host_uuid AND h.platform = 'windows'
			LEFT JOIN host_mdm hm ON hm.host_id = h.id
			LEFT JOIN mobile_device_management_solutions mdms ON mdms.id = hm.mdm_id
		) ranked
		WHERE ranked.rn = 1
		ON DUPLICATE KEY UPDATE host_id = host_mdm_uuid_owners.host_id`)
	if err != nil {
		return fmt.Errorf("backfilling windows owners in host_mdm_uuid_owners: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n > 0 {
		logger.Info.Printf("Recorded the Windows MDM host of %d hardware UUIDs.\n", n)
	}

	type sharedUUID struct {
		hostUUID string
		ownerID  uint
		otherIDs string
	}
	var shared []sharedUUID
	rows, err := tx.Query(`
		SELECT o.host_uuid, o.host_id, GROUP_CONCAT(h.id ORDER BY h.id)
		FROM host_mdm_uuid_owners o
		JOIN hosts h FORCE INDEX (idx_hosts_uuid) ON h.uuid = o.host_uuid AND h.platform = 'windows' AND h.id <> o.host_id
		WHERE o.mdm_platform = 'windows'
		GROUP BY o.host_uuid, o.host_id`)
	if err != nil {
		return fmt.Errorf("listing windows hosts sharing an owned uuid: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s sharedUUID
		if err := rows.Scan(&s.hostUUID, &s.ownerID, &s.otherIDs); err != nil {
			return fmt.Errorf("scanning windows hosts sharing an owned uuid: %w", err)
		}
		shared = append(shared, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating windows hosts sharing an owned uuid: %w", err)
	}
	for _, s := range shared {
		logger.Warn.Printf("Windows hosts %s and %d share hardware UUID %s. Fleet now treats host %d as the device's MDM host; "+
			"delete the hosts that are stale.\n", s.otherIDs, s.ownerID, s.hostUUID, s.ownerID)
	}
	return nil
}

func Down_20261125230914(tx *sql.Tx) error {
	return nil
}
