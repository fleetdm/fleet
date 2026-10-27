package service

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

type appReconcilerAndroidService struct {
	android.Service

	buildFleetAgentApplicationPolicyFunc func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error)
	setAppsForAndroidPolicyFunc          func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error)
}

func (s *appReconcilerAndroidService) BuildFleetAgentApplicationPolicy(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
	return s.buildFleetAgentApplicationPolicyFunc(ctx, hostUUID)
}

func (s *appReconcilerAndroidService) SetAppsForAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
	return s.setAppsForAndroidPolicyFunc(ctx, enterpriseName, appPolicies, hostUUIDs)
}

type appReconcilerRetryCounter struct {
	state *androidAppReconcileState
}

func (c *appReconcilerRetryCounter) RecordAttempt(ctx context.Context, hostID uint, softwareInstallerID uint, expireIn time.Duration) (int, error) {
	c.state.retries++
	c.state.recordedRetries++
	return c.state.retries, nil
}

func (c *appReconcilerRetryCounter) CountAttempts(ctx context.Context, hostID uint, softwareInstallerID uint) (int, error) {
	return c.state.retries, nil
}

func (c *appReconcilerRetryCounter) ResetAttempts(ctx context.Context, hostID uint, softwareInstallerID uint) error {
	return nil
}

func (c *appReconcilerRetryCounter) ResetInstallerAttempts(ctx context.Context, softwareInstallerIDs []uint) error {
	return nil
}

type androidAppReconcileSentList struct {
	hostUUID string
	packages []string
}

type androidAppReconcileWrite struct {
	hostID   uint
	installs []*fleet.HostAndroidSoftwareInstall
}

type androidAppReconcileState struct {
	androidEnabled         bool
	cursor                 uint
	hosts                  []*fleet.Host
	installs               []*fleet.HostAndroidSoftwareInstall
	appsInScopeByHostID    map[uint][]fleet.VPPAppTeam
	configurationUpdatedAt *time.Time
	firstVersionConfig     []byte
	secondVersionConfig    []byte
	configs                map[string][]byte
	pendingSetupHostIDs    map[uint]struct{}
	expandErr              error
	agentErr               error
	setAppsErrByHostUUID   map[string]error
	retries                int

	sent            []androidAppReconcileSentList
	replaced        []androidAppReconcileWrite
	failed          []androidAppReconcileWrite
	versionUpdates  []androidAppReconcileWrite
	recordedRetries int
	cursorSaved     bool
	savedCursor     uint
	listedHostsN    int
}

func TestReconcileAndroidApps(t *testing.T) {
	const (
		firstVersionID  uint = 10
		secondVersionID uint = 11
	)
	hostA := &fleet.Host{ID: 1, UUID: "host-a", Platform: "android"}
	hostB := &fleet.Host{ID: 2, UUID: "host-b", Platform: "android"}
	builtAt := time.Now().Add(-time.Hour)
	appOne := fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "com.example.one"}, AppTeamID: firstVersionID}
	appOneSecondVersion := fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "com.example.one"}, AppTeamID: secondVersionID}

	state := &androidAppReconcileState{}

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: state.androidEnabled}}, nil
	}
	ds.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "test"}, nil
	}
	ds.GetAndroidAppReconcileCursorFunc = func(ctx context.Context) (uint, error) {
		return state.cursor, nil
	}
	ds.ListAndroidHostsOnHostPolicyAfterIDFunc = func(ctx context.Context, afterHostID uint, limit int) ([]*fleet.Host, error) {
		var hosts []*fleet.Host
		for _, host := range state.hosts {
			if host.ID > afterHostID && len(hosts) < limit {
				hosts = append(hosts, host)
			}
		}
		state.listedHostsN = len(hosts)
		return hosts, nil
	}
	ds.GetAndroidHostIDsWithPendingSetupExperienceInstallFunc = func(ctx context.Context, hostIDs []uint) (map[uint]struct{}, error) {
		return state.pendingSetupHostIDs, nil
	}
	ds.ListHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostIDs []uint) ([]*fleet.HostAndroidSoftwareInstall, error) {
		return state.installs, nil
	}
	ds.GetAppStoreAppVersionsByFleetAndTitleFunc = func(ctx context.Context, platforms []fleet.InstallableDevicePlatform) (fleet.AppStoreAppVersionsByFleetAndTitleID, error) {
		require.Equal(t, []fleet.InstallableDevicePlatform{fleet.AndroidPlatform}, platforms)
		return fleet.AppStoreAppVersionsByFleetAndTitleID{0: {1: {
			{VPPAppsTeamsID: firstVersionID, ConfigurationUpdatedAt: state.configurationUpdatedAt, Configuration: state.firstVersionConfig},
			{VPPAppsTeamsID: secondVersionID, Configuration: state.secondVersionConfig},
		}}}, nil
	}
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return state.appsInScopeByHostID[hostID], nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		configs := make(map[string][]byte, len(state.configs))
		for appID, config := range state.configs {
			configs[appID] = config
		}
		return configs, nil
	}
	ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
		return "", state.expandErr
	}
	ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []*fleet.HostAndroidSoftwareInstall) error {
		state.replaced = append(state.replaced, androidAppReconcileWrite{hostID: hostID, installs: installs})
		return nil
	}
	ds.SetHostAndroidSoftwareInstallsRequestFailedFunc = func(ctx context.Context, hostID uint, installs []*fleet.HostAndroidSoftwareInstall) error {
		state.failed = append(state.failed, androidAppReconcileWrite{hostID: hostID, installs: installs})
		return nil
	}
	ds.UpdateHostAndroidSoftwareInstallVersionsFunc = func(ctx context.Context, hostID uint, installs []*fleet.HostAndroidSoftwareInstall) error {
		state.versionUpdates = append(state.versionUpdates, androidAppReconcileWrite{hostID: hostID, installs: installs})
		return nil
	}
	ds.SetAndroidAppReconcileCursorFunc = func(ctx context.Context, cursor uint) error {
		state.cursorSaved = true
		state.savedCursor = cursor
		return nil
	}

	androidSvc := &appReconcilerAndroidService{
		buildFleetAgentApplicationPolicyFunc: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
			if state.agentErr != nil {
				return nil, state.agentErr
			}
			return &androidmanagement.ApplicationPolicy{PackageName: "com.fleetdm.agent"}, nil
		},
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			requests := make(map[string]*android.MDMAndroidPolicyRequest, len(hostUUIDs))
			for hostUUID, policyID := range hostUUIDs {
				require.Equal(t, "enterprises/test", enterpriseName)
				require.Equal(t, hostUUID, policyID)
				packages := make([]string, 0, len(appPolicies))
				for _, appPolicy := range appPolicies {
					packages = append(packages, appPolicy.PackageName)
				}
				state.sent = append(state.sent, androidAppReconcileSentList{hostUUID: hostUUID, packages: packages})
				err := state.setAppsErrByHostUUID[hostUUID]
				if err != nil {
					return nil, err
				}
				requests[hostUUID] = &android.MDMAndroidPolicyRequest{PolicyVersion: sql.Null[int64]{V: 5, Valid: true}}
			}
			return requests, nil
		},
	}
	logger := slog.New(slog.DiscardHandler)
	retryCounter := &appReconcilerRetryCounter{state: state}

	t.Run("android mdm turned off sends nothing", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:               []*fleet.Host{hostA},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
		}

		// run the cron with Android MDM off, nothing should be sent and the cursor should not be saved
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.False(t, state.cursorSaved)
	})

	t.Run("a host with a new app in scope gets the app and the fleet agent and another host with no changes gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA, hostB},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostB.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
		}

		// run the cron, only host A should get a request, recorded as pending in the returned policy version
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.replaced, 1)
		require.Equal(t, hostA.ID, state.replaced[0].hostID)
		require.Len(t, state.replaced[0].installs, 1)
		require.Equal(t, "com.example.one", state.replaced[0].installs[0].AdamID)
		require.Equal(t, new(firstVersionID), state.replaced[0].installs[0].VPPAppTeamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.replaced[0].installs[0].Status)
		require.Equal(t, new(int64(5)), state.replaced[0].installs[0].InPolicyVersion)
		require.False(t, state.replaced[0].installs[0].PolicyBuiltAt.IsZero())
	})

	t.Run("a host with a row for an app out of scope gets a list without the app", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
		}

		// run the cron, host A should get a list with only the fleet agent and no rows should be written
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.replaced, 1)
		require.Empty(t, state.replaced[0].installs)
	})

	t.Run("a host whose row has another version with a different configuration gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:  []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig: []byte(`{"managedConfiguration":{"v":"2"}}`),
		}

		// run the cron, host A should get the app and its row should have the second version
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Len(t, state.replaced, 1)
		require.Equal(t, new(secondVersionID), state.replaced[0].installs[0].VPPAppTeamID)
	})

	t.Run("a host whose row has another version with the same configuration gets no request and its row moves to the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:  []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig: []byte(`{"managedConfiguration":{"v":"1"}}`),
		}

		// run the cron, nothing should be sent and host A's row should be updated to the second version
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Len(t, state.versionUpdates, 1)
		require.Len(t, state.versionUpdates[0].installs, 1)
		require.Equal(t, new(secondVersionID), state.versionUpdates[0].installs[0].VPPAppTeamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusVerified, state.versionUpdates[0].installs[0].Status)
	})

	t.Run("a host whose row has another version with the same configuration edited after its policy was built gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:     []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig:    []byte(`{"managedConfiguration":{"v":"1"}}`),
			configurationUpdatedAt: new(builtAt.Add(time.Minute)),
		}

		// run the cron, host A should get the app, the first version's configuration it was sent has changed since
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Empty(t, state.versionUpdates)
	})

	t.Run("a host whose row has a deleted version gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
		}

		// run the cron, host A should get the app
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose version configuration changed after its policy was built gets the app again", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			configurationUpdatedAt: new(builtAt.Add(time.Minute)),
		}

		// run the cron, host A should get the app
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose version configuration changed before its policy was built gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			configurationUpdatedAt: new(builtAt.Add(-time.Minute)),
		}

		// run the cron, nothing should be sent and the cursor should go back to the first host
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.True(t, state.cursorSaved)
		require.Zero(t, state.savedCursor)
	})

	t.Run("a host flagged for resend gets the app again", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusVerified, NeedsResend: true, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
		}

		// run the cron, host A should get the app and its new row should not be flagged
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.False(t, state.replaced[0].installs[0].NeedsResend)
	})

	t.Run("a host with a pending setup experience install gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled:      true,
			hosts:               []*fleet.Host{hostA},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			pendingSetupHostIDs: map[uint]struct{}{hostA.ID: {}},
		}

		// run the cron, nothing should be sent
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
	})

	t.Run("a host with no apps in scope and no rows gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
		}

		// run the cron, nothing should be sent
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
	})

	t.Run("a host rejected by the api is recorded as failed and another host still gets its request", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled:       true,
			hosts:                []*fleet.Host{hostA, hostB},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusBadRequest}},
		}

		// run the cron, host A's apps should be recorded as failed and host B's as sent
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 2)
		require.Len(t, state.failed, 1)
		require.Equal(t, hostA.ID, state.failed[0].hostID)
		require.Equal(t, "com.example.one", state.failed[0].installs[0].AdamID)
		require.Len(t, state.replaced, 1)
		require.Equal(t, hostB.ID, state.replaced[0].hostID)
	})

	t.Run("a host whose request failed less than an hour ago gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusFailed, NeedsResend: true, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-10 * time.Minute)},
				{HostID: hostA.ID, AdamID: "com.example.leaving", VPPAppTeamID: new(secondVersionID), Status: fleet.AndroidSoftwareInstallStatusFailed, NeedsResend: true, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-10 * time.Minute)},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
		}

		// run the cron, nothing should be sent, including the removal of the app out of scope, and no retry counted
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a host whose request failed over an hour ago is sent again and the retry is counted", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusFailed, NeedsResend: true, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-2 * time.Hour)},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			retries:             1,
		}

		// run the cron, host A should get the app and one retry should be counted
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Equal(t, 1, state.recordedRetries)
	})

	t.Run("a host retried twice gets no request until its retry count expires", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusFailed, NeedsResend: true, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-2 * time.Hour)},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			retries:             2,
		}

		// run the cron, nothing should be sent and no retry counted
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Zero(t, state.recordedRetries)

		// expire the retry count and run the cron again, host A should get the app
		state.retries = 0
		err = ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose request failed less than an hour ago gets a new app in scope right away without counting a retry", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusFailed, NeedsResend: true, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-10 * time.Minute)},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {
				appOne,
				{VPPAppID: fleet.VPPAppID{AdamID: "com.example.two"}, AppTeamID: secondVersionID},
			}},
			retries: 2,
		}

		// run the cron, host A should get both apps
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.example.two", "com.fleetdm.agent"}}}, state.sent)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a quota error stops the run without saving the cursor", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled:       true,
			hosts:                []*fleet.Host{hostA, hostB},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusTooManyRequests}},
		}

		// run the cron, it should fail after host A, host B should not be sent and nothing should be recorded
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.Error(t, err)
		require.Len(t, state.sent, 1)
		require.Empty(t, state.failed)
		require.Empty(t, state.replaced)
		require.False(t, state.cursorSaved)
	})

	t.Run("a fleet agent that can't be built sends nothing for the host", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled:      true,
			hosts:               []*fleet.Host{hostA},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			agentErr:            sql.ErrConnDone,
		}

		// run the cron, nothing should be sent or recorded
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Empty(t, state.replaced)
		require.Empty(t, state.failed)
	})

	t.Run("an app with a value the host can't supply is left out and recorded as failed", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {
				appOne,
				{VPPAppID: fleet.VPPAppID{AdamID: "com.example.vital"}, AppTeamID: secondVersionID},
			}},
			configs:   map[string][]byte{"com.example.vital": []byte(`{"managedConfiguration":{"tag":"$FLEET_HOST_VITAL_7"}}`)},
			expandErr: &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{7}, MissingNames: []string{"Asset tag"}},
		}

		// run the cron, host A should get the other app and the vital app's row should be failed
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.replaced, 1)
		statusByAdamID := make(map[string]string, len(state.replaced[0].installs))
		for _, install := range state.replaced[0].installs {
			statusByAdamID[install.AdamID] = install.Status
		}
		require.Equal(t, map[string]string{
			"com.example.one":   fleet.AndroidSoftwareInstallStatusPending,
			"com.example.vital": fleet.AndroidSoftwareInstallStatusFailed,
		}, statusByAdamID)
	})

	t.Run("a full batch saves its last host as the cursor and the next run continues after it", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA, hostB},
		}

		// run the cron with a batch size of 1, only host A should be checked and saved as the cursor
		err := ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 1)
		require.NoError(t, err)
		require.Equal(t, 1, state.listedHostsN)
		require.True(t, state.cursorSaved)
		require.Equal(t, hostA.ID, state.savedCursor)

		// run the cron again from host A, host B should be checked and saved as the cursor
		state.cursor = state.savedCursor
		err = ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 1)
		require.NoError(t, err)
		require.Equal(t, 1, state.listedHostsN)
		require.Equal(t, hostB.ID, state.savedCursor)

		// run the cron again from host B, no host should be checked and the cursor should go back to the first host
		state.cursor = state.savedCursor
		err = ReconcileApps(t.Context(), ds, androidSvc, retryCounter, logger, 1)
		require.NoError(t, err)
		require.Zero(t, state.listedHostsN)
		require.Zero(t, state.savedCursor)
	})

	t.Run("a run that times out doesn't save the cursor", func(t *testing.T) {
		*state = androidAppReconcileState{
			androidEnabled: true,
			hosts:          []*fleet.Host{hostA},
		}

		// run the cron with a context that is already canceled, it should fail without saving the cursor
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := ReconcileApps(ctx, ds, androidSvc, retryCounter, logger, 0)
		require.Error(t, err)
		require.False(t, state.cursorSaved)
	})
}
