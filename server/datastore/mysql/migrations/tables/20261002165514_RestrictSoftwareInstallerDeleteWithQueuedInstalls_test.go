package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261002165514(t *testing.T) {
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

func TestUp_20261002165514_PartiallyApplied(t *testing.T) {
	db := applyUpToPrev(t)

	// the first swap succeeded, the second didn't run
	execNoErr(t, db, `ALTER TABLE software_install_upcoming_activities
		DROP FOREIGN KEY fk_software_install_upcoming_activities_software_installer_id,
		ADD CONSTRAINT fk_siua_software_installer_id_tmp FOREIGN KEY (software_installer_id)
		REFERENCES software_installers (id) ON UPDATE CASCADE`)

	applyNext(t, db)

	var rules []string
	require.NoError(t, db.Select(&rules, `
		SELECT CONCAT(CONSTRAINT_NAME, ' ', DELETE_RULE) FROM information_schema.REFERENTIAL_CONSTRAINTS
		WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'software_install_upcoming_activities'
		AND REFERENCED_TABLE_NAME = 'software_installers'`))
	require.Equal(t, []string{"fk_software_install_upcoming_activities_software_installer_id NO ACTION"}, rules)
}
