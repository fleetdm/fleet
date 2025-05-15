package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestObfuscateSecrets(t *testing.T) {
	buildTeams := func(n int) []*fleet.Team {
		r := make([]*fleet.Team, 0, n)
		for i := 1; i <= n; i++ {
			r = append(r, &fleet.Team{
				ID: uint(i), //nolint:gosec // dismiss G115
				Secrets: []*fleet.EnrollSecret{
					{Secret: "abc"},
					{Secret: "123"},
				},
			})
		}
		return r
	}

	t.Run("no user", func(t *testing.T) {
		err := obfuscateSecrets(nil, nil)
		require.Error(t, err)
	})

	t.Run("no teams", func(t *testing.T) {
		user := fleet.User{}
		err := obfuscateSecrets(&user, nil)
		require.NoError(t, err)
	})

	t.Run("user is not a global observer", func(t *testing.T) {
		user := fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
		teams := buildTeams(3)

		err := obfuscateSecrets(&user, teams)
		require.NoError(t, err)

		for _, team := range teams {
			for _, s := range team.Secrets {
				require.NotEqual(t, fleet.MaskedPassword, s.Secret)
			}
		}
	})

	t.Run("user is global observer", func(t *testing.T) {
		roles := []*string{ptr.String(fleet.RoleObserver), ptr.String(fleet.RoleObserverPlus)}
		for _, r := range roles {
			user := &fleet.User{GlobalRole: r}
			teams := buildTeams(3)

			err := obfuscateSecrets(user, teams)
			require.NoError(t, err)

			for _, team := range teams {
				for _, s := range team.Secrets {
					require.Equal(t, fleet.MaskedPassword, s.Secret)
				}
			}
		}
	})

	t.Run("user is observer in some teams", func(t *testing.T) {
		teams := buildTeams(4)

		// Make user an observer in the 'even' teams
		user := &fleet.User{Teams: []fleet.UserTeam{
			{
				Team: *teams[1],
				Role: fleet.RoleObserver,
			},
			{
				Team: *teams[2],
				Role: fleet.RoleAdmin,
			},
			{
				Team: *teams[3],
				Role: fleet.RoleObserverPlus,
			},
		}}

		err := obfuscateSecrets(user, teams)
		require.NoError(t, err)

		for i, team := range teams {
			for _, s := range team.Secrets {
				require.Equal(t, fleet.MaskedPassword == s.Secret, i == 0 || i == 1 || i == 3)
			}
		}
	})
}
