package calendar

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"time"
)

type Event struct {
}

type Config struct {
	Context           context.Context
	IntegrationConfig *fleet.GoogleCalendarIntegration
	UserEmail         string
	Logger            log.Logger
}

type Calendar interface {
	GetEvent(e *Event) (event *Event, timeChanged bool, err error)
	CreateEvent(dayInUsersTimezone time.Time, body string) (event *Event, err error)
	DeleteEvent(event *Event) error
}
