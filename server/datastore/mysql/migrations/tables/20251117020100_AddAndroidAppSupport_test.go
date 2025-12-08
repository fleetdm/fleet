package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20251117020100(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// These tables should still have the FKs they had before the migration
	for tableName, fkName := range map[string]string{
		"vpp_app_upcoming_activities": "fk_vpp_app_upcoming_activities_adam_id_platform",
		"host_vpp_software_installs":  "host_vpp_software_installs_ibfk_3",
		"vpp_apps_teams":              "vpp_apps_teams_ibfk_3",
	} {
		var columnNames []string
		err := db.Select(&columnNames, `
		SELECT
			COLUMN_NAME
		FROM
		  INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE
		  REFERENCED_TABLE_SCHEMA = (SELECT DATABASE()) AND
		  TABLE_NAME = ? AND CONSTRAINT_NAME = ?`, tableName, fkName)
		require.NoError(t, err)

		assert.ElementsMatch(t, columnNames, []string{"adam_id", "platform"})
	}
}
