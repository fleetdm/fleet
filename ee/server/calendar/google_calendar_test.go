package calendar

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

const (
	baseServiceEmail = "service@example.com"
	basePrivateKey   = "private-key"
	baseUserEmail    = "user@example.com"
	baseServerURL    = "https://example.com"
)

var (
	baseCtx = context.Background()
	logger  = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
)

type MockGoogleCalendarLowLevelAPI struct {
	ConfigureFunc   func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL string) error
	GetSettingFunc  func(name string) (*calendar.Setting, error)
	ListEventsFunc  func(timeMin, timeMax string) (*calendar.Events, error)
	CreateEventFunc func(event *calendar.Event) (*calendar.Event, error)
	UpdateEventFunc func(event *calendar.Event) (*calendar.Event, error)
	GetEventFunc    func(id, eTag string) (*calendar.Event, error)
	DeleteEventFunc func(id string) error
	WatchFunc       func(eventUUID string, channelID string, ttl uint64) (resourceID string, err error)
	StopFunc        func(channelID string, resourceID string) error
}

func (m *MockGoogleCalendarLowLevelAPI) Watch(eventUUID string, channelID string, ttl uint64) (resourceID string, err error) {
	return m.WatchFunc(eventUUID, channelID, ttl)
}

func (m *MockGoogleCalendarLowLevelAPI) Stop(channelID string, resourceID string) error {
	return m.StopFunc(channelID, resourceID)
}

func (m *MockGoogleCalendarLowLevelAPI) Configure(
	ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL string,
) error {
	return m.ConfigureFunc(ctx, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL)
}

func (m *MockGoogleCalendarLowLevelAPI) GetSetting(name string) (*calendar.Setting, error) {
	return m.GetSettingFunc(name)
}

func (m *MockGoogleCalendarLowLevelAPI) ListEvents(timeMin, timeMax string) (*calendar.Events, error) {
	return m.ListEventsFunc(timeMin, timeMax)
}

func (m *MockGoogleCalendarLowLevelAPI) CreateEvent(event *calendar.Event) (*calendar.Event, error) {
	return m.CreateEventFunc(event)
}

func (m *MockGoogleCalendarLowLevelAPI) UpdateEvent(event *calendar.Event) (*calendar.Event, error) {
	return m.UpdateEventFunc(event)
}

func (m *MockGoogleCalendarLowLevelAPI) GetEvent(id, eTag string) (*calendar.Event, error) {
	return m.GetEventFunc(id, eTag)
}

func (m *MockGoogleCalendarLowLevelAPI) DeleteEvent(id string) error {
	return m.DeleteEventFunc(id)
}

func TestGoogleCalendar_Configure(t *testing.T) {
	t.Parallel()
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL string) error {
		assert.Equal(t, baseCtx, ctx)
		assert.Equal(t, baseServiceEmail, serviceAccountEmail)
		assert.Equal(t, basePrivateKey, privateKey)
		assert.Equal(t, baseUserEmail, userToImpersonateEmail)
		assert.Equal(t, baseServerURL, serverURL)
		return nil
	}

	// Happy path test
	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(baseUserEmail)
	assert.NoError(t, err)

	// Configure error test
	mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL string) error {
		return assert.AnError
	}
	err = cal.Configure(baseUserEmail)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGoogleCalendar_ConfigurePlusAddressing(t *testing.T) {
	// Do not run this test in t.Parallel(), since it involves modifying a global variable
	plusAddressing = true
	t.Cleanup(
		func() {
			plusAddressing = false
		},
	)
	email := "user+my_test+email@example.com"
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL string) error {
		assert.Equal(t, baseCtx, ctx)
		assert.Equal(t, baseServiceEmail, serviceAccountEmail)
		assert.Equal(t, basePrivateKey, privateKey)
		assert.Equal(t, baseServerURL, serverURL)
		assert.Equal(t, "user@example.com", userToImpersonateEmail)
		return nil
	}

	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(email)
	assert.NoError(t, err)
}

func makeConfig(mockAPI *MockGoogleCalendarLowLevelAPI) *GoogleCalendarConfig {
	if mockAPI != nil && mockAPI.ConfigureFunc == nil {
		mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail, serverURL string) error {
			return nil
		}
	}
	config := &GoogleCalendarConfig{
		Context: context.Background(),
		IntegrationConfig: &fleet.GoogleCalendarIntegration{
			ApiKey: map[string]string{
				fleet.GoogleCalendarEmail:      baseServiceEmail,
				fleet.GoogleCalendarPrivateKey: basePrivateKey,
			},
		},
		Logger:    logger,
		API:       mockAPI,
		ServerURL: baseServerURL,
	}
	return config
}

func TestGoogleCalendar_DeleteEvent(t *testing.T) {
	t.Parallel()
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	mockAPI.DeleteEventFunc = func(id string) error {
		assert.Equal(t, "event-id", id)
		return nil
	}

	// Happy path test
	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(baseUserEmail)
	assert.NoError(t, err)
	err = cal.DeleteEvent(&fleet.CalendarEvent{Data: []byte(`{"ID":"event-id"}`)})
	assert.NoError(t, err)

	// API error test
	mockAPI.DeleteEventFunc = func(id string) error {
		return assert.AnError
	}
	err = cal.DeleteEvent(&fleet.CalendarEvent{Data: []byte(`{"ID":"event-id"}`)})
	assert.ErrorIs(t, err, assert.AnError)

	// Event already deleted
	mockAPI.DeleteEventFunc = func(id string) error {
		return &googleapi.Error{Code: http.StatusGone}
	}
	err = cal.DeleteEvent(&fleet.CalendarEvent{Data: []byte(`{"ID":"event-id"}`)})
	assert.NoError(t, err)
}

func TestGoogleCalendar_unmarshalDetails(t *testing.T) {
	t.Parallel()
	var gCal = NewGoogleCalendar(makeConfig(&MockGoogleCalendarLowLevelAPI{}))
	err := gCal.Configure(baseUserEmail)
	assert.NoError(t, err)
	details, err := gCal.unmarshalDetails(&fleet.CalendarEvent{Data: []byte(`{"id":"event-id","etag":"event-eTag"}`)})
	assert.NoError(t, err)
	assert.Equal(t, "event-id", details.ID)
	assert.Equal(t, "event-eTag", details.ETag)

	// Missing ETag is OK
	details, err = gCal.unmarshalDetails(&fleet.CalendarEvent{Data: []byte(`{"id":"event-id"}`)})
	assert.NoError(t, err)
	assert.Equal(t, "event-id", details.ID)
	assert.Equal(t, "", details.ETag)

	// Bad JSON
	_, err = gCal.unmarshalDetails(&fleet.CalendarEvent{Data: []byte(`{"bozo`)})
	assert.Error(t, err)

	// Missing id
	_, err = gCal.unmarshalDetails(&fleet.CalendarEvent{Data: []byte(`{"myId":"event-id","etag":"event-eTag"}`)})
	assert.Error(t, err)
}

func TestGoogleCalendar_GetAndUpdateEvent(t *testing.T) {
	t.Parallel()
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	const baseETag = "event-eTag"
	const baseEventID = "event-id"
	const baseResourceID = "resource-id"
	baseTzName := "America/New_York"
	baseTzLocation, _ := time.LoadLocation(baseTzName)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		assert.Equal(t, baseEventID, id)
		assert.Equal(t, baseETag, eTag)
		return &calendar.Event{
			Etag: baseETag, // ETag matches -- no modifications to event
		}, nil
	}
	mockAPI.GetSettingFunc = func(name string) (*calendar.Setting, error) {
		return &calendar.Setting{Value: baseTzName}, nil
	}
	var genBodyFn fleet.CalendarGenBodyFn = func(bool) (string, bool, error) {
		t.Error("genBodyFn should not be called")
		return "event-body", false, nil
	}
	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(baseUserEmail)
	assert.NoError(t, err)

	eventStartTime := time.Now().In(baseTzLocation)
	event := &fleet.CalendarEvent{
		StartTime: eventStartTime,
		EndTime:   time.Now().Add(time.Hour).In(baseTzLocation),
		Data:      []byte(`{"ID":"` + baseEventID + `","ETag":"` + baseETag + `"}`),
		TimeZone:  &baseTzName,
	}

	// ETag matches
	retrievedEvent, updated, err := cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, event, retrievedEvent)

	// http.StatusNotModified response (ETag matches)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return nil, &googleapi.Error{Code: http.StatusNotModified}
	}
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, event, retrievedEvent)

	// Cannot unmarshal details
	eventBadDetails := &fleet.CalendarEvent{
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
		Data:      []byte(`{"bozo`),
	}
	_, _, err = cal.GetAndUpdateEvent(eventBadDetails, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.Error(t, err)

	// API error test
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return nil, assert.AnError
	}
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.ErrorIs(t, err, assert.AnError)

	// Event has been modified
	startTime := time.Now().Add(time.Minute).Truncate(time.Second)
	endTime := time.Now().Add(time.Hour).Truncate(time.Second)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag",
			Start: &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)},
			End:   &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)},
		}, nil
	}
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.NotEqual(t, event, retrievedEvent)
	require.NotNil(t, retrievedEvent)
	assert.Equal(t, startTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, endTime.UTC(), retrievedEvent.EndTime.UTC())
	assert.Equal(t, baseUserEmail, retrievedEvent.Email)
	gCal, _ := cal.(*GoogleCalendar)
	details, err := gCal.unmarshalDetails(retrievedEvent)
	require.NoError(t, err)
	assert.Equal(t, "new-eTag", details.ETag)
	assert.Equal(t, baseEventID, details.ID)

	// missing end time
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag",
			Start: &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)},
			End:   &calendar.EventDateTime{DateTime: ""},
		}, nil
	}
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.Error(t, err)

	// missing start time
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:   baseEventID,
			Etag: "new-eTag",
			End:  &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)},
		}, nil
	}
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.Error(t, err)

	// Bad time format
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag",
			Start: &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)},
			End:   &calendar.EventDateTime{DateTime: "bozo"},
		}, nil
	}
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	assert.Error(t, err)

	// Event has been modified, with custom timezone.
	newTzName := "Africa/Kinshasa"
	newTzLocation, _ := time.LoadLocation(newTzName)
	mockAPI.GetSettingFunc = func(name string) (*calendar.Setting, error) {
		return &calendar.Setting{Value: newTzName}, nil
	}
	startTime = time.Now().Add(time.Minute).Truncate(time.Second).In(newTzLocation)
	endTime = time.Now().Add(time.Hour).Truncate(time.Second).In(newTzLocation)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag",
			Start: &calendar.EventDateTime{DateTime: startTime.UTC().Format(time.RFC3339), TimeZone: newTzName},
			End:   &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339), TimeZone: newTzName},
		}, nil
	}
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{UpdateTimezone: true})
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.NotEqual(t, event, retrievedEvent)
	require.NotNil(t, retrievedEvent)
	assert.Equal(t, startTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, endTime.UTC(), retrievedEvent.EndTime.UTC())
	assert.Equal(t, baseUserEmail, retrievedEvent.Email)

	// 404 response (deleted)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return nil, &googleapi.Error{Code: http.StatusNotFound}
	}
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return &calendar.Events{}, nil
	}
	mockAPI.StopFunc = func(channelID string, resourceID string) error {
		details, err := gCal.unmarshalDetails(event)
		require.NoError(t, err)
		assert.Equal(t, details.ChannelID, channelID)
		assert.Equal(t, details.ResourceID, resourceID)
		return nil
	}
	var uuid, channelUUID string
	mockAPI.WatchFunc = func(eventUUID string, channelID string, ttl uint64) (resourceID string, err error) {
		uuid = eventUUID
		channelUUID = channelID
		assert.Greater(t, ttl, uint64(60*30-1))
		return baseResourceID, nil
	}
	genBodyFn = func(conflict bool) (string, bool, error) {
		assert.False(t, conflict)
		return "event-body", true, nil
	}
	eventCreated := false
	mockAPI.CreateEventFunc = func(event *calendar.Event) (*calendar.Event, error) {
		assert.Equal(t, eventTitle, event.Summary)
		body, _, _ := genBodyFn(false)
		assert.Equal(t, body, event.Description)
		event.Id = baseEventID
		event.Etag = baseETag
		eventCreated = true
		return event, nil
	}
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	require.NoError(t, err)
	assert.True(t, updated)
	assert.NotEqual(t, event, retrievedEvent)
	require.NotNil(t, retrievedEvent)
	assert.Equal(t, uuid, retrievedEvent.UUID)
	assert.Equal(t, baseUserEmail, retrievedEvent.Email)
	newEventDate := calculateNewEventDate(eventStartTime)
	expectedStartTime := time.Date(newEventDate.Year(), newEventDate.Month(), newEventDate.Day(), startHour, 0, 0, 0, newTzLocation)
	assert.Equal(t, expectedStartTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), retrievedEvent.EndTime.UTC())
	assert.True(t, eventCreated)
	details, err = gCal.unmarshalDetails(retrievedEvent)
	require.NoError(t, err)
	assert.Equal(t, channelUUID, details.ChannelID)
	assert.Equal(t, baseResourceID, details.ResourceID)

	// cancelled (deleted)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:     baseEventID,
			Etag:   "new-eTag",
			Start:  &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)},
			End:    &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)},
			Status: "cancelled",
		}, nil
	}
	eventCreated = false
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	require.NoError(t, err)
	assert.True(t, updated)
	require.NotNil(t, retrievedEvent)
	assert.NotEqual(t, event, retrievedEvent)
	assert.Equal(t, expectedStartTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), retrievedEvent.EndTime.UTC())
	assert.True(t, eventCreated)

	// all day event (deleted)
	mockAPI.DeleteEventFunc = func(id string) error {
		assert.Equal(t, baseEventID, id)
		return nil
	}
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag",
			Start: &calendar.EventDateTime{Date: startTime.Format("2006-01-02")},
			End:   &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)},
		}, nil
	}
	eventCreated = false
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	require.NoError(t, err)
	assert.True(t, updated)
	require.NotNil(t, retrievedEvent)
	assert.NotEqual(t, event, retrievedEvent)
	assert.Equal(t, expectedStartTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), retrievedEvent.EndTime.UTC())
	assert.True(t, eventCreated)

	// moved in the past event (deleted)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag in past",
			Start: &calendar.EventDateTime{DateTime: startTime.Add(-2 * time.Hour).Format(time.RFC3339)},
			End:   &calendar.EventDateTime{DateTime: endTime.Add(-2 * time.Hour).Format(time.RFC3339)},
		}, nil
	}
	eventCreated = false
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	require.NoError(t, err)
	assert.True(t, updated)
	require.NotNil(t, retrievedEvent)
	assert.NotEqual(t, event, retrievedEvent)
	assert.Equal(t, expectedStartTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), retrievedEvent.EndTime.UTC())
	assert.True(t, eventCreated)
}

func TestGoogleCalendar_CreateEvent(t *testing.T) {
	t.Parallel()
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	const baseEventID = "event-id"
	const baseETag = "event-eTag"
	const eventBody = "event-body"
	const baseResourceID = "resource-id"
	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(baseUserEmail)
	assert.NoError(t, err)

	tzId := "Africa/Kinshasa"
	mockAPI.GetSettingFunc = func(name string) (*calendar.Setting, error) {
		return &calendar.Setting{Value: tzId}, nil
	}
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return &calendar.Events{}, nil
	}
	mockAPI.CreateEventFunc = func(event *calendar.Event) (*calendar.Event, error) {
		assert.Equal(t, eventTitle, event.Summary)
		assert.Equal(t, eventBody, event.Description)
		event.Id = baseEventID
		event.Etag = baseETag
		return event, nil
	}
	genBodyFn := func(conflict bool) (string, bool, error) {
		assert.False(t, conflict)
		return eventBody, true, nil
	}
	genBodyConflictFn := func(conflict bool) (string, bool, error) {
		assert.True(t, conflict)
		return eventBody, true, nil
	}

	// Happy path test -- empty calendar
	date := time.Now().Add(48 * time.Hour)
	location, _ := time.LoadLocation(tzId)
	expectedStartTime := time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, location)
	_, expectedOffset := expectedStartTime.Zone()
	var uuid, channelUUID string
	mockAPI.WatchFunc = func(eventUUID string, channelID string, ttl uint64) (resourceID string, err error) {
		uuid = eventUUID
		channelUUID = channelID
		assert.Greater(t, ttl, uint64(60*30-1))
		return baseResourceID, nil
	}
	event, err := cal.CreateEvent(date, genBodyFn, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, uuid, event.UUID)
	assert.Equal(t, baseUserEmail, event.Email)
	assert.Equal(t, expectedStartTime.UTC(), event.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), event.EndTime.UTC())
	_, offset := event.StartTime.Zone()
	assert.Equal(t, expectedOffset, offset)
	_, offset = event.EndTime.Zone()
	assert.Equal(t, expectedOffset, offset)
	gCal, _ := cal.(*GoogleCalendar)
	details, err := gCal.unmarshalDetails(event)
	require.NoError(t, err)
	assert.Equal(t, baseETag, details.ETag)
	assert.Equal(t, baseEventID, details.ID)
	assert.Equal(t, channelUUID, details.ChannelID)
	assert.Equal(t, baseResourceID, details.ResourceID)
	assert.Equal(t, tzId, *event.TimeZone)

	// Workday already ended
	date = time.Now().Add(-48 * time.Hour)
	_, err = cal.CreateEvent(date, genBodyFn, fleet.CalendarCreateEventOpts{})
	assert.ErrorAs(t, err, &fleet.DayEndedError{})

	// There is no time left in the day to schedule an event
	date = time.Now().Add(48 * time.Hour)
	timeNow := func() time.Time {
		now := time.Date(date.Year(), date.Month(), date.Day(), endHour-1, 45, 0, 0, location)
		return now
	}
	_, err = gCal.createEvent(date, genBodyFn, timeNow, fleet.CalendarCreateEventOpts{})
	assert.ErrorAs(t, err, &fleet.DayEndedError{})

	// Workday already started
	date = time.Now().Add(48 * time.Hour)
	expectedStartTime = time.Date(date.Year(), date.Month(), date.Day(), endHour-1, 30, 0, 0, location)
	timeNow = func() time.Time {
		return expectedStartTime
	}
	event, err = gCal.createEvent(date, genBodyFn, timeNow, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, expectedStartTime.UTC(), event.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), event.EndTime.UTC())

	// Busy calendar
	date = time.Now().Add(48 * time.Hour)
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, location)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), endHour, 0, 0, 0, location)
	gEvents := &calendar.Events{}
	// Cancelled event
	gEvent := &calendar.Event{
		Id:     "cancelled-event-id",
		Start:  &calendar.EventDateTime{DateTime: dayStart.Format(time.RFC3339)},
		End:    &calendar.EventDateTime{DateTime: dayEnd.Format(time.RFC3339)},
		Status: "cancelled",
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	// All day events
	gEvent = &calendar.Event{
		Id:    "all-day-event-id",
		Start: &calendar.EventDateTime{Date: dayStart.Format(time.DateOnly)},
		End:   &calendar.EventDateTime{DateTime: dayEnd.Format(time.RFC3339)},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	gEvent = &calendar.Event{
		Id:    "all-day2-event-id",
		Start: &calendar.EventDateTime{DateTime: dayStart.Format(time.RFC3339)},
		End:   &calendar.EventDateTime{Date: dayEnd.Format(time.DateOnly)},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	// User-declined event
	gEvent = &calendar.Event{
		Id:        "user-declined-event-id",
		Start:     &calendar.EventDateTime{DateTime: dayStart.Format(time.RFC3339)},
		End:       &calendar.EventDateTime{DateTime: dayEnd.Format(time.RFC3339)},
		Attendees: []*calendar.EventAttendee{{Email: baseUserEmail, ResponseStatus: "declined"}},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	// Event before day
	gEvent = &calendar.Event{
		Id:    "before-event-id",
		Start: &calendar.EventDateTime{DateTime: dayStart.Add(-time.Hour).Format(time.RFC3339)},
		End:   &calendar.EventDateTime{DateTime: dayStart.Add(-30 * time.Minute).Format(time.RFC3339)},
	}
	gEvents.Items = append(gEvents.Items, gEvent)

	// Event from 6am to 11am
	eventStart := time.Date(date.Year(), date.Month(), date.Day(), 6, 0, 0, 0, location)
	eventEnd := time.Date(date.Year(), date.Month(), date.Day(), 11, 0, 0, 0, location)
	gEvent = &calendar.Event{
		Id:        "6-to-11-event-id",
		Start:     &calendar.EventDateTime{DateTime: eventStart.Format(time.RFC3339)},
		End:       &calendar.EventDateTime{DateTime: eventEnd.Format(time.RFC3339)},
		Attendees: []*calendar.EventAttendee{{Email: baseUserEmail, ResponseStatus: "accepted"}},
	}
	gEvents.Items = append(gEvents.Items, gEvent)

	// Event from 10am to 10:30am
	eventStart = time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, location)
	eventEnd = time.Date(date.Year(), date.Month(), date.Day(), 10, 30, 0, 0, location)
	gEvent = &calendar.Event{
		Id:        "10-to-10-30-event-id",
		Start:     &calendar.EventDateTime{DateTime: eventStart.Format(time.RFC3339)},
		End:       &calendar.EventDateTime{DateTime: eventEnd.Format(time.RFC3339)},
		Attendees: []*calendar.EventAttendee{{Email: "other@example.com", ResponseStatus: "accepted"}},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	// Event from 11am to 11:45am
	eventStart = time.Date(date.Year(), date.Month(), date.Day(), 11, 0, 0, 0, location)
	eventEnd = time.Date(date.Year(), date.Month(), date.Day(), 11, 45, 0, 0, location)
	gEvent = &calendar.Event{
		Id:        "11-to-11-45-event-id",
		Start:     &calendar.EventDateTime{DateTime: eventStart.Format(time.RFC3339)},
		End:       &calendar.EventDateTime{DateTime: eventEnd.Format(time.RFC3339)},
		Attendees: []*calendar.EventAttendee{{Email: "other@example.com", ResponseStatus: "accepted"}},
	}
	gEvents.Items = append(gEvents.Items, gEvent)

	// Event after day
	eventStart = time.Date(date.Year(), date.Month(), date.Day(), endHour, 0, 0, 0, location)
	eventEnd = time.Date(date.Year(), date.Month(), date.Day(), endHour, 45, 0, 0, location)
	gEvent = &calendar.Event{
		Id:        "after-event-id",
		Start:     &calendar.EventDateTime{DateTime: eventStart.Format(time.RFC3339)},
		End:       &calendar.EventDateTime{DateTime: eventEnd.Format(time.RFC3339)},
		Attendees: []*calendar.EventAttendee{{Email: "other@example.com", ResponseStatus: "accepted"}},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return gEvents, nil
	}
	expectedStartTime = time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, location)
	event, err = gCal.CreateEvent(date, genBodyFn, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, expectedStartTime.UTC(), event.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), event.EndTime.UTC())

	// Full schedule -- pick the last slot
	date = time.Now().Add(48 * time.Hour)
	dayStart = time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, location)
	dayEnd = time.Date(date.Year(), date.Month(), date.Day(), endHour, 0, 0, 0, location)
	gEvents = &calendar.Events{}
	gEvent = &calendar.Event{
		Id:    "9-to-5-event-id",
		Start: &calendar.EventDateTime{DateTime: dayStart.Format(time.RFC3339)},
		End:   &calendar.EventDateTime{DateTime: dayEnd.Format(time.RFC3339)},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return gEvents, nil
	}
	expectedStartTime = time.Date(date.Year(), date.Month(), date.Day(), endHour-1, 30, 0, 0, location)
	event, err = gCal.CreateEvent(date, genBodyConflictFn, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, expectedStartTime.UTC(), event.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), event.EndTime.UTC())

	// Almost full schedule -- pick the last slot
	date = time.Now().Add(48 * time.Hour)
	dayStart = time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, location)
	dayEnd = time.Date(date.Year(), date.Month(), date.Day(), endHour-1, 30, 0, 0, location)
	gEvents = &calendar.Events{}
	gEvent = &calendar.Event{
		Id:    "9-to-4-30-event-id",
		Start: &calendar.EventDateTime{DateTime: dayStart.Format(time.RFC3339)},
		End:   &calendar.EventDateTime{DateTime: dayEnd.Format(time.RFC3339)},
	}
	gEvents.Items = append(gEvents.Items, gEvent)
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return gEvents, nil
	}
	expectedStartTime = dayEnd
	event, err = gCal.CreateEvent(date, genBodyFn, fleet.CalendarCreateEventOpts{})
	require.NoError(t, err)
	assert.Equal(t, expectedStartTime.UTC(), event.StartTime.UTC())
	assert.Equal(t, expectedStartTime.Add(eventLength).UTC(), event.EndTime.UTC())

	// API error in ListEvents
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return nil, assert.AnError
	}
	_, err = gCal.CreateEvent(date, genBodyFn, fleet.CalendarCreateEventOpts{})
	assert.ErrorIs(t, err, assert.AnError)

	// API error in CreateEvent
	mockAPI.ListEventsFunc = func(timeMin, timeMax string) (*calendar.Events, error) {
		return &calendar.Events{}, nil
	}
	mockAPI.CreateEventFunc = func(event *calendar.Event) (*calendar.Event, error) {
		return nil, assert.AnError
	}
	_, err = gCal.CreateEvent(date, genBodyFn, fleet.CalendarCreateEventOpts{})
	assert.ErrorIs(t, err, assert.AnError)
}
