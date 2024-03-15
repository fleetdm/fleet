package calendar

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	baseServiceEmail = "service@example.com"
	basePrivateKey   = "private-key"
	baseUserEmail    = "user@example.com"
)

var (
	baseCtx = context.Background()
	logger  = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
)

type MockGoogleCalendarLowLevelAPI struct {
	ConfigureFunc   func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string) error
	GetSettingFunc  func(name string) (*calendar.Setting, error)
	ListEventsFunc  func(timeMin, timeMax string) (*calendar.Events, error)
	CreateEventFunc func(event *calendar.Event) (*calendar.Event, error)
	GetEventFunc    func(id, eTag string) (*calendar.Event, error)
	DeleteEventFunc func(id string) error
}

func (m *MockGoogleCalendarLowLevelAPI) Configure(
	ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string,
) error {
	return m.ConfigureFunc(ctx, serviceAccountEmail, privateKey, userToImpersonateEmail)
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

func (m *MockGoogleCalendarLowLevelAPI) GetEvent(id, eTag string) (*calendar.Event, error) {
	return m.GetEventFunc(id, eTag)
}

func (m *MockGoogleCalendarLowLevelAPI) DeleteEvent(id string) error {
	return m.DeleteEventFunc(id)
}

func TestGoogleCalendar_Configure(t *testing.T) {
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string) error {
		assert.Equal(t, baseCtx, ctx)
		assert.Equal(t, baseServiceEmail, serviceAccountEmail)
		assert.Equal(t, basePrivateKey, privateKey)
		assert.Equal(t, baseUserEmail, userToImpersonateEmail)
		return nil
	}

	// Happy path test
	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(baseUserEmail)
	assert.NoError(t, err)

	// Configure error test
	mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string) error {
		return assert.AnError
	}
	err = cal.Configure(baseUserEmail)
	assert.ErrorIs(t, err, assert.AnError)
}

func makeConfig(mockAPI *MockGoogleCalendarLowLevelAPI) *GoogleCalendarConfig {
	if mockAPI != nil && mockAPI.ConfigureFunc == nil {
		mockAPI.ConfigureFunc = func(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string) error {
			return nil
		}
	}
	config := &GoogleCalendarConfig{
		Context: context.Background(),
		IntegrationConfig: &fleet.GoogleCalendarIntegration{
			Email:      baseServiceEmail,
			PrivateKey: basePrivateKey,
		},
		Logger: logger,
		API:    mockAPI,
	}
	return config
}

func TestGoogleCalendar_DeleteEvent(t *testing.T) {
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
}

func TestGoogleCalendar_unmarshalDetails(t *testing.T) {
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
	mockAPI := &MockGoogleCalendarLowLevelAPI{}
	const baseETag = "event-eTag"
	const baseEventID = "event-id"
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		assert.Equal(t, baseEventID, id)
		assert.Equal(t, baseETag, eTag)
		return &calendar.Event{
			Etag: baseETag, // ETag matches -- no modifications to event
		}, nil
	}
	genBodyFn := func() string {
		t.Error("genBodyFn should not be called")
		return "event-body"
	}
	var cal fleet.UserCalendar = NewGoogleCalendar(makeConfig(mockAPI))
	err := cal.Configure(baseUserEmail)
	assert.NoError(t, err)

	event := &fleet.CalendarEvent{
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
		Data:      []byte(`{"ID":"` + baseEventID + `","ETag":"` + baseETag + `"}`),
	}

	// ETag matches
	retrievedEvent, updated, err := cal.GetAndUpdateEvent(event, genBodyFn)
	assert.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, event, retrievedEvent)

	// http.StatusNotModified response (ETag matches)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return nil, &googleapi.Error{Code: http.StatusNotModified}
	}
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn)
	assert.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, event, retrievedEvent)

	// Cannot unmarshal details
	eventBadDetails := &fleet.CalendarEvent{
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
		Data:      []byte(`{"bozo`),
	}
	_, _, err = cal.GetAndUpdateEvent(eventBadDetails, genBodyFn)
	assert.Error(t, err)

	// API error test
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return nil, assert.AnError
	}
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn)
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
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.NotEqual(t, event, retrievedEvent)
	require.NotNil(t, retrievedEvent)
	assert.Equal(t, startTime, retrievedEvent.StartTime)
	assert.Equal(t, endTime, retrievedEvent.EndTime)
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
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn)
	assert.Error(t, err)

	// missing start time
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:   baseEventID,
			Etag: "new-eTag",
			End:  &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)},
		}, nil
	}
	_, _, err = cal.GetAndUpdateEvent(event, genBodyFn)
	assert.Error(t, err)

	// Event has been modified, with custom timezone.
	tzId := "Africa/Kinshasa"
	location, _ := time.LoadLocation(tzId)
	startTime = time.Now().Add(time.Minute).Truncate(time.Second).In(location)
	endTime = time.Now().Add(time.Hour).Truncate(time.Second).In(location)
	mockAPI.GetEventFunc = func(id, eTag string) (*calendar.Event, error) {
		return &calendar.Event{
			Id:    baseEventID,
			Etag:  "new-eTag",
			Start: &calendar.EventDateTime{DateTime: startTime.UTC().Format(time.RFC3339), TimeZone: tzId},
			End:   &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339), TimeZone: tzId},
		}, nil
	}
	retrievedEvent, updated, err = cal.GetAndUpdateEvent(event, genBodyFn)
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.NotEqual(t, event, retrievedEvent)
	require.NotNil(t, retrievedEvent)
	assert.Equal(t, startTime.UTC(), retrievedEvent.StartTime.UTC())
	assert.Equal(t, endTime.UTC(), retrievedEvent.EndTime.UTC())
	assert.Equal(t, baseUserEmail, retrievedEvent.Email)

	// TODO: 404 response (deleted)

	// TODO: cancelled (deleted)

	// TODO: all day event (deleted)

	// TODO: moved in the past event (deleted)

}

//CreateEvent(dateOfEvent time.Time, body string) (event *CalendarEvent, err error)
