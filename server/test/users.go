package test

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

var (
	UserNoRoles = &fleet.User{
		ID: 1,
	}
	UserAdmin = &fleet.User{
		ID:         2,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	UserMaintainer = &fleet.User{
		ID:         3,
		GlobalRole: ptr.String(fleet.RoleMaintainer),
	}
	UserObserver = &fleet.User{
		ID:         4,
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
)
