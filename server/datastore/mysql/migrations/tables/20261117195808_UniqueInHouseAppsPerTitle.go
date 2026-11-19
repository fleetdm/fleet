package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261117195808, Down_20261117195808)
}

const hostInHouseInstallsBatchSize = 1000

type inHouseAppForDeduplication struct {
	ID             uint   `db:"id"`
	GlobalOrTeamID uint   `db:"global_or_team_id"`
	TitleID        uint   `db:"title_id"`
	Filename       string `db:"filename"`
	Version        string `db:"version"`
}

type inHouseAppFleetAndTitle struct {
	GlobalOrTeamID uint
	TitleID        uint
}

func Up_20261117195808(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Keep the first row of each title by filename, which is the row the UI and installs use
	// because their lookup reads through the (global_or_team_id, filename, platform) unique index.
	var allInHouseApps []inHouseAppForDeduplication
	err := txx.Select(&allInHouseApps, `
		SELECT id, global_or_team_id, title_id, filename, version
		FROM in_house_apps
		WHERE title_id IS NOT NULL
		ORDER BY global_or_team_id, title_id, filename, id`)
	if err != nil {
		return fmt.Errorf("select in-house apps: %w", err)
	}

	var fleetAndTitleOrder []inHouseAppFleetAndTitle
	inHouseAppsByFleetAndTitle := make(map[inHouseAppFleetAndTitle][]inHouseAppForDeduplication)
	for _, inHouseApp := range allInHouseApps {
		fleetAndTitle := inHouseAppFleetAndTitle{GlobalOrTeamID: inHouseApp.GlobalOrTeamID, TitleID: inHouseApp.TitleID}
		if _, ok := inHouseAppsByFleetAndTitle[fleetAndTitle]; !ok {
			fleetAndTitleOrder = append(fleetAndTitleOrder, fleetAndTitle)
		}
		inHouseAppsByFleetAndTitle[fleetAndTitle] = append(inHouseAppsByFleetAndTitle[fleetAndTitle], inHouseApp)
	}

	for _, fleetAndTitle := range fleetAndTitleOrder {
		inHouseApps := inHouseAppsByFleetAndTitle[fleetAndTitle]
		if len(inHouseApps) < 2 {
			continue
		}

		keptInHouseApp := inHouseApps[0]

		for _, inHouseApp := range inHouseApps {
			if inHouseApp.ID == keptInHouseApp.ID {
				continue
			}

			// Move installs to the kept row so they are not deleted with the duplicate.
			for {
				var result sql.Result
				var rowsAffected int64
				result, err = tx.Exec(
					`UPDATE host_in_house_software_installs SET in_house_app_id = ? WHERE in_house_app_id = ? LIMIT ?`,
					keptInHouseApp.ID, inHouseApp.ID, hostInHouseInstallsBatchSize,
				)
				if err != nil {
					return fmt.Errorf("move host in-house installs from in-house app %d: %w", inHouseApp.ID, err)
				}
				rowsAffected, err = result.RowsAffected()
				if err != nil {
					return fmt.Errorf("count moved host in-house installs from in-house app %d: %w", inHouseApp.ID, err)
				}
				if rowsAffected < hostInHouseInstallsBatchSize {
					break
				}
			}
			_, err = tx.Exec(
				`UPDATE in_house_app_upcoming_activities SET in_house_app_id = ? WHERE in_house_app_id = ?`,
				keptInHouseApp.ID, inHouseApp.ID,
			)
			if err != nil {
				return fmt.Errorf("move upcoming in-house installs from in-house app %d: %w", inHouseApp.ID, err)
			}
			_, err = tx.Exec(
				`UPDATE setup_experience_status_results SET in_house_app_id = ? WHERE in_house_app_id = ?`,
				keptInHouseApp.ID, inHouseApp.ID,
			)
			if err != nil {
				return fmt.Errorf("move setup experience results from in-house app %d: %w", inHouseApp.ID, err)
			}

			_, err = tx.Exec(`DELETE FROM in_house_apps WHERE id = ?`, inHouseApp.ID)
			if err != nil {
				return fmt.Errorf("delete duplicate in-house app %d: %w", inHouseApp.ID, err)
			}
			logger.Info.Printf(
				"Deleted duplicate in-house app title_id=%d filename=%q version=%q\n",
				inHouseApp.TitleID,
				inHouseApp.Filename,
				inHouseApp.Version,
			)
		}
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
