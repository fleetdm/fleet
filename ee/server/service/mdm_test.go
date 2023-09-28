package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*mock.Store, *Service) {
	ds := new(mock.Store)
	svc := &Service{
		ds: ds,
		config: config.FleetConfig{
			MDM: config.MDMConfig{
				AppleSCEPCertBytes: testCert,
				AppleSCEPKeyBytes:  testKey,
			},
		},
	}
	return ds, svc
}

func TestMDMAppleEnableFileVaultAndEscrow(t *testing.T) {
	ctx := context.Background()

	t.Run("fails if SCEP is not configured", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds, config: config.FleetConfig{}}
		err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, nil)
		require.Error(t, err)
	})

	t.Run("fails if the profile can't be saved in the db", func(t *testing.T) {
		ds, svc := setup(t)
		testErr := errors.New("test")
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
			return nil, testErr
		}
		err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, nil)
		require.ErrorIs(t, err, testErr)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
	})

	t.Run("happy path", func(t *testing.T) {
		var teamID uint = 4
		ds, svc := setup(t)
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
			require.Equal(t, &teamID, p.TeamID)
			require.Equal(t, p.Identifier, mobileconfig.FleetFileVaultPayloadIdentifier)
			require.Equal(t, p.Name, "Disk encryption")
			require.Contains(t, string(p.Mobileconfig), `MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUxEzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhvcml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoMClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VHcJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/Ufa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JSkPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVjE26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSwVeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTfUNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSndAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0HMOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLXKLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6ClieTTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=`)
			return nil, nil
		}

		err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, ptr.Uint(teamID))
		require.NoError(t, err)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
	})
}

func TestMDMAppleDisableFileVaultAndEscrow(t *testing.T) {
	var wantTeamID uint
	ds, svc := setup(t)
	ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
		require.NotNil(t, teamID)
		require.Equal(t, wantTeamID, *teamID)
		require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, profileIdentifier)
		return nil
	}

	err := svc.MDMAppleDisableFileVaultAndEscrow(context.Background(), ptr.Uint(wantTeamID))
	require.NoError(t, err)
	require.True(t, ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFuncInvoked)
}

func TestGetOrCreatePreassignTeam(t *testing.T) {
	ds, svc := setup(t)
	a, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc.authz = a
	svc.logger = log.NewNopLogger()

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
		},
	}}
	preassignGroups := []string{"one", "three"}
	// initialize team store with team one and two already created
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
			return nil, ctxerr.Wrap(ctx, &notFoundError{})
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

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: test.UserAdmin})

	t.Run("get preassign team", func(t *testing.T) {
		setupDS(t)

		// preasign group corresponds to existing team so simply get it
		team, err := svc.getOrCreatePreassignTeam(ctx, preassignGroups[0:1])
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
		// setup ds with assertions for this test
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
			// // NOTE: BootstrapPackage is currently ignored by svc.ModifyTeam and gets set
			// // instead by CopyDefaultMDMAppleBootstrapPackage below
			// require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication) // set to default
			require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant, team.Config.MDM.MacOSSetup.MacOSSetupAssistant)                 // set to default
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
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			wantArgs, err := json.Marshal(map[string]interface{}{
				"task":    worker.MacosSetupAssistantUpdateProfile,
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
		globalSetupAsst := &fleet.MDMAppleSetupAssistant{
			ID:          15,
			TeamID:      nil,
			Name:        "test asst",
			Profile:     json.RawMessage(`{"foo": "bar"}`),
			ProfileUUID: "abc-def",
		}
		getSetupAsstFuncCalls := 0
		ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
			// first call is to grab the global team setup assistant, the
			// rest are for the team being created
			if getSetupAsstFuncCalls == 0 {
				require.Nil(t, teamID)
			} else {
				require.NotNil(t, teamID)
				require.EqualValues(t, lastTeamID, *teamID)
			}
			getSetupAsstFuncCalls++
			return globalSetupAsst, nil
		}
		ds.SetOrUpdateMDMAppleSetupAssistantFunc = func(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
			require.Equal(t, globalSetupAsst.Name, asst.Name)
			require.JSONEq(t, string(globalSetupAsst.Profile), string(asst.Profile))
			require.NotNil(t, asst.TeamID)
			require.EqualValues(t, lastTeamID, *asst.TeamID)
			return asst, nil
		}

		// new team is created with bootstrap package and end user auth based on app config
		team, err := svc.getOrCreatePreassignTeam(ctx, preassignGroups)
		require.NoError(t, err)
		require.Equal(t, uint(3), team.ID)
		require.Equal(t, teamNameFromPreassignGroups(preassignGroups), team.Name)
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
		require.True(t, ds.NewJobFuncInvoked)
		resetInvoked()

		// when called again, simply get the previously created team
		team, err = svc.getOrCreatePreassignTeam(ctx, preassignGroups)
		require.NoError(t, err)
		require.Equal(t, uint(3), team.ID)
		require.Equal(t, teamNameFromPreassignGroups(preassignGroups), team.Name)
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
		resetInvoked()

		// when a custom setup assistant is not set for "no team", we don't create a custom setup assistant
		ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
			require.Nil(t, teamID)
			return nil, ctxerr.Wrap(ctx, &notFoundError{})
		}
		preassignGrousWithFoo := append(preassignGroups, "foo")
		team, err = svc.getOrCreatePreassignTeam(ctx, preassignGrousWithFoo)
		require.NoError(t, err)
		require.Equal(t, uint(4), team.ID)
		require.Equal(t, teamNameFromPreassignGroups(preassignGrousWithFoo), team.Name)
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
		resetInvoked()
	})

	t.Run("modify team", func(t *testing.T) {
		// setup ds with assertions this test
		setupDS(t)
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)                                         // sanity check
			require.Equal(t, tm.Name, team.Name)                                     // sanity check
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication) // not modified
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)      // not modified
			require.NotEmpty(t, team.Description)                                    // modified
			teamStore[tm.ID].Description = team.Description
			return teamStore[tm.ID], nil
		}

		// modify team does not apply defaults
		_, err := svc.ModifyTeam(ctx, 2, fleet.TeamPayload{Description: ptr.String("new description")})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("new team", func(t *testing.T) {
		// setup ds with assertions this test
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
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication) // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)      // not set
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
		// setup ds with assertions this test
		setupDS(t)
		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1)
			_, ok := teamStore[id]
			require.False(t, ok)                                                        // sanity check
			require.Equal(t, "new team spec", team.Name)                                // set
			require.Equal(t, "12.0", team.Config.MDM.MacOSUpdates.MinimumVersion.Value) // set
			require.Equal(t, "2024-01-01", team.Config.MDM.MacOSUpdates.Deadline.Value) // set                                    // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)    // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)         // not set
			team.ID = id
			teamStore[id] = team
			return team, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)                                            // sanity check
			require.Equal(t, tm.Name, team.Name)                                        // sanity check
			require.Equal(t, "12.0", team.Config.MDM.MacOSUpdates.MinimumVersion.Value) // unchanged
			require.Equal(t, "2025-01-01", team.Config.MDM.MacOSUpdates.Deadline.Value) // modified                                    // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)    // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)         // not set

			teamStore[tm.ID].Description = team.Description
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

var (
	testCert = `-----BEGIN CERTIFICATE-----
MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUx
EzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhv
cml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoM
ClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VH
cJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/U
fa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0
bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj
6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JS
kPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG
9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVj
E26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4
QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSw
VeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTf
UNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSn
dAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0H
MOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLX
KLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6Clie
TTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py
1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk
8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=
-----END CERTIFICATE-----`

	testKey = testingKey(`-----BEGIN RSA TESTING KEY-----
MIIEogIBAAKCAQEApIdwfwSNK+nc2bggcAXEb0Nmbz7SHUHBTQ5fRUdwn4J/5/qH
Rdgxwyj0sUSmdQlFEyMqsziaQjvk0gi580E34au2qpM9OJ3zB84a79R9rkQRTWFv
aPsCIyWkzqLT5nKxdI2j5Vz9w1lPQ9d/I60mdibqn1ZkzzggPW3Z5rRsvALXZxMq
/mjJtUa0keHvpNHEBHcV5D35JjfS9QDFibqhbJiGjytF0q1RJ7XWlePo6HHF9Nme
xJafiWjS4GldqLrde2J5NWnPviG1I0LVu6cga/v/+bkXrSwfkysXklKQ/F8J+Cqy
SLzggINJE2W+lP54w+LKAAAnzNjzBhRdFgVhNQIDAQABAoIBAAtUbFHC3XnVq+iu
PkWYkBNdX9NvTwbGvWnyAGuD5OSHFwnBfck4fwzCaD9Ay/mpPsF3nXwj/LNs7m/s
O+ndZty6d2S9qOyaK98wuTgkuNbkRxC+Ee73wgjrkbLNEax/32p4Sn4D7lGid8vj
LhUl2k0ult+MEnsWkVnJk8TITeiQaT2AHhMr3HKdaI86hJJfam3wEBiLBglnnKqA
TInMqHoudnFOn/C8iVCFuHCE0oo1dMalbc4rlZuRBqezVhbSMWPLypMVXQb7eixM
ScJ3m8+DooGDSIe+EW/afhN2VnFbrhQC9/DlxGfwTwsUseWv7pgp53ufyyAzzydn
2plW/4ECgYEA1Va5RzSUDxr75JX003YZiBcYrG268vosiNYWRhE7frvn5EorZBRW
t4R70Y2gcXA10aPHzpbq40t6voWtpkfynU3fyRzbBmwfiWLEgckrYMwtcNz8nhG2
ETAg4LXO9CufbwuDa66h76TpkBzQVNc5TSbBUr/apLDWjKPMz6qW7VUCgYEAxW4K
Yqp3NgJkC5DhuD098jir9AH96hGhUryOi2CasCvmbjWCgWdolD7SRZJfxOXFOtHv
7Dkp9glA1Cg/nSmEHKslaTJfBIWK+5rqVD6k6kZE/+4QQWQtUxXXVgGINnGrnPvo
6MlRJxqGUtYJ0GRTFJP4Py0gwuzf5BMIwe+fpGECgYAOhLRfMCjTTlbOG5ZpvaPH
Kys2sNEEMBpPxaIGaq3N1iPV2WZSjT/JhW6XuDevAJ/pAGhcmtCpXz2fMaG7qzHL
mr0cBqaxLTKIOvx8iKA3Gi4NfDyE1Ve6m7fhEv5eh4l2GSZ8cYn7sRFkCVH0NCFm
KrkFVKEgjBhNwefySf2zcQKBgHDVPgw7nlv4q9LMX6RbI98eMnAG/2XZ45gUeWcA
tAeBX3WXEVoBjoxDBwuJ5z/xjXHbb8JSvT+G9E0MH6cjhgSYb44aoqFD7TV0yP2S
u8/Ej0SxewrURO8aKXJW99Edz9WtRuRbwgyWJTSMbRlzbOPy2UrJ8NJWbHK9yiCE
YXmhAoGAA3QUiCCl11c1C4VsF68Fa2i7qwnty3fvFidZpW3ds0tzZdIvkpRLp5+u
XAJ5+zStdEGdnu0iXALQlY7ektawXguT/zYKg3nfS9RMGW6CxZotn4bqfQwDuttf
b1xn1jGQd/o0xFf9ojpDNy6vNojidQGHh6E3h0GYvxbnQmVNq5U=
-----END RSA TESTING KEY-----`)
)

// prevent static analysis tools from raising issues due to detection of
// private key in code.
func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }
