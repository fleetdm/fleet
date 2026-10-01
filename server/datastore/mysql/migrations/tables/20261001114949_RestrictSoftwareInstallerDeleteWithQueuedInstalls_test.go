package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261001114949(t *testing.T) {
	db := applyUpToPrev(t)

	uaID := execNoErrLastID(t, db, `
		INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
		VALUES (1, 'software_install', 'orphan-exec', '{}')`)
	execNoErr(t, db, `
		INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_installer_id)
		VALUES (?, NULL)`, uaID)

	applyNext(t, db)

	var rules struct {
		DeleteRule string `db:"DELETE_RULE"`
		UpdateRule string `db:"UPDATE_RULE"`
	}
	require.NoError(t, db.Get(&rules, `
		SELECT DELETE_RULE, UPDATE_RULE FROM information_schema.REFERENTIAL_CONSTRAINTS
		WHERE CONSTRAINT_SCHEMA = DATABASE()
		AND CONSTRAINT_NAME = 'fk_software_install_upcoming_activities_software_installer_id'`))
	require.Equal(t, "NO ACTION", rules.DeleteRule)
	require.Equal(t, "CASCADE", rules.UpdateRule)

	require.Equal(t, []string{"software_installer_id"},
		indexColumns(t, db, "software_install_upcoming_activities", "fk_software_install_upcoming_activities_software_installer_id"))

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM software_install_upcoming_activities WHERE upcoming_activity_id = ?`, uaID))
	require.Equal(t, 1, count)
}
