package tables

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20240314085226(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

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

	var event fleet.CalendarEvent
	err := db.Get(&event, `SELECT * FROM calendar_events WHERE id = ?;`, sampleEvent.ID)
	require.NoError(t, err)
	sampleEvent.CreatedAt = event.CreatedAt // sampleEvent doesn't have this set.
	sampleEvent.UpdatedAt = event.UpdatedAt // sampleEvent doesn't have this set.
	sampleEvent.StartTime = sampleEvent.StartTime.Round(time.Second)
	sampleEvent.EndTime = sampleEvent.EndTime.Round(time.Second)
	event.StartTime = event.StartTime.Round(time.Second)
	event.EndTime = event.EndTime.Round(time.Second)
	require.Equal(t, sampleEvent, event)

	var hostEvent fleet.HostCalendarEvent
	err = db.Get(&hostEvent, `SELECT * FROM host_calendar_events WHERE id = ?;`, sampleHostEvent.ID)
	require.NoError(t, err)
	sampleHostEvent.CreatedAt = hostEvent.CreatedAt // sampleHostEvent doesn't have this set.
	sampleHostEvent.UpdatedAt = hostEvent.UpdatedAt // sampleHostEvent doesn't have this set.
	require.Equal(t, sampleHostEvent, hostEvent)
}
