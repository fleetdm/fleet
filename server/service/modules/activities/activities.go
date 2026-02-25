package activities

import (
	"context"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ActivityModule is a thin facade that translates fleet types to bounded context
// types and delegates to the activity bounded context service.
type ActivityModule interface {
	NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error
}

// Module is the concrete implementation of ActivityModule. It holds a reference
// to the activity bounded context service, which must be set via SetService
// before any calls to NewActivity.
type Module struct {
	svc activity_api.NewActivityService
}

// NewActivityModule creates a new activity module. The bounded context service
// must be set via SetService before use.
func NewActivityModule() *Module {
	return &Module{}
}

// SetService sets the activity bounded context service.
func (m *Module) SetService(svc activity_api.NewActivityService) {
	m.svc = svc
}

func (m *Module) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	var apiUser *activity_api.User
	if user != nil {
		apiUser = &activity_api.User{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Deleted: user.Deleted,
		}
	}
	return m.svc.NewActivity(ctx, apiUser, activity)
}
