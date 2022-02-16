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
	UserTeamAdminTeam1 = &fleet.User{
		ID: 5,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleAdmin,
			},
		},
	}
	UserTeamAdminTeam2 = &fleet.User{
		ID: 6,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleAdmin,
			},
		},
	}
	UserTeamMaintainerTeam1 = &fleet.User{
		ID: 7,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleMaintainer,
			},
		},
	}
	UserTeamMaintainerTeam2 = &fleet.User{
		ID: 8,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleMaintainer,
			},
		},
	}
	UserTeamObserverTeam1 = &fleet.User{
		ID: 9,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleObserver,
			},
		},
	}
	UserTeamObserverTeam2 = &fleet.User{
		ID: 10,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleObserver,
			},
		},
	}
	UserTeamObserverTeam1TeamAdminTeam2 = &fleet.User{
		ID: 11,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleObserver,
			},
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleAdmin,
			},
		},
	}
)
