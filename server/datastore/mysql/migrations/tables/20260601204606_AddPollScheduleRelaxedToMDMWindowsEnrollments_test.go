package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260601204606(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Selecting the new column succeeds only if the migration added it.
	_, err := db.Exec(`SELECT poll_schedule_relaxed FROM mdm_windows_enrollments LIMIT 0`)
	require.NoError(t, err)

	// New enrollments default to not-relaxed (the fast poll), matching NewDMClientProvisioningData.
	var def string
	require.NoError(t, db.Get(&def,
		`SELECT COLUMN_DEFAULT FROM INFORMATION_SCHEMA.COLUMNS
		 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'mdm_windows_enrollments' AND COLUMN_NAME = 'poll_schedule_relaxed'`))
	require.Equal(t, "0", def)
}
