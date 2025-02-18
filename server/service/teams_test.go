package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamAuth(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return &fleet.Team{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{}, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return &fleet.Team{}, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return nil, nil
	}
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return nil, nil
	}
	ds.DeleteTeamFunc = func(ctx context.Context, tid uint) error {
		return nil
	}
	ds.TeamEnrollSecretsFunc = func(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
		return nil, nil
	}
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return []*fleet.Host{}, nil
	}
	ds.CleanupDiskEncryptionKeysOnTeamChangeFunc = func(ctx context.Context, hostIDs []uint, newTeamID *uint) error {
		return nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		switch name {
		case "team1":
			return &fleet.Team{ID: 1}, nil
		default:
			return &fleet.Team{ID: 2}, nil
		}
	}

	testCases := []struct {
		name                       string
		user                       *fleet.User
		shouldFailTeamWrite        bool
		shouldFailGlobalWrite      bool
		shouldFailRead             bool
		shouldFailTeamSecretsWrite bool
	}{
		{
			name:                       "global admin",
			user:                       &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailTeamWrite:        false,
			shouldFailGlobalWrite:      false,
			shouldFailRead:             false,
			shouldFailTeamSecretsWrite: false,
		},
		{
			name:                       "global maintainer",
			user:                       &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             false,
			shouldFailTeamSecretsWrite: false,
		},
		{
			name:                       "global observer",
			user:                       &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             false,
			shouldFailTeamSecretsWrite: true,
		},
		{
			name:                       "team admin, belongs to team",
			user:                       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailTeamWrite:        false,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             false,
			shouldFailTeamSecretsWrite: false,
		},
		{
			name:                       "team maintainer, belongs to team",
			user:                       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             false,
			shouldFailTeamSecretsWrite: false,
		},
		{
			name:                       "team observer, belongs to team",
			user:                       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             false,
			shouldFailTeamSecretsWrite: true,
		},
		{
			name:                       "team admin, DOES NOT belong to team",
			user:                       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             true,
			shouldFailTeamSecretsWrite: true,
		},
		{
			name:                       "team maintainer, DOES NOT belong to team",
			user:                       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             true,
			shouldFailTeamSecretsWrite: true,
		},
		{
			name:                       "team observer, DOES NOT belong to team",
			user:                       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			shouldFailTeamWrite:        true,
			shouldFailGlobalWrite:      true,
			shouldFailRead:             true,
			shouldFailTeamSecretsWrite: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.NewTeam(ctx, fleet.TeamPayload{Name: ptr.String("name")})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyTeam(ctx, 1, fleet.TeamPayload{Name: ptr.String("othername")})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.ModifyTeamAgentOptions(ctx, 1, nil, fleet.ApplySpecOptions{})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.AddTeamUsers(ctx, 1, []fleet.TeamUser{})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.DeleteTeamUsers(ctx, 1, []fleet.TeamUser{})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.ListTeamUsers(ctx, 1, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ListTeams(ctx, fleet.ListOptions{})
			checkAuthErr(t, false, err) // everybody can do this

			_, err = svc.GetTeam(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			err = svc.DeleteTeam(ctx, 1)
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.TeamEnrollSecrets(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyTeamEnrollSecrets(ctx, 1, []fleet.EnrollSecret{{Secret: "newteamsecret", CreatedAt: time.Now()}})
			checkAuthErr(t, tt.shouldFailTeamSecretsWrite, err)

			_, err = svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{{Name: "team1"}}, fleet.ApplyTeamSpecOptions{})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)
		})
	}
}

func TestApplyTeamSpecs(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	baseFeatures := fleet.Features{
		EnableHostUsers:         true,
		EnableSoftwareInventory: true,
		AdditionalQueries:       ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
	}

	mkspec := func(s string) *json.RawMessage {
		return ptr.RawMessage(json.RawMessage(s))
	}

	t.Run("Features for new teams", func(t *testing.T) {
		cases := []struct {
			name   string
			spec   *json.RawMessage
			global fleet.Features
			result fleet.Features
		}{
			{
				name:   "no spec features uses global config as defaults",
				spec:   nil,
				global: baseFeatures,
				result: baseFeatures,
			},
			{
				name:   "missing spec features uses new config default values",
				spec:   mkspec(`{"enable_software_inventory": false}`),
				global: baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         true,
					EnableSoftwareInventory: false,
					AdditionalQueries:       nil,
				},
			},
			{
				name:   "defaults can be overwritten",
				spec:   mkspec(`{"enable_host_users": false}`),
				global: baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         false,
					EnableSoftwareInventory: true,
					AdditionalQueries:       nil,
				},
			},
			{
				name: "all config can be changed",
				spec: mkspec(`{
          "enable_host_users": false,
          "enable_software_inventory": false,
          "additional_queries": {"example": "query"}
        }`),
				global: baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         false,
					EnableSoftwareInventory: false,
					AdditionalQueries:       ptr.RawMessage([]byte(`{"example": "query"}`)),
				},
			},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
					return nil, newNotFoundError()
				}

				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{Features: tt.global}, nil
				}

				ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
					require.Equal(t, "team1", team.Name)
					require.Equal(t, tt.result, team.Config.Features)
					team.ID = 1
					return team, nil
				}

				ds.NewActivityFunc = func(
					ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
				) error {
					act := activity.(fleet.ActivityTypeAppliedSpecTeam)
					require.Len(t, act.Teams, 1)
					return nil
				}

				_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{{Name: "team1", Features: tt.spec}}, fleet.ApplyTeamSpecOptions{})
				require.NoError(t, err)
			})
		}
	})

	t.Run("Features for existing teams", func(t *testing.T) {
		cases := []struct {
			name   string
			spec   *json.RawMessage
			old    fleet.Features
			result fleet.Features
		}{
			{
				name:   "no spec features uses old config",
				spec:   nil,
				old:    baseFeatures,
				result: baseFeatures,
			},
			{
				name: "missing spec features uses new config default values",
				spec: mkspec(`{"enable_software_inventory": false}`),
				old:  baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         true,
					EnableSoftwareInventory: false,
					AdditionalQueries:       nil,
				},
			},
			{
				name: "config has defaults based on what are the global defaults",
				spec: mkspec(`{"additional_queries": {}}`),
				old:  baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         true,
					EnableSoftwareInventory: true,
					AdditionalQueries:       nil,
				},
			},
			{
				name: "defaults can be overwritten",
				spec: mkspec(`{"enable_host_users": false}`),
				old:  baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         false,
					EnableSoftwareInventory: false,
					AdditionalQueries:       nil,
				},
			},
			{
				name: "all config can be changed",
				spec: mkspec(`{
          "enable_host_users": false,
          "enable_software_inventory": true,
          "additional_queries": {"example": "query"}
        }`),
				old: baseFeatures,
				result: fleet.Features{
					EnableHostUsers:         false,
					EnableSoftwareInventory: true,
					AdditionalQueries:       ptr.RawMessage([]byte(`{"example": "query"}`)),
				},
			},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
					return &fleet.Team{ID: 123, Config: fleet.TeamConfig{Features: tt.old}}, nil
				}

				ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
					return &fleet.Team{ID: 123}, nil
				}

				ds.NewActivityFunc = func(
					ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
				) error {
					act := activity.(fleet.ActivityTypeAppliedSpecTeam)
					require.Len(t, act.Teams, 1)
					return nil
				}

				idsByTeam, err := svc.ApplyTeamSpecs(
					ctx, []*fleet.TeamSpec{{Name: "team1", Features: tt.spec}}, fleet.ApplyTeamSpecOptions{},
				)
				require.NoError(t, err)
				require.Len(t, idsByTeam, 1)
				require.Equal(t, uint(123), idsByTeam["team1"])
			})
		}
	})
}

// Tests that a new enroll secret is created for new teams when none are provided
func TestApplyTeamSpecEnrollSecretForNewTeams(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, newNotFoundError()
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}

	t.Run("creates enroll secret when not included for a new team spec", func(t *testing.T) {
		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			require.Len(t, team.Secrets, 1)
			require.NotEmpty(t, team.Secrets[0])
			return &fleet.Team{ID: 1}, nil
		}

		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{{Name: "Foo"}}, fleet.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.True(t, ds.NewTeamFuncInvoked)
	})

	t.Run("does not create enroll secret when one is included for a new team spec", func(t *testing.T) {
		ds.NewTeamFuncInvoked = false
		enrollSecret := fleet.EnrollSecret{Secret: "test"}

		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			require.Len(t, team.Secrets, 1)
			require.Equal(t, enrollSecret.Secret, team.Secrets[0].Secret)
			return &fleet.Team{ID: 1}, nil
		}
		ds.NewTeamFuncInvoked = false

		// Dry run -- secret already used
		ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
			return false, nil
		}
		_, err := svc.ApplyTeamSpecs(
			ctx, []*fleet.TeamSpec{{Name: "Foo", Secrets: &[]fleet.EnrollSecret{enrollSecret}}},
			fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}},
		)
		assert.ErrorContains(t, err, "is already being used")

		// Normal dry run
		ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
			return true, nil
		}
		_, err = svc.ApplyTeamSpecs(
			ctx, []*fleet.TeamSpec{{Name: "Foo", Secrets: &[]fleet.EnrollSecret{enrollSecret}}},
			fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}},
		)
		assert.NoError(t, err)
		assert.False(t, ds.NewTeamFuncInvoked)

		_, err = svc.ApplyTeamSpecs(
			ctx, []*fleet.TeamSpec{{Name: "Foo", Secrets: &[]fleet.EnrollSecret{enrollSecret}}}, fleet.ApplyTeamSpecOptions{},
		)
		require.NoError(t, err)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.True(t, ds.NewTeamFuncInvoked)
	})

	ds.TeamByNameFuncInvoked = false
	ds.NewTeamFuncInvoked = false
}

// TestApplyTeamSpecsErrorInTeamByName tests that an error in ds.TeamByName will
// result in a proper error returned (instead of the authorization check missing error).
func TestApplyTeamSpecsErrorInTeamByName(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, errors.New("unknown error")
	}
	authzctx := &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, authzctx)
	_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{{Name: "Foo"}}, fleet.ApplyTeamSpecOptions{})
	require.Error(t, err)
	az, ok := authz_ctx.FromContext(ctx)
	require.True(t, ok)
	require.True(t, az.Checked())
}
