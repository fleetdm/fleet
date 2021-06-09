package test

import (
	"github.com/fleetdm/fleet/server/fleet"
	"github.com/fleetdm/fleet/server/ptr"
)

var (
	UserNoRoles = &fleet.User{
		ID:       1,
		Username: "no_roles",
	}
	UserAdmin = &fleet.User{
		ID:         2,
		GlobalRole: ptr.String(fleet.RoleAdmin),
		Username:   "global_admin",
	}
	UserMaintainer = &fleet.User{
		ID:         3,
		GlobalRole: ptr.String(fleet.RoleMaintainer),
		Username:   "global_maintainer",
	}
	UserObserver = &fleet.User{
		ID:         4,
		GlobalRole: ptr.String(fleet.RoleObserver),
		Username:   "global_observer",
	}
)
