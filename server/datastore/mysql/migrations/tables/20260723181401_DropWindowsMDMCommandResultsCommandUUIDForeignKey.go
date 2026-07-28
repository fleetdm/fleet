package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20260723181401, Down_20260723181401)
}

func Up_20260723181401(tx *sql.Tx) error {
	// windows_mdm_command_results is written on every Windows MDM check-in (MDMWindowsSaveResponse). During an active
	// profile installation, tens of thousands of hosts insert result rows that all reference the same small set of shared
	// command rows in windows_mdm_commands (one command per non-variable profile, fanned out to every host). The
	// command_uuid foreign key forces InnoDB to take a shared lock on those few shared parent rows on every insert and hold
	// it for the whole transaction, which piles up shared-lock structures on a handful of rows and couples the hot insert
	// path to any exclusive lock on windows_mdm_commands.
	//
	// We drop ONLY the command_uuid foreign key: it is the one that references rows shared across all hosts, so it is the
	// only one that piles up. Its ON DELETE CASCADE never actually fires right now (nothing deletes windows_mdm_commands), so
	// removing it changes no cleanup behavior and creates no orphaned rows. The insert path already verifies the command
	// exists (MDMWindowsSaveResponse SELECTs matching commands before inserting), so the constraint is redundant there.
	referencedTables := map[string]struct{}{"windows_mdm_commands": {}}
	table := "windows_mdm_command_results"

	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return err
	}
	if len(constraints) == 0 {
		// Already dropped (e.g. re-run); nothing to do.
		return nil
	}

	// Only 1 constraint will be dropped here:
	// CONSTRAINT `windows_mdm_command_results_ibfk_2` FOREIGN KEY (`command_uuid`) REFERENCES `windows_mdm_commands` (`command_uuid`) ON DELETE CASCADE ON UPDATE CASCADE
	for _, constraint := range constraints {
		quotedConstraint := "`" + strings.ReplaceAll(constraint, "`", "``") + "`"
		if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY %s;", table, quotedConstraint)); err != nil {
			return fmt.Errorf("dropping fk %s: %w", constraint, err)
		}
	}
	return nil
}

func Down_20260723181401(_ *sql.Tx) error {
	return nil
}
