package calendar

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	kitlog "github.com/go-kit/log"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

type GoogleCalendarMockAPI struct {
	logger kitlog.Logger
}

var (
	mockEvents = make(map[string]*calendar.Event)
	mu         sync.Mutex
	id         uint64
)

const latency = 500 * time.Millisecond

// Configure creates a new Google Calendar service using the provided credentials.
func (lowLevelAPI *GoogleCalendarMockAPI) Configure(_ context.Context, _ string, _ string, userToImpersonate string) error {
	if lowLevelAPI.logger == nil {
		lowLevelAPI.logger = kitlog.With(kitlog.NewLogfmtLogger(os.Stderr), "mock", "GoogleCalendarMockAPI", "user", userToImpersonate)
	}
	return nil
}

func (lowLevelAPI *GoogleCalendarMockAPI) GetSetting(name string) (*calendar.Setting, error) {
	time.Sleep(latency)
	lowLevelAPI.logger.Log("msg", "GetSetting", "name", name)
	if name == "timezone" {
		return &calendar.Setting{
			Id:    "timezone",
			Value: "America/Chicago",
		}, nil
	}
	return nil, errors.New("setting not supported")
}

func (lowLevelAPI *GoogleCalendarMockAPI) CreateEvent(event *calendar.Event) (*calendar.Event, error) {
	time.Sleep(latency)
	mu.Lock()
	defer mu.Unlock()
	id += 1
	event.Id = strconv.FormatUint(id, 10)
	lowLevelAPI.logger.Log("msg", "CreateEvent", "id", event.Id, "start", event.Start.DateTime)
	mockEvents[event.Id] = event
	return event, nil
}

func (lowLevelAPI *GoogleCalendarMockAPI) GetEvent(id, _ string) (*calendar.Event, error) {
	time.Sleep(latency)
	mu.Lock()
	defer mu.Unlock()
	event, ok := mockEvents[id]
	if !ok {
		return nil, &googleapi.Error{Code: http.StatusNotFound}
	}
	lowLevelAPI.logger.Log("msg", "GetEvent", "id", id, "start", event.Start.DateTime)
	return event, nil
}

func (lowLevelAPI *GoogleCalendarMockAPI) ListEvents(string, string) (*calendar.Events, error) {
	time.Sleep(latency)
	lowLevelAPI.logger.Log("msg", "ListEvents")
	return &calendar.Events{}, nil
}

func (lowLevelAPI *GoogleCalendarMockAPI) DeleteEvent(id string) error {
	time.Sleep(latency)
	mu.Lock()
	defer mu.Unlock()
	lowLevelAPI.logger.Log("msg", "DeleteEvent", "id", id)
	delete(mockEvents, id)
	return nil
}

func ListGoogleMockEvents() map[string]*calendar.Event {
	return mockEvents
}

func ClearMockEvents() {
	mu.Lock()
	defer mu.Unlock()
	mockEvents = make(map[string]*calendar.Event)
}

func SetMockEventsToNow() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	for _, mockEvent := range mockEvents {
		mockEvent.Start = &calendar.EventDateTime{DateTime: now.Format(time.RFC3339)}
		mockEvent.End = &calendar.EventDateTime{DateTime: now.Add(30 * time.Minute).Format(time.RFC3339)}
	}
}
