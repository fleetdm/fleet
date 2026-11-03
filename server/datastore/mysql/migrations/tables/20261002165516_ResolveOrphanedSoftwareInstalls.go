package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261002165516, Down_20261002165516)
}

// Installs whose installer was deleted while they were still pending never resolve. Fail them the way fleetd
// reports a missing installer, and drop their queue rows so the hosts' queues drain.
func Up_20261002165516(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		incrementalMigrationStep(countOrphanedSoftwareInstalls, failOrphanedSoftwareInstalls),
		incrementalMigrationStep(countOrphanedInstallQueueRows, deleteOrphanedInstallQueueRows),
	}, tx)
}

// The software_installer_id index holds the NULL entries in primary key order, so it drives both the count and
// the keyset scan without walking the whole table.
const orphanedSoftwareInstallsFrom = `
	FROM host_software_installs FORCE INDEX (fk_host_software_installs_installer_id)
	WHERE software_installer_id IS NULL
	AND status IN ('pending_install', 'pending_uninstall')
	AND host_deleted_at IS NULL`

const orphanedInstallQueueRowsFrom = `
	FROM software_install_upcoming_activities FORCE INDEX (fk_software_install_upcoming_activities_software_installer_id)
	WHERE software_installer_id IS NULL`

const orphanedBatchSize = 1000

func countOrphanedSoftwareInstalls(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`SELECT COUNT(*) ` + orphanedSoftwareInstallsFrom).Scan(&total)
	return total, err
}

func failOrphanedSoftwareInstalls(tx *sql.Tx, increment incrementCountFn) error {
	// fleet.ExitCodeInstallerNotFound, hardcoded so the migration doesn't change with the constant.
	const exitCode = -4
	const output = "Installer no longer exists on the server."

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var lastID uint
	for {
		var batch []struct {
			ID          uint   `db:"id"`
			Uninstall   bool   `db:"uninstall"`
			ExecutionID string `db:"execution_id"`
		}
		if err := txx.Select(&batch, `SELECT id, uninstall, execution_id `+orphanedSoftwareInstallsFrom+`
			AND id > ? ORDER BY id LIMIT ?`, lastID, orphanedBatchSize); err != nil {
			return fmt.Errorf("selecting orphaned software installs after id %d: %w", lastID, err)
		}
		if len(batch) == 0 {
			return nil
		}
		var installIDs, uninstallIDs []uint
		var uninstallExecIDs []string
		for _, r := range batch {
			if r.Uninstall {
				uninstallIDs = append(uninstallIDs, r.ID)
				uninstallExecIDs = append(uninstallExecIDs, r.ExecutionID)
			} else {
				installIDs = append(installIDs, r.ID)
			}
			increment()
		}
		lastID = batch[len(batch)-1].ID

		if len(installIDs) > 0 {
			if err := execIn(tx, `UPDATE host_software_installs SET install_script_exit_code = ?, install_script_output = ?
				WHERE id IN (?)`, exitCode, output, installIDs); err != nil {
				return fmt.Errorf("failing orphaned installs: %w", err)
			}
		}
		if len(uninstallIDs) > 0 {
			if err := execIn(tx, `UPDATE host_software_installs SET uninstall_script_exit_code = ?, uninstall_script_output = ?
				WHERE id IN (?)`, exitCode, output, uninstallIDs); err != nil {
				return fmt.Errorf("failing orphaned uninstalls: %w", err)
			}
			// Activating an uninstall writes its pending script result; drop it like deleting the installer would.
			if err := execIn(tx, `DELETE FROM host_script_results WHERE execution_id IN (?)`, uninstallExecIDs); err != nil {
				return fmt.Errorf("deleting orphaned uninstall script results: %w", err)
			}
		}
	}
}

func countOrphanedInstallQueueRows(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`SELECT COUNT(*) ` + orphanedInstallQueueRowsFrom).Scan(&total)
	return total, err
}

// Activated rows go too: no host result can ever arrive to remove them, and the unblock cron activates the next
// activity once they're gone.
func deleteOrphanedInstallQueueRows(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx}
	var lastID uint64
	for {
		var ids []uint64
		if err := txx.Select(&ids, `SELECT upcoming_activity_id `+orphanedInstallQueueRowsFrom+`
			AND upcoming_activity_id > ? ORDER BY upcoming_activity_id LIMIT ?`, lastID, orphanedBatchSize); err != nil {
			return fmt.Errorf("selecting orphaned install queue rows after id %d: %w", lastID, err)
		}
		if len(ids) == 0 {
			return nil
		}
		// The child row cascades.
		if err := execIn(tx, `DELETE FROM upcoming_activities WHERE id IN (?)`, ids); err != nil {
			return fmt.Errorf("deleting orphaned install queue rows: %w", err)
		}
		for range ids {
			increment()
		}
		lastID = ids[len(ids)-1]
	}
}

func execIn(tx *sql.Tx, stmt string, args ...any) error {
	query, expanded, err := sqlx.In(stmt, args...)
	if err != nil {
		return err
	}
	_, err = tx.Exec(query, expanded...)
	return err
}

func Down_20261002165516(tx *sql.Tx) error {
	return nil
}
