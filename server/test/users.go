package test

import (
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/ptr"
)

var (
	UserNoRoles = &kolide.User{
		ID:       1,
		Username: "no_roles",
	}
	UserAdmin = &kolide.User{
		ID:         2,
		GlobalRole: ptr.String(kolide.RoleAdmin),
		Username:   "global_admin",
	}
	UserMaintainer = &kolide.User{
		ID:         3,
		GlobalRole: ptr.String(kolide.RoleMaintainer),
		Username:   "global_maintainer",
	}
	UserObserver = &kolide.User{
		ID:         4,
		GlobalRole: ptr.String(kolide.RoleObserver),
		Username:   "global_observer",
	}
)
