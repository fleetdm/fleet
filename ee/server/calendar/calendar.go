package calendar

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
)

type GoogleCalendarConfig struct {
	Context           context.Context
	IntegrationConfig *fleet.GoogleCalendarIntegration
	UserEmail         string
	Logger            log.Logger
	// Should be nil for production
	API GoogleCalendarAPI
}
