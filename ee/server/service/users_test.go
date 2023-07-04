package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestRolesChanged(t *testing.T) {
	for _, tc := range []struct {
		name string

		oldGlobal *string
		oldTeams  []fleet.UserTeam
		newGlobal *string
		newTeams  []fleet.UserTeam

		expectedRolesChanged bool
	}{
		{
			name:                 "no roles",
			expectedRolesChanged: false,
		},
		{
			name:                 "no-role-to-global-role",
			newGlobal:            ptr.String("admin"),
			expectedRolesChanged: true,
		},
		{
			name:                 "global-role-to-no-role",
			oldGlobal:            ptr.String("admin"),
			expectedRolesChanged: true,
		},
		{
			name:                 "global-role-unchanged",
			oldGlobal:            ptr.String("admin"),
			newGlobal:            ptr.String("admin"),
			expectedRolesChanged: false,
		},
		{
			name:                 "global-role-to-other-role",
			oldGlobal:            ptr.String("admin"),
			newGlobal:            ptr.String("maintainer"),
			expectedRolesChanged: true,
		},
		{
			name:      "global-role-to-team-role",
			oldGlobal: ptr.String("admin"),
			newTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "change-role-in-teams",
			oldTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "maintainer",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			newTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "remove-from-team",
			oldTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "maintainer",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			newTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "no-change-teams",
			oldTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			newTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: false,
		},
		{
			name: "added-to-teams",
			newTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "removed-from-a-team-and-added-to-another",
			oldTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: fleet.Team{ID: 3},
					Role: "observer",
				},
			},
			newTeams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: fleet.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedRolesChanged, rolesChanged(tc.oldGlobal, tc.oldTeams, tc.newGlobal, tc.newTeams))
		})
	}
}
