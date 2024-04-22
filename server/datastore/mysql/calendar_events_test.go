package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCalendarEvents(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpdateCalendarEvent", testUpdateCalendarEvent},
		{"CreateOrUpdateCalendarEvent", testCreateOrUpdateCalendarEvent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testUpdateCalendarEvent(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	err = ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "foo@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	startTime1 := time.Now()
	endTime1 := startTime1.Add(30 * time.Minute)
	calendarEvent, err := ds.CreateOrUpdateCalendarEvent(ctx, "foo@example.com", startTime1, endTime1, []byte(`{}`), host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	err = ds.UpdateCalendarEvent(ctx, calendarEvent.ID, startTime1, endTime1, []byte(`{}`))
	require.NoError(t, err)

	calendarEvent2, err := ds.GetCalendarEvent(ctx, "foo@example.com")
	require.NoError(t, err)
	require.NotEqual(t, *calendarEvent, *calendarEvent2)
	calendarEvent.UpdatedAt = calendarEvent2.UpdatedAt
	require.Equal(t, *calendarEvent, *calendarEvent2)

	// TODO(lucas): Add more tests here.
}

func testCreateOrUpdateCalendarEvent(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	err = ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "foo@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	startTime1 := time.Now()
	endTime1 := startTime1.Add(30 * time.Minute)
	calendarEvent, err := ds.CreateOrUpdateCalendarEvent(ctx, "foo@example.com", startTime1, endTime1, []byte(`{}`), host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	calendarEvent2, err := ds.CreateOrUpdateCalendarEvent(ctx, "foo@example.com", startTime1, endTime1, []byte(`{}`), host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)
	require.Greater(t, calendarEvent2.UpdatedAt, calendarEvent.UpdatedAt)
	calendarEvent.UpdatedAt = calendarEvent2.UpdatedAt
	require.Equal(t, *calendarEvent, *calendarEvent2)

	time.Sleep(1 * time.Second)

	startTime2 := startTime1.Add(1 * time.Hour)
	endTime2 := startTime1.Add(30 * time.Minute)
	calendarEvent3, err := ds.CreateOrUpdateCalendarEvent(ctx, "foo@example.com", startTime2, endTime2, []byte(`{"foo": "bar"}`), host.ID, fleet.CalendarWebhookStatusPending)
	require.NoError(t, err)
	require.Greater(t, calendarEvent3.UpdatedAt, calendarEvent2.UpdatedAt)
	require.WithinDuration(t, startTime2, calendarEvent3.StartTime, 1*time.Second)
	require.WithinDuration(t, endTime2, calendarEvent3.EndTime, 1*time.Second)
	require.Equal(t, string(calendarEvent3.Data), `{"foo": "bar"}`)

	calendarEvent3b, err := ds.GetCalendarEvent(ctx, "foo@example.com")
	require.NoError(t, err)
	require.Equal(t, calendarEvent3, calendarEvent3b)

	// TODO(lucas): Add more tests here.
}
