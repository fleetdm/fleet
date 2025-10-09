package tables

import (
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251009091733, Down_20251009091733)
}

func Up_20251009091733(tx *sql.Tx) error {
	//
	// NOTE: This migration was copied from server/datastore/mysql/migrations/data/20210330130314_UpdateBuiltinLabels.go.
	// We are now running this on the "tables" migration, because the "data" migrations (deprecated) run after
	// the "tables" migration, and so this causes differences for
	// builtin labels between environments depending on when they deployed Fleet initially.
	//
	// E.g., user deploys Fleet 4.54.0 on environment "A", which means:
	// 	1. Run all table migrations up to 4.54.0.
	//	2. Run all data migrations.
	// Then the user upgrades to 4.76.0, which means:
	//	1. Run some table migrations X, Y, and Z up to 4.76.0. (X, Y, and Z being some new tables or alter)
	//
	// Another user deploys Fleet 4.76.0 on environment "B", which means:
	// 	1. Run all table migrations up to 4.76.0 (which include X, Y, and Z).
	//	2. Run data migrations.
	//
	// So you can see that on environment "A", X, Y, and Z migrations were applied _after_ all data migrations,
	// whereas on environment "B" they were applied before them.
	//
	// This caused a difference in particular with builtin labels which were cleared of their platform on an old data migration
	// (server/datastore/mysql/migrations/data/20210330130314_UpdateBuiltinLabels.go)
	//
	// To solve this we run the builtin labels platform clearing on a "tables" migration.
	//
	// Finally, why are we clearing platform on all builtin labels?
	//	- We want to bring all environments to the same state (no platforms on builtin labels).
	// 	- For builtin "dynamic" labels we want to run all label queries on hosts to cover the scenario of a user installing another OS.
	//    E.g. installing Windows on a host that previously had Ubuntu: if the label platform is not empty then the label
	//    won't run on the host to clear the membership status and will have both "Windows" and "Linux" labels (see #33065 and #33245).
	//	- For builtin "manual" labels, "platform" does not provide any purpose.
	//	- At the time of writing there are no builtin "host vital" labels.
	//

	// Use a constant time so that the generated schema is deterministic
	updatedAt := time.Date(2025, 10, 9, 0, 0, 0, 0, time.UTC)

	sql := "UPDATE labels SET platform = '', updated_at = ? WHERE label_type = ?"
	if _, err := tx.Exec(sql, updatedAt, fleet.LabelTypeBuiltIn); err != nil {
		return errors.Wrap(err, "clear platform column on all builtin labels")
	}
	return nil
}

func Down_20251009091733(tx *sql.Tx) error {
	return nil
}
