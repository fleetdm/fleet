package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20251027101155(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := insertHost(t, db, nil)
	contentIDs := insertScriptContents(t, db, 1)

	// create an upcoming activity for script run on that host
	execID := uuid.NewString()
	uaID := execNoErrLastID(t, db, `INSERT INTO upcoming_activities (
		host_id, activity_type, execution_id, payload
	) VALUES (?, ?, ?, ?)`, hostID, "script", execID, `{}`)

	execNoErr(t, db, `INSERT INTO script_upcoming_activities (
		upcoming_activity_id, script_content_id
	) VALUES (?, ?)`, uaID, contentIDs[0])

	// Apply current migration.
	applyNext(t, db)

	assertRowCount(t, db, "upcoming_activities", 1)

	// activity type is still "script"
	var activityType string
	err := db.Get(&activityType, "SELECT activity_type FROM upcoming_activities WHERE id = ?", uaID)
	require.NoError(t, err)
	require.Equal(t, "script", activityType)

	// activity can now be in_house_app_install
	execID2 := uuid.NewString()
	uaID2 := execNoErrLastID(t, db, `INSERT INTO upcoming_activities (
		host_id, activity_type, execution_id, payload
	) VALUES (?, ?, ?, ?)`, hostID, "in_house_app_install", execID2, `{}`)

	err = db.Get(&activityType, "SELECT activity_type FROM upcoming_activities WHERE id = ?", uaID2)
	require.NoError(t, err)
	require.Equal(t, "in_house_app_install", activityType)
}
