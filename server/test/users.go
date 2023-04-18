package test

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

var (
	GoodPassword  = "password123#"
	GoodPassword2 = "password123!"
	UserNoRoles   = &fleet.User{
		ID: 1,
	}
	UserAdmin = &fleet.User{
		ID:         2,
		GlobalRole: ptr.String(fleet.RoleAdmin),
		Email:      "useradmin@example.com",
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
	UserObserverPlus = &fleet.User{
		ID:         12,
		GlobalRole: ptr.String(fleet.RoleObserverPlus),
	}
	UserTeamObserverPlusTeam1 = &fleet.User{
		ID: 13,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleObserverPlus,
			},
		},
	}
	UserTeamObserverPlusTeam2 = &fleet.User{
		ID: 14,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleObserverPlus,
			},
		},
	}
	UserGitOps = &fleet.User{
		ID:         15,
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	UserTeamGitOpsTeam1 = &fleet.User{
		ID: 16,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleGitOps,
			},
		},
	}
	UserTeamGitOpsTeam2 = &fleet.User{
		ID: 17,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleGitOps,
			},
		},
	}
)
