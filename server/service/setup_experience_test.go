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
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id}, nil
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

func TestMaybeUpdateSetupExperience(t *testing.T) {
	ds := new(mock.Store)
	// _, ctx := newTestService(t, ds, nil, nil, nil)
	ctx := context.Background()

	hostUUID := "host-uuid"
	scriptUUID := "script-uuid"
	softwareUUID := "software-uuid"
	vppUUID := "vpp-uuid"

	t.Run("unsupported result type", func(t *testing.T) {
		_, err := maybeUpdateSetupExperienceStatus(ctx, ds, map[string]interface{}{"key": "value"}, true)
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

				result := fleet.SetupExperienceScriptResult{
					HostUUID:    hostUUID,
					ExecutionID: scriptUUID,
					ExitCode:    tt.exitCode,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, true)
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
				requireTerminalStatus := true // when this flag is true, we don't expect pending status to update

				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, softwareUUID)
					require.Equal(t, tt.expectStatus, status)
					require.True(t, status.IsValid())
					require.True(t, status.IsTerminalStatus())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false

				result := fleet.SetupExperienceSoftwareInstallResult{
					HostUUID:        hostUUID,
					ExecutionID:     softwareUUID,
					InstallerStatus: tt.status,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked)

				requireTerminalStatus = false // when this flag is false, we do expect pending status to update

				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, executionID, softwareUUID)
					require.Equal(t, tt.expectStatus, status)
					require.True(t, status.IsValid())
					if status.IsTerminalStatus() {
						require.True(t, status == fleet.SetupExperienceStatusSuccess || status == fleet.SetupExperienceStatusFailure)
					} else {
						require.True(t, status == fleet.SetupExperienceStatusPending || status == fleet.SetupExperienceStatusRunning)
					}
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked = false
				updated, err = maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				shouldUpdate := tt.alwaysUpdated
				if tt.expectStatus == fleet.SetupExperienceStatusPending || tt.expectStatus == fleet.SetupExperienceStatusRunning {
					shouldUpdate = true
				}
				require.Equal(t, shouldUpdate, updated)
				require.Equal(t, shouldUpdate, ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFuncInvoked)
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
				requireTerminalStatus := true // when this flag is true, we don't expect pending status to update

				ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(ctx context.Context, hostUUID string, cmdUUID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, cmdUUID, vppUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked = false

				result := fleet.SetupExperienceVPPInstallResult{
					HostUUID:      hostUUID,
					CommandUUID:   vppUUID,
					CommandStatus: tt.status,
				}
				updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				require.Equal(t, tt.alwaysUpdated, updated)
				require.Equal(t, tt.alwaysUpdated, ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked)

				requireTerminalStatus = false // when this flag is false, we do expect pending status to update

				ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(ctx context.Context, hostUUID string, cmdUUID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
					require.Equal(t, hostUUID, hostUUID)
					require.Equal(t, cmdUUID, vppUUID)
					require.Equal(t, tt.expected, status)
					require.True(t, status.IsValid())
					if status.IsTerminalStatus() {
						require.True(t, status == fleet.SetupExperienceStatusSuccess || status == fleet.SetupExperienceStatusFailure)
					} else {
						require.True(t, status == fleet.SetupExperienceStatusPending || status == fleet.SetupExperienceStatusRunning)
					}
					return true, nil
				}
				ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked = false

				updated, err = maybeUpdateSetupExperienceStatus(ctx, ds, result, requireTerminalStatus)
				require.NoError(t, err)
				shouldUpdate := tt.alwaysUpdated
				if tt.expected == fleet.SetupExperienceStatusPending || tt.expected == fleet.SetupExperienceStatusRunning {
					shouldUpdate = true
				}
				require.Equal(t, shouldUpdate, updated)
				require.Equal(t, shouldUpdate, ds.MaybeUpdateSetupExperienceVPPStatusFuncInvoked)
			})
		}
	})
}
