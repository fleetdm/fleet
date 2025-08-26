package service

import (
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUpdateTeamMDMDiskEncryption(t *testing.T) {
	testCases := []struct {
		name           string
		mdmConfig      fleet.TeamMDM
		diskEncryption *bool
		requireTPMPIN  *bool
		expectedError  string
	}{
		{
			name: "try to disable disk encryption with TPM PIN enabled",
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: true,
				RequireBitLockerPIN:  true,
			},
			diskEncryption: ptr.Bool(false),
			requireTPMPIN:  ptr.Bool(true),

			expectedError: fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg,
		},
		{
			name: "try to enable disk encryption with TPM PIN enabled",
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: false,
				RequireBitLockerPIN:  false,
			},
			diskEncryption: ptr.Bool(false),
			requireTPMPIN:  ptr.Bool(true),
			expectedError:  fleet.CantEnablePINRequiredIfDiskEncryptionEnabled,
		},
		{
			name: "try to disable disk encryption with TPM PIN enabled when disk encryption prev enabled",
			mdmConfig: fleet.TeamMDM{
				EnableDiskEncryption: true,
				RequireBitLockerPIN:  false,
			},
			diskEncryption: ptr.Bool(false),
			requireTPMPIN:  ptr.Bool(true),
			expectedError:  fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg,
		},
	}

	ds := new(mock.Store)

	svc := &Service{
		ds: ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: "something",
			},
		},
	}

	ctx := context.Background()

	for _, tC := range testCases {
		team := fleet.Team{
			Config: fleet.TeamConfig{
				MDM: tC.mdmConfig,
			},
		}

		err := svc.updateTeamMDMDiskEncryption(
			ctx,
			&team,
			tC.diskEncryption,
			tC.requireTPMPIN,
		)

		if tC.expectedError != "" {
			require.NotNil(t, err)
			require.True(
				t,
				strings.Contains(err.Error(), tC.expectedError),
				"Expected '%s' to contain '%s'",
				err.Error(), tC.expectedError)
		}
	}
}

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
