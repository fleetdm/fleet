package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var backfillAppleBuiltinLabelMembershipsBatchSize = 1000

func init() {
	MigrationClient.AddMigration(Up_20260723181412, Down_20260723181412)
}

func Up_20260723181412(tx *sql.Tx) error {
	step := incrementalMigrationStep(countHostsMissingAppleBuiltinLabelMemberships, backfillHostsMissingAppleBuiltinLabelMemberships)
	if err := step(tx); err != nil {
		return fmt.Errorf("backfilling Apple built-in label memberships: %w", err)
	}
	return nil
}

func countHostsMissingAppleBuiltinLabelMemberships(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`
		SELECT COUNT(DISTINCT h.id)
		FROM hosts h
		JOIN labels l ON l.label_type = 1 AND (
			l.name = 'All Hosts' OR
			l.name = CASE h.platform
				WHEN 'darwin' THEN 'macOS'
				WHEN '' THEN 'macOS'
				WHEN 'ios' THEN 'iOS'
				WHEN 'ipados' THEN 'iPadOS'
			END
		)
		LEFT JOIN label_membership lm ON lm.host_id = h.id AND lm.label_id = l.id
		WHERE (
			h.platform IN ('darwin', 'ios', 'ipados') OR
			(h.platform = '' AND EXISTS (
				SELECT 1 FROM host_mdm hmdm
				JOIN mobile_device_management_solutions mdm ON mdm.id = hmdm.mdm_id
				WHERE hmdm.host_id = h.id AND mdm.name = 'Fleet'
			))
		) AND lm.host_id IS NULL
	`).Scan(&total)
	return total, err
}

func backfillHostsMissingAppleBuiltinLabelMemberships(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var lastHostID uint
	for {
		var hostIDs []uint
		if err := txx.Select(&hostIDs, `
			SELECT DISTINCT h.id
			FROM hosts h
			JOIN labels l ON l.label_type = 1 AND (
				l.name = 'All Hosts' OR
				l.name = CASE h.platform
					WHEN 'darwin' THEN 'macOS'
					WHEN '' THEN 'macOS'
					WHEN 'ios' THEN 'iOS'
					WHEN 'ipados' THEN 'iPadOS'
				END
			)
			LEFT JOIN label_membership lm ON lm.host_id = h.id AND lm.label_id = l.id
			WHERE (
				h.platform IN ('darwin', 'ios', 'ipados') OR
				(h.platform = '' AND EXISTS (
					SELECT 1 FROM host_mdm hmdm
					JOIN mobile_device_management_solutions mdm ON mdm.id = hmdm.mdm_id
					WHERE hmdm.host_id = h.id AND mdm.name = 'Fleet'
				))
			)
				AND h.id > ? AND lm.host_id IS NULL
			ORDER BY h.id
			LIMIT ?`, lastHostID, backfillAppleBuiltinLabelMembershipsBatchSize); err != nil {
			return fmt.Errorf("selecting hosts missing Apple built-in label memberships after host ID %d: %w", lastHostID, err)
		}
		if len(hostIDs) == 0 {
			return nil
		}

		query, args, err := sqlx.In(`
			INSERT IGNORE INTO label_membership (host_id, label_id)
			SELECT h.id, l.id
			FROM hosts h
			JOIN labels l ON l.label_type = 1 AND (
				l.name = 'All Hosts' OR
				l.name = CASE h.platform
					WHEN 'darwin' THEN 'macOS'
					WHEN '' THEN 'macOS'
					WHEN 'ios' THEN 'iOS'
					WHEN 'ipados' THEN 'iPadOS'
				END
			)
			LEFT JOIN label_membership lm ON lm.host_id = h.id AND lm.label_id = l.id
			WHERE h.id IN (?) AND lm.host_id IS NULL`, hostIDs)
		if err != nil {
			return fmt.Errorf("building Apple built-in label membership backfill after host ID %d: %w", lastHostID, err)
		}
		if _, err := txx.Exec(query, args...); err != nil {
			return fmt.Errorf("backfilling Apple built-in label memberships after host ID %d: %w", lastHostID, err)
		}

		for range hostIDs {
			increment()
		}
		lastHostID = hostIDs[len(hostIDs)-1]
	}
}

func Down_20260723181412(tx *sql.Tx) error {
	return nil
}
