package service

import (
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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

type bootstrapNotFoundError struct {
	msg string
}

func (e *bootstrapNotFoundError) Error() string {
	return e.msg
}

func (e *bootstrapNotFoundError) IsNotFound() bool {
	return true
}

func TestUpdateTeamMDMAppleSetupManualAgent(t *testing.T) {
	cases := []struct {
		Name            string
		Count           fleet.SetupExperienceCount
		Error           string
		MacOSSetup      fleet.MacOSSetup
		MDMSetupPayload fleet.MDMAppleSetupPayload
	}{
		{
			Name: "good case",
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
		},
		{
			Name: "no bootstrap package",
			Count: fleet.SetupExperienceCount{
				Installers: 0,
				VPP:        0,
				Scripts:    0,
			},
			Error: "bootstrap_package",
		},
		{
			Name: "installers exist",
			Count: fleet.SetupExperienceCount{
				Installers: 1,
				VPP:        0,
				Scripts:    0,
			},
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
			Error: "disable setup experience software",
		},
		{
			Name: "vpp apps exist",
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
			Count: fleet.SetupExperienceCount{
				VPP: 1,
			},
			Error: "disable setup experience software",
		},
		{
			Name: "script exists",
			Count: fleet.SetupExperienceCount{
				Scripts: 1,
			},
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage: optjson.SetString("package"),
			},
			Error: "remove your setup experience script",
		},
	}

	ds := new(mock.Store)

	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		return nil
	}

	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return &fleet.Team{}, nil
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	svc := &Service{
		ds: ds,
		config: config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: "something",
			},
		},
		authz: authorizer,
	}

	// Add admin user to context
	adminUser := &fleet.User{
		ID:         2,
		GlobalRole: ptr.String(fleet.RoleAdmin),
		Email:      "useradmin@example.com",
	}
	ctx := test.UserContext(context.Background(), adminUser)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ds.GetMDMAppleBootstrapPackageMetaFunc = func(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
				if tc.MacOSSetup.BootstrapPackage.Value == "" {
					return nil, &bootstrapNotFoundError{msg: "bootstrap package not found"}
				}
				return &fleet.MDMAppleBootstrapPackage{
					Name: tc.MacOSSetup.BootstrapPackage.Value,
				}, nil
			}

			ds.GetSetupExperienceCountFunc = func(ctx context.Context, platform string, teamID *uint) (*fleet.SetupExperienceCount, error) {
				return &tc.Count, nil
			}

			tm := &fleet.Team{}
			tm.Config.MDM.MacOSSetup = tc.MacOSSetup

			payload := fleet.MDMAppleSetupPayload{
				ManualAgentInstall: ptr.Bool(true),
			}

			err := svc.updateTeamMDMAppleSetup(ctx, tm, payload)
			if tc.Error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.Error)
			}
		})
		t.Run(tc.Name+" no team", func(t *testing.T) {
			ds.GetSetupExperienceCountFunc = func(ctx context.Context, platform string, teamID *uint) (*fleet.SetupExperienceCount, error) {
				return &tc.Count, nil
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				appConfig := &fleet.AppConfig{}
				appConfig.MDM.MacOSSetup = tc.MacOSSetup
				return appConfig, nil
			}

			tm := &fleet.Team{}
			tm.Config.MDM.MacOSSetup = tc.MacOSSetup

			payload := fleet.MDMAppleSetupPayload{
				ManualAgentInstall: ptr.Bool(true),
			}

			err := svc.updateAppConfigMDMAppleSetup(ctx, payload)
			if tc.Error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.Error)
			}
		})

	}
}
