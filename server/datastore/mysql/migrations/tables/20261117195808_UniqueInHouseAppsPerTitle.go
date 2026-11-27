package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration("Allow only one in-house app per title in each fleet", Up_20261117195808, Down_20261117195808)
}

const hostInHouseInstallsBatchSize = 1000

func Up_20261117195808(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Keep the first added row of each title. The UI and installs look up the row without an ORDER BY,
	// so which duplicate they used depends on the index MySQL picked.
	var duplicateInHouseApps []struct {
		ID               uint   `db:"id"`
		GlobalOrTeamID   uint   `db:"global_or_team_id"`
		TitleID          uint   `db:"title_id"`
		Filename         string `db:"filename"`
		Version          string `db:"version"`
		StorageID        string `db:"storage_id"`
		KeepInHouseAppID uint   `db:"keep_in_house_app_id"`
		KeepStorageID    string `db:"keep_storage_id"`
	}
	err := txx.Select(&duplicateInHouseApps, `
		SELECT id, global_or_team_id, title_id, filename, version, storage_id, keep_in_house_app_id, keep_storage_id
		FROM (
			SELECT
				id,
				global_or_team_id,
				title_id,
				filename,
				version,
				storage_id,
				FIRST_VALUE(id) OVER (PARTITION BY global_or_team_id, title_id ORDER BY id) AS keep_in_house_app_id,
				FIRST_VALUE(storage_id) OVER (PARTITION BY global_or_team_id, title_id ORDER BY id) AS keep_storage_id
			FROM in_house_apps
			WHERE title_id IS NOT NULL
		) in_house_apps_with_keep_id
		WHERE id != keep_in_house_app_id`)
	if err != nil {
		return fmt.Errorf("select duplicate in-house apps: %w", err)
	}

	for _, duplicate := range duplicateInHouseApps {
		// Move installs to the row to keep so they are not deleted with the duplicate.
		moveInstallsStatement := `UPDATE host_in_house_software_installs SET in_house_app_id = ? WHERE in_house_app_id = ? LIMIT ?`
		if duplicate.StorageID != duplicate.KeepStorageID {
			// Mark installs of a different file removed, the same as when an in-house app's file is edited.
			moveInstallsStatement = `UPDATE host_in_house_software_installs SET in_house_app_id = ?, removed = TRUE WHERE in_house_app_id = ? LIMIT ?`
		}
		for {
			var result sql.Result
			var rowsAffected int64
			result, err = tx.Exec(moveInstallsStatement, duplicate.KeepInHouseAppID, duplicate.ID, hostInHouseInstallsBatchSize)
			if err != nil {
				return fmt.Errorf("move host in-house installs from in-house app %d: %w", duplicate.ID, err)
			}
			rowsAffected, err = result.RowsAffected()
			if err != nil {
				return fmt.Errorf("count moved host in-house installs from in-house app %d: %w", duplicate.ID, err)
			}
			if rowsAffected < hostInHouseInstallsBatchSize {
				break
			}
		}
		_, err = tx.Exec(
			`UPDATE in_house_app_upcoming_activities SET in_house_app_id = ? WHERE in_house_app_id = ?`,
			duplicate.KeepInHouseAppID, duplicate.ID,
		)
		if err != nil {
			return fmt.Errorf("move upcoming in-house installs from in-house app %d: %w", duplicate.ID, err)
		}
		_, err = tx.Exec(
			`UPDATE setup_experience_status_results SET in_house_app_id = ? WHERE in_house_app_id = ?`,
			duplicate.KeepInHouseAppID, duplicate.ID,
		)
		if err != nil {
			return fmt.Errorf("move setup experience results from in-house app %d: %w", duplicate.ID, err)
		}

		_, err = tx.Exec(`DELETE FROM in_house_apps WHERE id = ?`, duplicate.ID)
		if err != nil {
			return fmt.Errorf("delete duplicate in-house app %d: %w", duplicate.ID, err)
		}
		logger.Info.Printf(
			"Deleted duplicate in-house app from fleet %d title_id=%d filename=%q version=%q\n",
			duplicate.GlobalOrTeamID,
			duplicate.TitleID,
			duplicate.Filename,
			duplicate.Version,
		)
	}

	if indexExistsTx(tx, "in_house_apps", "global_or_team_id") {
		_, err = tx.Exec(`ALTER TABLE in_house_apps DROP INDEX global_or_team_id`)
		if err != nil {
			return fmt.Errorf("drop in_house_apps filename unique index: %w", err)
		}
	}

	if !indexExistsTx(tx, "in_house_apps", "idx_in_house_apps_global_or_team_id_title_id") {
		_, err = tx.Exec(`ALTER TABLE in_house_apps ADD UNIQUE INDEX idx_in_house_apps_global_or_team_id_title_id (global_or_team_id, title_id)`)
		if err != nil {
			return fmt.Errorf("add in_house_apps title unique index: %w", err)
		}
	}

	return nil
}

func Down_20261117195808(tx *sql.Tx) error {
	return nil
}
