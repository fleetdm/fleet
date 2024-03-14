package calendar

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"time"
)

type GoogleCalendarConfig struct {
	Context           context.Context
	IntegrationConfig *fleet.GoogleCalendarIntegration
	UserEmail         string
	Logger            log.Logger
	// Should be nil for production
	API GoogleCalendarAPI
}

type Calendar interface {
	// Connect to calendar. This method must be called first. Currently, config must be a *GoogleCalendarConfig
	Connect(config any) (Calendar, error)
	// GetAndUpdateEvent retrieves the event with the given ID. If the event has been deleted, it schedules a new event and returns the new event.
	GetAndUpdateEvent(event *fleet.CalendarEvent, genBodyFn func() string) (updatedEvent *fleet.CalendarEvent, updated bool, err error)
	// CreateEvent creates a new event on the calendar on the given date.
	CreateEvent(dateOfEvent time.Time, body string) (event *fleet.CalendarEvent, err error)
	// DeleteEvent deletes the event with the given ID.
	DeleteEvent(event *fleet.CalendarEvent) error
}
