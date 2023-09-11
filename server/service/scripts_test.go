package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestHostRunScript(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	// use a custom implementation of checkAuthErr as the service call will fail
	// with a not found error for unknown host in case of authorization success,
	// and the package-wide checkAuthErr requires no error.
	checkAuthErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
		} else if err != nil {
			require.NotEqual(t, (&authz.Forbidden{}).Error(), err.Error())
		}
	}

	teamHost := &fleet.Host{ID: 1, Hostname: "host-team", TeamID: ptr.Uint(1), SeenTime: time.Now()}
	noTeamHost := &fleet.Host{ID: 2, Hostname: "host-no-team", TeamID: nil, SeenTime: time.Now()}
	nonExistingHost := &fleet.Host{ID: 3, Hostname: "no-such-host", TeamID: nil}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.HostFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		if hostID == 1 {
			return teamHost, nil
		}
		if hostID == 2 {
			return noTeamHost, nil
		}
		return nil, newNotFoundError()
	}
	ds.NewHostScriptExecutionRequestFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
		return &fleet.HostScriptResult{HostID: request.HostID, ScriptContents: request.ScriptContents, ExecutionID: "abc"}, nil
	}
	ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, ignoreOlder time.Duration) ([]*fleet.HostScriptResult, error) {
		return nil, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		require.IsType(t, fleet.ActivityTypeRanScript{}, activity)
		return nil
	}

	t.Run("authorization checks", func(t *testing.T) {
		testCases := []struct {
			name                  string
			user                  *fleet.User
			shouldFailTeamWrite   bool
			shouldFailGlobalWrite bool
		}{
			{
				name:                  "global admin",
				user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: false,
			},
			{
				name:                  "global maintainer",
				user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: false,
			},
			{
				name:                  "global observer",
				user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global observer+",
				user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "global gitops",
				user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team admin, belongs to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team maintainer, belongs to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
				shouldFailTeamWrite:   false,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer, belongs to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer+, belongs to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team gitops, belongs to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team admin, DOES NOT belong to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team maintainer, DOES NOT belong to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer, DOES NOT belong to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team observer+, DOES NOT belong to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserverPlus}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
			{
				name:                  "team gitops, DOES NOT belong to team",
				user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleGitOps}}},
				shouldFailTeamWrite:   true,
				shouldFailGlobalWrite: true,
			},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

				_, err := svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{HostID: noTeamHost.ID, ScriptContents: "abc"}, 0)
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
				_, err = svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{HostID: teamHost.ID, ScriptContents: "abc"}, 0)
				checkAuthErr(t, tt.shouldFailTeamWrite, err)

				// a non-existing host is authorized as for global write (because we can't know what team it belongs to)
				_, err = svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{HostID: nonExistingHost.ID, ScriptContents: "abc"}, 0)
				checkAuthErr(t, tt.shouldFailGlobalWrite, err)
			})
		}
	})

	t.Run("script contents validation", func(t *testing.T) {
		testCases := []struct {
			name    string
			script  string
			wantErr string
		}{
			{"empty script", "", "Script contents must not be empty."},
			{"overly long script", strings.Repeat("a", 10001), "Script is too large."},
			{"invalid utf8", "\xff\xfa", "Wrong data format."},
			{"valid without hashbang", "echo 'a'", ""},
			{"valid with hashbang", "#!/bin/sh\necho 'a'", ""},
			{"valid with hashbang and spacing", "#! /bin/sh  \necho 'a'", ""},
			{"valid with hashbang and Windows newline", "#! /bin/sh  \r\necho 'a'", ""},
			{"invalid hashbang", "#!/bin/bash\necho 'a'", "Interpreter not supported."},
			{"invalid hashbang suffix", "#!/bin/sh -n\necho 'a'", "Interpreter not supported."},
		}

		ctx = viewer.NewContext(ctx, viewer.Viewer{User: test.UserAdmin})
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				_, err := svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{HostID: noTeamHost.ID, ScriptContents: tt.script}, 0)
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestGetScriptResult(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	const (
		noTeamHostExecID      = "no-team-host"
		teamHostExecID        = "team-host"
		nonExistingHostExecID = "non-existing-host"
	)

	checkAuthErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
		} else if err != nil {
			require.NotEqual(t, (&authz.Forbidden{}).Error(), err.Error())
		}
	}

	teamHost := &fleet.Host{ID: 1, Hostname: "host-team", TeamID: ptr.Uint(1), SeenTime: time.Now()}
	noTeamHost := &fleet.Host{ID: 2, Hostname: "host-no-team", TeamID: nil, SeenTime: time.Now()}
	nonExistingHost := &fleet.Host{ID: 3, Hostname: "no-such-host", TeamID: nil}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.GetHostScriptExecutionResultFunc = func(ctx context.Context, executionID string) (*fleet.HostScriptResult, error) {
		switch executionID {
		case noTeamHostExecID:
			return &fleet.HostScriptResult{HostID: noTeamHost.ID, ScriptContents: "abc", ExecutionID: executionID}, nil
		case teamHostExecID:
			return &fleet.HostScriptResult{HostID: teamHost.ID, ScriptContents: "abc", ExecutionID: executionID}, nil
		case nonExistingHostExecID:
			return &fleet.HostScriptResult{HostID: nonExistingHost.ID, ScriptContents: "abc", ExecutionID: executionID}, nil
		default:
			return nil, newNotFoundError()
		}
	}
	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		if hostID == 1 {
			return teamHost, nil
		}
		if hostID == 2 {
			return noTeamHost, nil
		}
		return nil, newNotFoundError()
	}

	testCases := []struct {
		name                 string
		user                 *fleet.User
		shouldFailTeamRead   bool
		shouldFailGlobalRead bool
	}{
		{
			name:                 "global admin",
			user:                 &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global maintainer",
			user:                 &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global observer",
			user:                 &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global observer+",
			user:                 &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: false,
		},
		{
			name:                 "global gitops",
			user:                 &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team admin, belongs to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team maintainer, belongs to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer, belongs to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer+, belongs to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			shouldFailTeamRead:   false,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team gitops, belongs to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team admin, DOES NOT belong to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team maintainer, DOES NOT belong to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer, DOES NOT belong to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team observer+, DOES NOT belong to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserverPlus}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
		{
			name:                 "team gitops, DOES NOT belong to team",
			user:                 &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleGitOps}}},
			shouldFailTeamRead:   true,
			shouldFailGlobalRead: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetScriptResult(ctx, noTeamHostExecID)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)
			_, err = svc.GetScriptResult(ctx, teamHostExecID)
			checkAuthErr(t, tt.shouldFailTeamRead, err)

			// a non-existing host is authorized as for global write (because we can't know what team it belongs to)
			_, err = svc.GetScriptResult(ctx, nonExistingHostExecID)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)
		})
	}
}

func TestNewScript(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewScriptFunc = func(ctx context.Context, script *fleet.Script) (*fleet.Script, error) {
		newScript := *script
		newScript.ID = 1
		return &newScript, nil
	}

	testCases := []struct {
		name                  string
		user                  *fleet.User
		shouldFailTeamWrite   bool
		shouldFailGlobalWrite bool
	}{
		{
			name:                  "global admin",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
		},
		{
			name:                  "global maintainer",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: false,
		},
		{
			name:                  "global observer",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "global observer+",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "global gitops",
			user:                  &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team admin, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team maintainer, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamWrite:   false,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team observer, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team observer+, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team gitops, belongs to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team admin, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team maintainer, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team observer, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team observer+, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserverPlus}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
		{
			name:                  "team gitops, DOES NOT belong to team",
			user:                  &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleGitOps}}},
			shouldFailTeamWrite:   true,
			shouldFailGlobalWrite: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.NewScript(ctx, nil, "test.sh", strings.NewReader("echo"))
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)
			_, err = svc.NewScript(ctx, ptr.Uint(1), "test.sh", strings.NewReader("echo"))
			checkAuthErr(t, tt.shouldFailTeamWrite, err)
		})
	}
}
