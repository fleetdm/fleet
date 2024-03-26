package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/WatchBeam/clock"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func setupMockDatastorePremiumService() (*mock.Store, *eeservice.Service, context.Context) {
	ds := new(mock.Store)
	lic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	ctx := license.NewContext(context.Background(), lic)

	logger := kitlog.NewNopLogger()
	fleetConfig := config.FleetConfig{
		MDM: config.MDMConfig{
			AppleSCEPCertBytes: eeservice.TestCert,
			AppleSCEPKeyBytes:  eeservice.TestKey,
		},
	}
	var depStorage nanodep_storage.AllDEPStorage = &nanodep_mock.Storage{}

	freeSvc, err := service.NewService(
		ctx,
		ds,
		nil,
		nil,
		logger,
		nil,
		fleetConfig,
		nil,
		clock.C,
		nil,
		nil,
		ds,
		nil,
		nil,
		&fleet.NoOpGeoIP{},
		nil,
		depStorage,
		nil,
		nil,
		"",
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}
	svc, err := eeservice.NewService(
		freeSvc,
		ds,
		logger,
		fleetConfig,
		nil,
		clock.C,
		depStorage,
		nil,
		"",
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}
	return ds, svc, ctx
}

func TestGetOrCreatePreassignTeam(t *testing.T) {
	ds, svc, ctx := setupMockDatastorePremiumService()

	ssoSettings := fleet.SSOProviderSettings{
		EntityID:    "foo",
		MetadataURL: "https://example.com/metadata.xml",
		IssuerURI:   "https://example.com",
	}
	appConfig := &fleet.AppConfig{MDM: fleet.MDM{
		EnabledAndConfigured:  true,
		EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: ssoSettings},
		MacOSSetup: fleet.MacOSSetup{
			BootstrapPackage:            optjson.SetString("https://example.com/bootstrap.pkg"),
			EnableEndUserAuthentication: true,
			EnableReleaseDeviceManually: optjson.SetBool(true),
		},
	}}
	preassignGroups := []string{"one", "three"}

	// initialize team store with team one and two already created, one matches
	// preassign group [0], two does not match any preassign group
	team1 := &fleet.Team{
		ID:   1,
		Name: preassignGroups[0],
	}
	team2 := &fleet.Team{
		ID:   2,
		Name: "two",
		Config: fleet.TeamConfig{
			MDM: fleet.TeamMDM{
				MacOSSetup: fleet.MacOSSetup{MacOSSetupAssistant: optjson.SetString("foo/bar")},
			},
		},
	}
	teamStore := map[uint]*fleet.Team{1: team1, 2: team2}

	resetInvoked := func() {
		ds.TeamByNameFuncInvoked = false
		ds.NewTeamFuncInvoked = false
		ds.SaveTeamFuncInvoked = false
		ds.NewMDMAppleConfigProfileFuncInvoked = false
		ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked = false
		ds.AppConfigFuncInvoked = false
		ds.NewJobFuncInvoked = false
		ds.GetMDMAppleSetupAssistantFuncInvoked = false
		ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked = false
	}
	setupDS := func(t *testing.T) {
		resetInvoked()

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appConfig, nil
		}
		ds.NewActivityFunc = func(ctx context.Context, u *fleet.User, a fleet.ActivityDetails) error {
			return nil
		}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			for _, team := range teamStore {
				if team.Name == name {
					return team, nil
				}
			}
			return nil, ctxerr.Wrap(ctx, &eeservice.NotFoundError{})
		}
		ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
			tm, ok := teamStore[id]
			if !ok {
				return nil, errors.New("team id not found")
			}
			if id != tm.ID {
				// sanity chec
				return nil, errors.New("team id mismatch")
			}
			return tm, nil
		}
		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			return nil, errors.New("not implemented")
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			return nil, errors.New("not implemented")
		}
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, profile fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
			return nil, errors.New("not implemented")
		}
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
			return errors.New("not implemented")
		}
		ds.CopyDefaultMDMAppleBootstrapPackageFunc = func(ctx context.Context, ac *fleet.AppConfig, toTeamID uint) error {
			return errors.New("not implemented")
		}
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, errors.New("not implemented")
		}
		ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
			return nil, errors.New("not implemented")
		}
	}

	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, authzCtx)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: test.UserAdmin})
	actx, _ := authz_ctx.FromContext(ctx)
	actx.SetChecked()

	t.Run("get preassign team", func(t *testing.T) {
		setupDS(t)

		// preasign group corresponds to existing team so simply get it
		team, err := svc.GetOrCreatePreassignTeam(ctx, preassignGroups[0:1])
		require.NoError(t, err)
		require.Equal(t, uint(1), team.ID)
		require.Equal(t, preassignGroups[0], team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("create preassign team", func(t *testing.T) {
		setupDS(t)

		lastTeamID := uint(0)
		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1)
			_, ok := teamStore[id]
			require.False(t, ok) // sanity check
			team.ID = id
			teamStore[id] = team
			lastTeamID = id
			return team, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)     // sanity check
			require.Equal(t, tm.Name, team.Name) // sanity check
			// NOTE: BootstrapPackage gets set by CopyDefaultMDMAppleBootstrapPackage
			// require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)             // set to default
			require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant, team.Config.MDM.MacOSSetup.MacOSSetupAssistant)                             // set to default
			require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // set to default
			teamStore[tm.ID] = team
			return team, nil
		}
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, profile fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
			require.Equal(t, lastTeamID, *profile.TeamID)
			require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, profile.Identifier)
			return &profile, nil
		}
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
			require.Equal(t, lastTeamID, *teamID)
			require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, profileIdentifier)
			return nil
		}
		ds.CopyDefaultMDMAppleBootstrapPackageFunc = func(ctx context.Context, ac *fleet.AppConfig, toTeamID uint) error {
			require.Equal(t, lastTeamID, toTeamID)
			require.NotNil(t, ac)
			require.Equal(t, "https://example.com/bootstrap.pkg", ac.MDM.MacOSSetup.BootstrapPackage.Value)
			teamStore[toTeamID].Config.MDM.MacOSSetup.BootstrapPackage = optjson.SetString(ac.MDM.MacOSSetup.BootstrapPackage.Value)
			return nil
		}
		var jobTask string
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			// first task is UpdateProfile, next is ProfileChanged (when setup assistant is set)
			if jobTask == "" {
				jobTask = string(worker.MacosSetupAssistantUpdateProfile)
			} else {
				jobTask = string(worker.MacosSetupAssistantProfileChanged)
			}
			wantArgs, err := json.Marshal(map[string]interface{}{
				"task":    jobTask,
				"team_id": lastTeamID,
			})
			require.NoError(t, err)
			wantJob := &fleet.Job{
				Name:  "macos_setup_assistant",
				Args:  (*json.RawMessage)(&wantArgs),
				State: fleet.JobStateQueued,
			}
			require.Equal(t, wantJob.Name, job.Name)
			require.Equal(t, string(*wantJob.Args), string(*job.Args))
			require.Equal(t, wantJob.State, job.State)
			return job, nil
		}
		setupAsstByTeam := make(map[uint]*fleet.MDMAppleSetupAssistant)
		globalSetupAsst := &fleet.MDMAppleSetupAssistant{
			ID:          15,
			TeamID:      nil,
			Name:        "test asst",
			Profile:     json.RawMessage(`{"foo": "bar"}`),
			ProfileUUID: "abc-def",
		}
		setupAsstByTeam[0] = globalSetupAsst
		ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
			var tmID uint
			if teamID != nil {
				tmID = *teamID
			}
			asst := setupAsstByTeam[tmID]
			if asst == nil {
				return nil, eeservice.NotFoundError{}
			}
			return asst, nil
		}
		ds.SetOrUpdateMDMAppleSetupAssistantFunc = func(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
			require.Equal(t, globalSetupAsst.Name, asst.Name)
			require.JSONEq(t, string(globalSetupAsst.Profile), string(asst.Profile))
			require.NotNil(t, asst.TeamID)
			require.EqualValues(t, lastTeamID, *asst.TeamID)
			setupAsstByTeam[*asst.TeamID] = asst
			return asst, nil
		}

		// new team ("one - three") is created with bootstrap package and end user auth based on app config
		team, err := svc.GetOrCreatePreassignTeam(ctx, preassignGroups)
		require.NoError(t, err)
		require.Equal(t, uint(3), team.ID)
		require.Equal(t, eeservice.TeamNameFromPreassignGroups(preassignGroups), team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.True(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.GetMDMAppleSetupAssistantFuncInvoked)
		require.True(t, ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked)
		require.NotEmpty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant.Value, team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
		require.True(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.True(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.True(t, ds.NewJobFuncInvoked)
		resetInvoked()

		jobTask = ""

		// when called again, simply get the previously created team
		team, err = svc.GetOrCreatePreassignTeam(ctx, preassignGroups)
		require.NoError(t, err)
		require.Equal(t, uint(3), team.ID)
		require.Equal(t, eeservice.TeamNameFromPreassignGroups(preassignGroups), team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		require.False(t, ds.GetMDMAppleSetupAssistantFuncInvoked)
		require.False(t, ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked)
		require.NotEmpty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant.Value, team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
		require.True(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.True(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		resetInvoked()

		jobTask = ""

		// when a custom setup assistant is not set for "no team", we don't create
		// a custom setup assistant
		setupAsstByTeam[0] = nil

		preassignGrousWithFoo := append(preassignGroups, "foo")
		team, err = svc.GetOrCreatePreassignTeam(ctx, preassignGrousWithFoo)
		require.NoError(t, err)
		require.Equal(t, uint(4), team.ID)
		require.Equal(t, eeservice.TeamNameFromPreassignGroups(preassignGrousWithFoo), team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.True(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.GetMDMAppleSetupAssistantFuncInvoked)
		require.False(t, ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked)
		require.NotEmpty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.True(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.True(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		resetInvoked()
	})

	t.Run("modify team via apply team spec", func(t *testing.T) {
		setupDS(t)

		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)                                               // sanity check
			require.Equal(t, tm.Name, team.Name)                                           // sanity check
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not modified
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not modified
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not modified
			return teamStore[tm.ID], nil
		}

		// apply team spec does not apply defaults
		spec := &fleet.TeamSpec{
			Name: team2.Name,
		}
		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplySpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("new team", func(t *testing.T) {
		setupDS(t)

		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1)
			_, ok := teamStore[id]
			require.False(t, ok) // sanity check
			require.Equal(t, "new team", team.Name)
			require.Equal(t, "new description", team.Description)
			require.False(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not set
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not set
			team.ID = id
			teamStore[id] = team
			return team, nil
		}

		// new team does not apply defaults
		_, err := svc.NewTeam(ctx, fleet.TeamPayload{
			Name:        ptr.String("new team"),
			Description: ptr.String("new description"),
		})
		require.NoError(t, err)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("apply team spec", func(t *testing.T) {
		setupDS(t)

		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1)
			_, ok := teamStore[id]
			require.False(t, ok)                                                           // sanity check
			require.Equal(t, "new team spec", team.Name)                                   // set
			require.Equal(t, "12.0", team.Config.MDM.MacOSUpdates.MinimumVersion.Value)    // set
			require.Equal(t, "2024-01-01", team.Config.MDM.MacOSUpdates.Deadline.Value)    // set
			require.False(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not set
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not set
			team.ID = id
			teamStore[id] = team
			return team, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)                                               // sanity check
			require.Equal(t, tm.Name, team.Name)                                           // sanity check
			require.Equal(t, "12.0", team.Config.MDM.MacOSUpdates.MinimumVersion.Value)    // unchanged
			require.Equal(t, "2025-01-01", team.Config.MDM.MacOSUpdates.Deadline.Value)    // modified
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not set
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not set

			return teamStore[tm.ID], nil
		}

		spec := &fleet.TeamSpec{
			Name: "new team spec",
			MDM: fleet.TeamSpecMDM{
				MacOSUpdates: fleet.MacOSUpdates{
					MinimumVersion: optjson.SetString("12.0"),
					Deadline:       optjson.SetString("2024-01-01"),
				},
			},
		}

		// apply team spec creates new team without defaults
		_, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplySpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()

		// apply team spec edits existing team without applying defaults
		spec.MDM.MacOSUpdates.Deadline = optjson.SetString("2025-01-01")
		_, err = svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplySpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})
}
