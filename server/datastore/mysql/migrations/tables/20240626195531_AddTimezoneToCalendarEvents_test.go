package tables

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20240626195531(t *testing.T) {
	db := applyUpToPrev(t)

	// insert data to prev schema
	sampleEvent := fleet.CalendarEvent{
		Email:     "foo@example.com",
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC().Add(30 * time.Minute),
		Data:      []byte("{\"foo\": \"bar\"}"),
	}
	sampleEvent.ID = uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
		`INSERT INTO calendar_events (email, start_time, end_time, event) VALUES (?, ?, ?, ?);`,
		sampleEvent.Email, sampleEvent.StartTime, sampleEvent.EndTime, sampleEvent.Data,
	))

	sampleHostEvent := fleet.HostCalendarEvent{
		HostID:          1,
		CalendarEventID: sampleEvent.ID,
		WebhookStatus:   fleet.CalendarWebhookStatusPending,
	}
	sampleHostEvent.ID = uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
		`INSERT INTO host_calendar_events (host_id, calendar_event_id, webhook_status) VALUES (?, ?, ?);`,
		sampleHostEvent.HostID, sampleHostEvent.CalendarEventID, sampleHostEvent.WebhookStatus,
	))

	// apply migration
	applyNext(t, db)

	// verify migration
	// check that it's NULL
	selectTzStmt := `SELECT timezone FROM calendar_events WHERE id = ?`
	var dbOutTz string
	err := db.Get(&dbOutTz, selectTzStmt, sampleEvent.ID)
	require.Error(t, err) // db.Get returns error if empty result set, which we expect

	// insert a timezone
	testTz := "America/Argentina/Buenos_Aires"
	execNoErr(t, db, `UPDATE calendar_events SET timezone = ? WHERE id = ?`, testTz, sampleEvent.ID)

	// check that it comes out unchanged
	err = db.Get(&dbOutTz, `SELECT timezone FROM calendar_events WHERE id = ?;`, sampleEvent.ID)
	require.NoError(t, err)
	require.Equal(t, testTz, dbOutTz)
}
