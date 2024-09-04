package calendar

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar/load_test"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type googleCalendarIntegrationTestSuite struct {
	suite.Suite
	server *httptest.Server
	dbFile *os.File
}

func (s *googleCalendarIntegrationTestSuite) SetupSuite() {
	dbFile, err := os.CreateTemp("", "calendar.db")
	s.Require().NoError(err)
	handler, err := calendartest.Configure(dbFile.Name())
	s.Require().NoError(err)
	server := httptest.NewUnstartedServer(handler)
	server.Listener.Addr()
	server.Start()
	s.server = server
}

func (s *googleCalendarIntegrationTestSuite) TearDownSuite() {
	if s.dbFile != nil {
		s.dbFile.Close()
		_ = os.Remove(s.dbFile.Name())
	}
	if s.server != nil {
		s.server.Close()
	}
	calendartest.Close()
}

// TestGoogleCalendarIntegration tests should be able to be run in parallel, but this is not natively supported by suites: https://github.com/stretchr/testify/issues/187
// There are workarounds that can be explored.
func TestIntegrationsGoogleCalendar(t *testing.T) {
	testingSuite := new(googleCalendarIntegrationTestSuite)
	suite.Run(t, testingSuite)
}

func (s *googleCalendarIntegrationTestSuite) TestCreateGetDeleteEvent() {
	t := s.T()
	userEmail := "user1@example.com"
	config := &GoogleCalendarConfig{
		Context: context.Background(),
		IntegrationConfig: &fleet.GoogleCalendarIntegration{
			Domain: "example.com",
			ApiKey: map[string]string{
				"client_email": loadEmail,
				"private_key":  s.server.URL,
			},
		},
		Logger: kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout)),
	}
	gCal := NewGoogleCalendar(config)
	err := gCal.Configure(userEmail)
	require.NoError(t, err)
	var genBodyFn fleet.CalendarGenBodyFn = func(bool) (string, bool, error) {
		return "Test event", true, nil
	}
	eventDate := time.Now().Add(48 * time.Hour)
	event, err := gCal.CreateEvent(eventDate, genBodyFn, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, startHour, event.StartTime.Hour())
	assert.Equal(t, 0, event.StartTime.Minute())
	details, err := gCal.unmarshalDetails(event)
	require.NoError(t, err)
	eventUUID := event.UUID
	channelID := details.ChannelID
	resourceID := details.ResourceID
	assert.NotEmpty(t, eventUUID)
	assert.NotEmpty(t, channelID)
	assert.NotEmpty(t, resourceID)

	eventRsp, updated, err := gCal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, event, eventRsp)

	err = gCal.DeleteEvent(event)
	assert.NoError(t, err)
	// delete again
	err = gCal.DeleteEvent(event)
	assert.NoError(t, err)

	// Try to get deleted event
	eventRsp, updated, err = gCal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	require.NoError(t, err)
	assert.True(t, updated)
	assert.NotEqual(t, event.StartTime.UTC().Truncate(24*time.Hour), eventRsp.StartTime.UTC().Truncate(24*time.Hour))

	opts := fleet.CalendarCreateEventOpts{
		ChannelID:  channelID,
		ResourceID: resourceID,
		EventUUID:  eventUUID,
	}
	event, err = gCal.CreateEvent(eventDate, genBodyFn, opts)
	require.NoError(t, err)
	assert.Equal(t, startHour, event.StartTime.Hour())
	assert.Equal(t, 0, event.StartTime.Minute())
	details, err = gCal.unmarshalDetails(event)
	require.NoError(t, err)
	assert.Equal(t, channelID, details.ChannelID)
	assert.Equal(t, resourceID, details.ResourceID)
	assert.Equal(t, eventUUID, event.UUID)
}

func (s *googleCalendarIntegrationTestSuite) TestFillUpCalendar() {
	t := s.T()
	userEmail := "user2@example.com"
	config := &GoogleCalendarConfig{
		Context: context.Background(),
		IntegrationConfig: &fleet.GoogleCalendarIntegration{
			Domain: "example.com",
			ApiKey: map[string]string{
				"client_email": loadEmail,
				"private_key":  s.server.URL,
			},
		},
		Logger: kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout)),
	}
	gCal := NewGoogleCalendar(config)
	err := gCal.Configure(userEmail)
	require.NoError(t, err)
	genBodyFn := func(bool) (string, bool, error) {
		return "Test event", true, nil
	}
	eventDate := time.Now().Add(48 * time.Hour)
	event, err := gCal.CreateEvent(eventDate, genBodyFn, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, startHour, event.StartTime.Hour())
	assert.Equal(t, 0, event.StartTime.Minute())

	currentEventTime := event.StartTime
	for i := 0; i < 20; i++ {
		if !(currentEventTime.Hour() == endHour-1 && currentEventTime.Minute() == 30) {
			currentEventTime = currentEventTime.Add(30 * time.Minute)
		}
		event, err = gCal.CreateEvent(eventDate, genBodyFn, fleet.CalendarCreateEventOpts{})
		require.NoError(t, err)
		assert.Equal(t, currentEventTime.UTC(), event.StartTime.UTC())
	}

}
