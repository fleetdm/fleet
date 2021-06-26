package test

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// UserContext returns a new context with the provided user as the viewer.
func UserContext(user *fleet.User) context.Context {
	return viewer.NewContext(context.Background(), viewer.Viewer{User: user})
}
