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
	GetEvent(e *fleet.CalendarEvent) (event *fleet.CalendarEvent, timeChanged bool, err error)
	CreateEvent(dayOfEvent time.Time, body string) (event *fleet.CalendarEvent, err error)
	DeleteEvent(event *fleet.CalendarEvent) error
}
