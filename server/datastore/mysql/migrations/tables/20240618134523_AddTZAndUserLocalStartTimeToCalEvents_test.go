package tables

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20240618134523(t *testing.T) {
	db := applyUpToPrev(t)

	sT := time.Now().UTC()
	// insert data to prev schema
	sampleEvent := fleet.CalendarEvent{
		Email:     "foo@example.com",
		StartTime: sT,
		EndTime:   sT.Add(30 * time.Minute),
		Data:      []byte("{\"foo\": \"bar\"}"),
	}
	sampleEvent.ID = uint(execNoErrLastID(t, db,
		`INSERT INTO calendar_events (email, start_time, end_time, event) VALUES (?, ?, ?, ?);`,
		sampleEvent.Email, sampleEvent.StartTime, sampleEvent.EndTime, sampleEvent.Data,
	))

	sampleHostEvent := fleet.HostCalendarEvent{
		HostID:          1,
		CalendarEventID: sampleEvent.ID,
		WebhookStatus:   fleet.CalendarWebhookStatusPending,
	}
	sampleHostEvent.ID = uint(execNoErrLastID(t, db,
		`INSERT INTO host_calendar_events (host_id, calendar_event_id, webhook_status) VALUES (?, ?, ?);`,
		sampleHostEvent.HostID, sampleHostEvent.CalendarEventID, sampleHostEvent.WebhookStatus,
	))

	// apply migration
	applyNext(t, db)

	// verify migration
	// check that new column values are NULL
	selectTzStmt := `SELECT timezone FROM calendar_events WHERE id = ?`
	var dbOutTz string
	err := db.Get(&dbOutTz, selectTzStmt, sampleEvent.ID)
	require.Error(t, err) // db.Get returns error if empty result set, which we expect

	selectULSTStmt := `SELECT user_local_start_time FROM calendar_events WHERE id = ?`
	var dbOutULST string
	err = db.Get(&dbOutULST, selectULSTStmt, sampleEvent.ID)
	require.Error(t, err)

	// insert a timezone and user-local start time
	testTz := "America/Argentina/Buenos_Aires"
	testUserLocation, err := time.LoadLocation(testTz)
	require.NoError(t, err)
	testULST := sT.In(testUserLocation).String()
	execNoErr(t, db, `UPDATE calendar_events SET timezone = ?, user_local_start_time = ? WHERE id = ?`, testTz, testULST, sampleEvent.ID)

	// check that they come out unchanged
	err = db.Get(&dbOutTz, `SELECT timezone FROM calendar_events WHERE id = ?;`, sampleEvent.ID)
	require.NoError(t, err)
	require.Equal(t, testTz, dbOutTz)
	err = db.Get(&dbOutULST, `SELECT user_local_start_time FROM calendar_events WHERE id = ?;`, sampleEvent.ID)
	require.NoError(t, err)
	require.Equal(t, testULST, dbOutULST)
}
