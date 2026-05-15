package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSetupExperienceAuth(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	teamID := uint(1)
	teamScriptID := uint(1)
	noTeamScriptID := uint(2)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SetSetupExperienceScriptFunc = func(ctx context.Context, script *fleet.Script) error {
		return nil
	}

	ds.GetSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) (*fleet.Script, error) {
		if teamID == nil {
			return &fleet.Script{ID: noTeamScriptID}, nil
		}
		switch *teamID {
		case uint(1):
			return &fleet.Script{ID: teamScriptID, TeamID: teamID}, nil
		default:
			return nil, newNotFoundError()
		}
	}
	ds.GetAnyScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("echo"), nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		if teamID == nil {
			return nil
		}
		switch *teamID {
		case uint(1):
			return nil
		default:
			return newNotFoundError() // TODO: confirm if we want to return not found on deletes
		}
	}
	ds.TeamLiteFunc = func(ctx context.Context, id uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{ID: id}, nil
	}
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
		return document, nil
	}

	testCases := []struct {
		name                  string
		user                  *fleet.User
		shouldFailTeamWrite   bool
		shouldFailGlobalWrite bool
		shouldFailTeamRead    bool
		shouldFailGlobalRead  bool
	}{
		{
			name:                  "global admin",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global maintainer",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global observer",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global observer+",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  false,
		},
		{
			name:                  "global gitops",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team admin, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team maintainer, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer+, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    false,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team gitops, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team admin, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team maintainer, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team observer+, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserverPlus}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
		{
			name:                  "team gitops, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleGitOps}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
			shouldFailTeamRead:    true,
			shouldFailGlobalRead:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			t.Run("setup experience script", func(t *testing.T) {
				err := svc.SetSetupExperienceScript(ctx, nil, "test.sh", strings.NewReader("echo"))
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				err = svc.DeleteSetupExperienceScript(ctx, nil)
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, nil, false)
				checkAuthErr(t, tt.shouldFailGlobalRead, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, nil, true)
				checkAuthErr(t, tt.shouldFailGlobalRead, err)

				err = svc.SetSetupExperienceScript(ctx, &teamID, "test.sh", strings.NewReader("echo"))
				checkAuthErr(t, tt.shouldFailTeamWrite, err)
				err = svc.DeleteSetupExperienceScript(ctx, &teamID)
				checkAuthErr(t, tt.shouldFailTeamWrite, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, &teamID, false)
				checkAuthErr(t, tt.shouldFailTeamRead, err)
				_, _, err = svc.GetSetupExperienceScript(ctx, &teamID, true)
				checkAuthErr(t, tt.shouldFailTeamRead, err)
			})
		})
	}
}

func TestIsAllSetupExperienceSoftwareRequired(t *testing.T) {
	ds := new(mock.Store)

	teamID := uint(1)
	// Use different values for macOS vs Windows to ensure the correct field is read for each platform.
	appCfg := &fleet.AppConfig{}
	appCfg.MDM.MacOSSetup.RequireAllSoftware = true
	appCfg.MDM.MacOSSetup.RequireAllSoftwareWindows = false

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appCfg, nil
	}
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{
			ID:   tid,
			Name: "team",
			Config: fleet.TeamConfigLite{
				MDM: fleet.TeamMDM{
					MacOSSetup: fleet.MacOSSetup{
						RequireAllSoftware:        false,
						RequireAllSoftwareWindows: true,
					},
				},
			},
		}, nil
	}

	tests := []struct {
		name     string
		host     *fleet.Host
		expected bool
	}{
		{
			name:     "macOS host, no team, reads macOS global config (true)",
			host:     &fleet.Host{Platform: "darwin"},
			expected: true,
		},
		{
			name:     "macOS host, with team, reads macOS team config (false)",
			host:     &fleet.Host{Platform: "darwin", TeamID: &teamID},
			expected: false,
		},
		{
			name:     "windows host, no team, reads Windows global config (false)",
			host:     &fleet.Host{Platform: "windows"},
			expected: false,
		},
		{
			name:     "windows host, with team, reads Windows team config (true)",
			host:     &fleet.Host{Platform: "windows", TeamID: &teamID},
			expected: true,
		},
		{
			name:     "linux host returns false",
			host:     &fleet.Host{Platform: "ubuntu"},
			expected: false,
		},
		{
			name:     "ios host returns false",
			host:     &fleet.Host{Platform: "ios"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := isAllSetupExperienceSoftwareRequired(t.Context(), ds, tt.host)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMaybeUpdateSetupExperience(t *testing.T) {
	ds := new(mock.Store)
	// _, ctx := newTestService(t, ds, nil, nil, nil)
	ctx := context.Background()

	hostUUID := "host-uuid"
	scriptUUID := "script-uuid"
	softwareUUID := "software-uuid"
	vppUUID := "vpp-uuid"

	t.Run("unsupported result type", func(t *testing.T) {
		_, err := maybeUpdateSetupExperienceStatus(ctx, ds, map[string]any{"key": "value"}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported result type")
	})

	t.Run("script results", func(t *testing.T) {
		testCases := []struct {
			name          string
			exitCode      int
			expected      fleet.SetupExperienceStatusResultStatus
			alwaysUpdated bool
		}{
			{
				name:          "success",
				exitCode:      0,
				expected:      fleet.SetupExperienceStatusSuccess,
				alwaysUpdated: true,
			},
			{
				name:          "failure",
				exitCode:      1,
				expected:      fleet.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ds.MaybeUpdateSetupExperienceScriptStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, scriptUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceScriptStatusFuncInvoked = false
				ds.HostByIdentifierFunc = func(ctx context.Context, uuid string) (*fleet.Host, error) {
					require.Equal(t, hostUUID, uuid)
					return &fleet.Host{ID: 1, UUID: uuid, Platform: "linux"}, nil
				}

				result := fleet.SetupExperienceScriptResult{
					HostUUID:    hostUUID,
					ExecutionID: scriptUUID,
					ExitCode:    tt.exitCode,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, nil)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceScriptStatusFuncInvoked)
			})
		}
	})

	t.Run("software install results", func(t *testing.T) {
		testCases := []struct {
			name          string
			status        fleet.SoftwareInstallerStatus
			expectStatus  fleet.SetupExperienceStatusResultStatus
			alwaysUpdated bool
		}{
			{
				name:          "success",
				status:        fleet.SoftwareInstalled,
				expectStatus:  fleet.SetupExperienceStatusSuccess,
				alwaysUpdated: true,
			},
			{
				name:          "failure",
				status:        fleet.SoftwareInstallFailed,
				expectStatus:  fleet.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
			{
				name:          "pending",
				status:        fleet.SoftwareInstallPending,
				expectStatus:  fleet.SetupExperienceStatusPending,
				alwaysUpdated: false,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, softwareUUID)
					require.Equal(t, tt.expectStatus, status)
					require.True(t, status.IsValid())
					require.True(t, status.IsTerminalStatus())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false
				ds.HostByIdentifierFunc = func(ctx context.Context, uuid string) (*fleet.Host, error) {
					require.Equal(t, hostUUID, uuid)
					return &fleet.Host{ID: 1, UUID: uuid, Platform: "linux"}, nil
				}

				result := fleet.SetupExperienceSoftwareInstallResult{
					HostUUID:        hostUUID,
					ExecutionID:     softwareUUID,
					InstallerStatus: tt.status,
				}
				activityFnCalled := false
				activityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
					activityFnCalled = true
					return nil
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, activityFn)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked)
				require.False(t, activityFnCalled)
			})
		}
	})

	t.Run("vpp install results", func(t *testing.T) {
		testCases := []struct {
			name          string
			status        string
			expected      fleet.SetupExperienceStatusResultStatus
			alwaysUpdated bool
		}{
			{
				name:          "success",
				status:        fleet.MDMAppleStatusAcknowledged,
				expected:      fleet.SetupExperienceStatusSuccess,
				alwaysUpdated: true,
			},
			{
				name:          "failure",
				status:        fleet.MDMAppleStatusError,
				expected:      fleet.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
			{
				name:          "format error",
				status:        fleet.MDMAppleStatusCommandFormatError,
				expected:      fleet.SetupExperienceStatusFailure,
				alwaysUpdated: true,
			},
			{
				name:          "pending",
				status:        fleet.MDMAppleStatusNotNow,
				expected:      fleet.SetupExperienceStatusPending,
				alwaysUpdated: false,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(ctx context.Context, hostUUID string, cmdUUID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, cmdUUID, vppUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked = false
				ds.HostByIdentifierFunc = func(ctx context.Context, uuid string) (*fleet.Host, error) {
					require.Equal(t, hostUUID, uuid)
					return &fleet.Host{ID: 1, UUID: uuid, Platform: "linux"}, nil
				}

				result := fleet.SetupExperienceVPPInstallResult{
					HostUUID:      hostUUID,
					CommandUUID:   vppUUID,
					CommandStatus: tt.status,
				}
				activityFnCalled := false
				activityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
					activityFnCalled = true
					return nil
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, activityFn)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked)
				require.False(t, activityFnCalled)
			})
		}
	})

	t.Run("software install failure triggers cancel and activity", func(t *testing.T) {
		teamID := uint(1)
		failedSoftwareTitleID := uint(42)
		failedSoftwareName := "FailedApp"
		pendingExecID := "pending-exec-id"

		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			require.Equal(t, hostUUID, hUUID)
			require.Equal(t, softwareUUID, executionID)
			require.Equal(t, fleet.SetupExperienceStatusFailure, status)
			return true, nil
		}
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID:       1,
				UUID:     hostUUID,
				Platform: "darwin",
				TeamID:   &teamID,
			}, nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			require.Equal(t, teamID, tid)
			return &fleet.TeamLite{
				ID: teamID,
				Config: fleet.TeamConfigLite{
					MDM: fleet.TeamMDM{
						MacOSSetup: fleet.MacOSSetup{
							RequireAllSoftware: true,
						},
					},
				},
			}, nil
		}

		installerID := uint(10)
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, tID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{
					ID:                              1,
					HostUUID:                        hostUUID,
					Name:                            failedSoftwareName,
					Status:                          fleet.SetupExperienceStatusFailure,
					SoftwareInstallerID:             &installerID,
					HostSoftwareInstallsExecutionID: &softwareUUID,
					SoftwareTitleID:                 &failedSoftwareTitleID,
				},
				{
					ID:                              2,
					HostUUID:                        hostUUID,
					Name:                            "PendingApp",
					Status:                          fleet.SetupExperienceStatusPending,
					SoftwareInstallerID:             &installerID,
					HostSoftwareInstallsExecutionID: &pendingExecID,
				},
			}, nil
		}
		ds.CancelHostUpcomingActivityFunc = func(ctx context.Context, hID uint, executionID string) (fleet.ActivityDetails, error) {
			require.Equal(t, uint(1), hID)
			require.Equal(t, pendingExecID, executionID)
			return nil, nil
		}
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			require.Equal(t, hostUUID, hUUID)
			return nil
		}

		var activityFnCalled bool
		var recordedActivity fleet.ActivityDetails
		activityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityFnCalled = true
			recordedActivity = activity
			return nil
		}

		result := fleet.SetupExperienceSoftwareInstallResult{
			HostUUID:        hostUUID,
			ExecutionID:     softwareUUID,
			InstallerStatus: fleet.SoftwareInstallFailed,
		}
		updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, activityFn)
		require.NoError(t, err)
		require.True(t, updated)
		require.True(t, activityFnCalled)
		require.True(t, ds.CancelPendingSetupExperienceStepsFuncInvoked)
		require.True(t, ds.CancelHostUpcomingActivityFuncInvoked)

		canceledActivity, ok := recordedActivity.(fleet.ActivityTypeCanceledSetupExperience)
		require.True(t, ok)
		require.Equal(t, uint(1), canceledActivity.HostID)
		require.Equal(t, failedSoftwareName, canceledActivity.SoftwareTitle)
		require.Equal(t, failedSoftwareTitleID, canceledActivity.SoftwareTitleID)
	})

	t.Run("late arriving result for canceled item does not trigger duplicate activity", func(t *testing.T) {
		// See https://github.com/fleetdm/fleet/pull/43437#discussion_r3074297752
		// 1. Software install A fails → triggers cancel of pending VPP install B + emits activity
		// 2. Later, B's MDM command result (Error) arrives. The datastore guard returns
		//    updated=false because B is already in "canceled" state, so the cancel/activity
		//    path is NOT entered a second time.

		teamID := uint(1)
		failedSoftwareTitleID := uint(42)
		failedSoftwareName := "FailedApp"
		pendingVPPCommandUUID := "pending-vpp-cmd"
		installerID := uint(10)
		vppTeamID := uint(1)

		// ---- Step 1: Software install A fails ----

		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			require.Equal(t, hostUUID, hUUID)
			require.Equal(t, softwareUUID, executionID)
			require.Equal(t, fleet.SetupExperienceStatusFailure, status)
			return true, nil // updated
		}
		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false

		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID:       1,
				UUID:     hostUUID,
				Platform: "darwin",
				TeamID:   &teamID,
			}, nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{
				ID: teamID,
				Config: fleet.TeamConfigLite{
					MDM: fleet.TeamMDM{
						MacOSSetup: fleet.MacOSSetup{
							RequireAllSoftware: true,
						},
					},
				},
			}, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, tID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return []*fleet.SetupExperienceStatusResult{
				{
					ID:                              1,
					HostUUID:                        hostUUID,
					Name:                            failedSoftwareName,
					Status:                          fleet.SetupExperienceStatusFailure,
					SoftwareInstallerID:             &installerID,
					HostSoftwareInstallsExecutionID: &softwareUUID,
					SoftwareTitleID:                 &failedSoftwareTitleID,
				},
				{
					ID:              2,
					HostUUID:        hostUUID,
					Name:            "PendingVPPApp",
					Status:          fleet.SetupExperienceStatusPending,
					VPPAppTeamID:    &vppTeamID,
					NanoCommandUUID: &pendingVPPCommandUUID,
				},
			}, nil
		}
		ds.CancelHostUpcomingActivityFunc = func(ctx context.Context, hID uint, executionID string) (fleet.ActivityDetails, error) {
			return nil, nil
		}
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			require.Equal(t, hostUUID, hUUID)
			return nil
		}
		ds.CancelPendingSetupExperienceStepsFuncInvoked = false
		ds.CancelHostUpcomingActivityFuncInvoked = false

		activityCallCount := 0
		activityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityCallCount++
			return nil
		}

		result := fleet.SetupExperienceSoftwareInstallResult{
			HostUUID:        hostUUID,
			ExecutionID:     softwareUUID,
			InstallerStatus: fleet.SoftwareInstallFailed,
		}
		updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, activityFn)
		require.NoError(t, err)
		require.True(t, updated)
		require.True(t, ds.CancelPendingSetupExperienceStepsFuncInvoked)
		require.Equal(t, 1, activityCallCount, "activity should have been emitted exactly once")

		// ---- Step 2: Late-arriving VPP result for B (already canceled) ----
		// The datastore guard returns (false, nil) because B's row is already "canceled".

		ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(ctx context.Context, hUUID string, cmdUUID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			require.Equal(t, hostUUID, hUUID)
			require.Equal(t, vppUUID, cmdUUID)
			require.Equal(t, fleet.SetupExperienceStatusFailure, status)
			return false, nil // guard blocked: row already canceled
		}
		ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked = false

		// Reset invoked flags so we can assert they are NOT set again.
		ds.CancelPendingSetupExperienceStepsFuncInvoked = false
		ds.CancelHostUpcomingActivityFuncInvoked = false

		vppResult := fleet.SetupExperienceVPPInstallResult{
			HostUUID:      hostUUID,
			CommandUUID:   vppUUID,
			CommandStatus: fleet.MDMAppleStatusError,
		}
		updated, err = maybeUpdateSetupExperienceStatus(ctx, ds, vppResult, activityFn)
		require.NoError(t, err)
		require.False(t, updated, "update should be blocked by datastore guard")
		require.False(t, ds.CancelPendingSetupExperienceStepsFuncInvoked, "cancel should NOT be called again")
		require.False(t, ds.CancelHostUpcomingActivityFuncInvoked, "cancel upcoming activity should NOT be called again")
		require.Equal(t, 1, activityCallCount, "activity should still have been emitted only once (no duplicate)")
	})

	t.Run("windows software install failure with require_all_software_windows=true emits activity and cancels", func(t *testing.T) {
		// Mirror of "software install failure triggers cancel and activity"
		// for a Windows host. Asserts that the same emit-once-per-host
		// invariant holds when the gating setting is `require_all_software_windows`
		// (rather than `require_all_software`, which is the macOS counterpart).
		teamID := uint(1)
		failedSoftwareTitleID := uint(99)
		failedSoftwareName := "WindowsApp"
		pendingExecID := "pending-win-exec"

		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			require.Equal(t, hostUUID, hUUID)
			require.Equal(t, softwareUUID, executionID)
			require.Equal(t, fleet.SetupExperienceStatusFailure, status)
			return true, nil
		}
		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false
		// Windows uses OsqueryHostID as the setup-experience host identifier
		// (see fleet.HostUUIDForSetupExperience). Set it so the cancel
		// helper can locate setup-experience rows.
		osqueryHostID := "windows-osquery-id"
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID: 2, UUID: hostUUID, Platform: "windows",
				TeamID: &teamID, OsqueryHostID: &osqueryHostID,
			}, nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			require.Equal(t, teamID, tid)
			return &fleet.TeamLite{
				ID: teamID,
				Config: fleet.TeamConfigLite{
					MDM: fleet.TeamMDM{
						MacOSSetup: fleet.MacOSSetup{
							RequireAllSoftwareWindows: true,
						},
					},
				},
			}, nil
		}

		installerID := uint(20)
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, tID uint) ([]*fleet.SetupExperienceStatusResult, error) {
			require.Equal(t, osqueryHostID, hUUID, "Windows looks up by OsqueryHostID, not UUID")
			return []*fleet.SetupExperienceStatusResult{
				{
					ID:                              5,
					HostUUID:                        osqueryHostID,
					Name:                            failedSoftwareName,
					Status:                          fleet.SetupExperienceStatusFailure,
					SoftwareInstallerID:             &installerID,
					HostSoftwareInstallsExecutionID: &softwareUUID,
					SoftwareTitleID:                 &failedSoftwareTitleID,
				},
				{
					ID:                              6,
					HostUUID:                        osqueryHostID,
					Name:                            "PendingWinApp",
					Status:                          fleet.SetupExperienceStatusPending,
					SoftwareInstallerID:             &installerID,
					HostSoftwareInstallsExecutionID: &pendingExecID,
				},
			}, nil
		}
		ds.CancelHostUpcomingActivityFuncInvoked = false
		ds.CancelHostUpcomingActivityFunc = func(ctx context.Context, hID uint, executionID string) (fleet.ActivityDetails, error) {
			require.Equal(t, uint(2), hID)
			require.Equal(t, pendingExecID, executionID)
			return nil, nil
		}
		ds.CancelPendingSetupExperienceStepsFuncInvoked = false
		ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
			require.Equal(t, osqueryHostID, hUUID)
			return nil
		}

		var activityFnCalled bool
		var recordedActivity fleet.ActivityDetails
		activityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityFnCalled = true
			recordedActivity = activity
			return nil
		}

		result := fleet.SetupExperienceSoftwareInstallResult{
			HostUUID:        hostUUID,
			ExecutionID:     softwareUUID,
			InstallerStatus: fleet.SoftwareInstallFailed,
		}
		updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, activityFn)
		require.NoError(t, err)
		require.True(t, updated)
		require.True(t, activityFnCalled, "Windows host with require_all_software_windows=true must emit canceled_setup_experience")
		require.True(t, ds.CancelPendingSetupExperienceStepsFuncInvoked, "Windows host must cancel pending setup-experience steps")
		require.True(t, ds.CancelHostUpcomingActivityFuncInvoked)

		canceledActivity, ok := recordedActivity.(fleet.ActivityTypeCanceledSetupExperience)
		require.True(t, ok)
		require.Equal(t, uint(2), canceledActivity.HostID)
		require.Equal(t, failedSoftwareName, canceledActivity.SoftwareTitle)
		require.Equal(t, failedSoftwareTitleID, canceledActivity.SoftwareTitleID)
	})

	t.Run("software install failure with require_all=false does not emit activity or cancel", func(t *testing.T) {
		// Spec invariant: when require_all_software (macOS) /
		// require_all_software_windows (Windows) is false, a software
		// install failure during ESP MUST NOT cancel pending steps and MUST
		// NOT emit a canceled_setup_experience activity. The device just
		// proceeds to the desktop and the failure is visible only in
		// Fleet's host activity feed (via the install-status path, not
		// canceled_setup_experience).
		teamID := uint(1)

		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			return true, nil
		}
		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false
		// Windows host with OsqueryHostID set so the cancel helper can run if
		// it ever (incorrectly) reaches the lookup path. This test asserts
		// it does NOT reach that path because of the require_all=false
		// early-return.
		osqueryHostID := "windows-osquery-id-noreq"
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID: 3, UUID: hostUUID, Platform: "windows",
				TeamID: &teamID, OsqueryHostID: &osqueryHostID,
			}, nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{
				ID: teamID,
				Config: fleet.TeamConfigLite{
					MDM: fleet.TeamMDM{
						MacOSSetup: fleet.MacOSSetup{
							RequireAllSoftwareWindows: false,
						},
					},
				},
			}, nil
		}
		ds.CancelPendingSetupExperienceStepsFuncInvoked = false
		ds.CancelHostUpcomingActivityFuncInvoked = false
		ds.ListSetupExperienceResultsByHostUUIDFuncInvoked = false

		var activityFnCalled bool
		activityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityFnCalled = true
			return nil
		}

		result := fleet.SetupExperienceSoftwareInstallResult{
			HostUUID:        hostUUID,
			ExecutionID:     softwareUUID,
			InstallerStatus: fleet.SoftwareInstallFailed,
		}
		updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, activityFn)
		require.NoError(t, err)
		require.True(t, updated, "the installer status row should still be updated to failure")
		require.False(t, activityFnCalled, "no canceled_setup_experience activity when require_all=false")
		require.False(t, ds.CancelPendingSetupExperienceStepsFuncInvoked,
			"no cancel-pending-steps when require_all=false")
		require.False(t, ds.CancelHostUpcomingActivityFuncInvoked,
			"no upcoming-activity cancel when require_all=false")
		// The early-return path inside maybeCancelPendingSetupExperienceSteps
		// should not even reach the ListSetupExperienceResultsByHostUUID query.
		require.False(t, ds.ListSetupExperienceResultsByHostUUIDFuncInvoked,
			"require_all=false should early-return before listing setup-experience results")
	})
}
