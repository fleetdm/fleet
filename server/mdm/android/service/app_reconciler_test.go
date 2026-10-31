package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

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

type androidAppReconcileState struct {
	hosts                  []*fleet.Host
	installs               []*fleet.HostAndroidSoftwareInstall
	appsInScopeByHostID    map[uint][]fleet.VPPAppTeam
	configurationUpdatedAt *time.Time
	firstVersionConfig     []byte
	secondVersionConfig    []byte
	setupApps              []*fleet.HostAndroidVPPSoftwareInstall
	profilesToSend         []*fleet.MDMAndroidProfilePayload
	expandErr              error
	agentErr               error
	setAppsErrByHostUUID   map[string]error
	retries                int

	sent                     []androidAppReconcileSentList
	sentOpts                 []androidmgmt.PoliciesPatchOpts
	sentConfigsByHostUUID    map[string]map[string]string
	upserted                 []fleet.HostAndroidSoftwareInstall
	removed                  []fleet.HostAndroidSoftwareInstall
	recordedRetries          int
	pendingApplyConfigAppIDs []string
	flaggedSentHostIDs       []uint
}

func TestReconcileAndroidApps(t *testing.T) {
	const (
		firstVersionID  uint = 10
		secondVersionID uint = 11
	)
	hostA := &fleet.Host{ID: 1, UUID: "host-a", Platform: "android"}
	hostB := &fleet.Host{ID: 2, UUID: "host-b", Platform: "android"}
	builtAt := time.Now().Add(-time.Hour)
	appOne := fleet.VPPAppTeam{AdamID: "com.example.one", AppTeamID: firstVersionID}
	appOneSecondVersion := fleet.VPPAppTeam{AdamID: "com.example.one", AppTeamID: secondVersionID}

	state := &androidAppReconcileState{}

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	ds.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "test"}, nil
	}
	ds.ListAndroidHostsOnHostPolicyFunc = func(ctx context.Context) ([]*fleet.Host, error) {
		return state.hosts, nil
	}
	ds.ListAndroidAppsInScopeForHostsFunc = func(ctx context.Context, hostIDs []uint) (map[uint][]fleet.VPPAppTeam, error) {
		return state.appsInScopeByHostID, nil
	}
	ds.ListHostMDMAndroidVPPAppsFromSetupExperienceFunc = func(ctx context.Context, hostIDs []uint) (map[uint][]*fleet.HostAndroidVPPSoftwareInstall, error) {
		setupAppsByHostID := make(map[uint][]*fleet.HostAndroidVPPSoftwareInstall)
		for _, setupApp := range state.setupApps {
			setupAppsByHostID[setupApp.HostID] = append(setupAppsByHostID[setupApp.HostID], setupApp)
		}
		return setupAppsByHostID, nil
	}
	ds.ListHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostIDs []uint) (map[uint]map[string]*fleet.HostAndroidSoftwareInstall, error) {
		installsByHostID := make(map[uint]map[string]*fleet.HostAndroidSoftwareInstall)
		for _, install := range state.installs {
			if installsByHostID[install.HostID] == nil {
				installsByHostID[install.HostID] = make(map[string]*fleet.HostAndroidSoftwareInstall)
			}
			installsByHostID[install.HostID][install.AdamID] = install
		}
		return installsByHostID, nil
	}
	ds.GetAndroidAppStoreAppVersionsByIDFunc = func(ctx context.Context) (map[uint]*fleet.VPPAppStoreApp, error) {
		return map[uint]*fleet.VPPAppStoreApp{
			firstVersionID:  {VPPAppsTeamsID: firstVersionID, ConfigurationUpdatedAt: state.configurationUpdatedAt, Configuration: state.firstVersionConfig},
			secondVersionID: {VPPAppsTeamsID: secondVersionID, Configuration: state.secondVersionConfig},
		}, nil
	}
	ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
		return "", state.expandErr
	}
	ds.BulkUpsertHostAndroidSoftwareInstallsFunc = func(ctx context.Context, installs []fleet.HostAndroidSoftwareInstall, removedInstalls []fleet.HostAndroidSoftwareInstall) error {
		state.upserted = append(state.upserted, installs...)
		state.removed = append(state.removed, removedInstalls...)
		return nil
	}
	ds.NewAndroidPolicyRequestFunc = func(ctx context.Context, req *android.MDMAndroidPolicyRequest) error {
		return nil
	}
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{Host: &fleet.Host{UUID: hostUUID}, Device: &android.Device{AppliedPolicyID: new(hostUUID)}}, nil
	}
	ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
		if state.agentErr != nil {
			return nil, state.agentErr
		}
		return []*fleet.EnrollSecret{{Secret: "secret"}}, nil
	}
	ds.ListCertificateTemplatesForHostsFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.CertificateTemplateForHost, error) {
		return nil, nil
	}
	ds.SetAndroidAppInstallPendingApplyConfigFunc = func(ctx context.Context, hostUUID, applicationID string, policyVersion int64) error {
		state.pendingApplyConfigAppIDs = append(state.pendingApplyConfigAppIDs, applicationID)
		return nil
	}
	ds.SetHostAndroidSoftwareInstallsNeedsResendFunc = func(ctx context.Context, hostIDs []uint, builtAfter *time.Time, adamID *string) error {
		state.flaggedSentHostIDs = append(state.flaggedSentHostIDs, hostIDs...)
		return nil
	}

	// certificate templates
	ds.ListAndroidHostUUIDsWithPendingCertificateTemplatesFunc = func(ctx context.Context, afterHostUUID string, limit int) ([]string, error) {
		return nil, nil
	}

	// profiles
	ds.GetMDMAndroidReconcileCursorFunc = func(ctx context.Context) (string, error) {
		return "", nil
	}
	ds.SetMDMAndroidReconcileCursorFunc = func(ctx context.Context, cursor string) error {
		return nil
	}
	ds.GetMDMAndroidProfilesWriteTimeFunc = func(ctx context.Context) (time.Time, error) {
		return time.Now(), nil
	}
	ds.ListMDMAndroidProfilesToSendFunc = func(ctx context.Context, cursor string, batchSize int) ([]*fleet.MDMAndroidProfilePayload, []*fleet.MDMAndroidProfilePayload, error) {
		return state.profilesToSend, nil, nil
	}
	ds.GetMDMAndroidProfilesContentsFunc = func(ctx context.Context, uuids []string) (map[string]json.RawMessage, error) {
		contents := make(map[string]json.RawMessage, len(uuids))
		for _, profileUUID := range uuids {
			contents[profileUUID] = json.RawMessage(`{"cameraDisabled": true}`)
		}
		return contents, nil
	}
	ds.BulkUpsertMDMAndroidHostProfilesUnlessResetSinceFunc = func(ctx context.Context, payload []*fleet.MDMAndroidProfilePayload, since time.Time) error {
		return nil
	}

	client := &android_mock.Client{}
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		hostUUID, found := strings.CutPrefix(policyName, "enterprises/test/policies/")
		require.True(t, found)
		if !opts.ExcludeApps {
			packages := make([]string, 0, len(policy.Applications))
			for _, appPolicy := range policy.Applications {
				packages = append(packages, appPolicy.PackageName)
				if len(appPolicy.ManagedConfiguration) > 0 {
					if state.sentConfigsByHostUUID == nil {
						state.sentConfigsByHostUUID = make(map[string]map[string]string)
					}
					if state.sentConfigsByHostUUID[hostUUID] == nil {
						state.sentConfigsByHostUUID[hostUUID] = make(map[string]string)
					}
					state.sentConfigsByHostUUID[hostUUID][appPolicy.PackageName] = string(appPolicy.ManagedConfiguration)
				}
			}
			state.sent = append(state.sent, androidAppReconcileSentList{hostUUID: hostUUID, packages: packages})
			state.sentOpts = append(state.sentOpts, opts)
		}
		err := state.setAppsErrByHostUUID[hostUUID]
		if err != nil {
			return nil, err
		}
		return &androidmanagement.Policy{Version: 5}, nil
	}
	logger := slog.New(slog.DiscardHandler)
	retryCounter := &appReconcilerRetryCounter{state: state}
	reconcile := func(batchSize int) error {
		return ReconcileProfilesWithClient(t.Context(), ds, logger, "", client, config.AndroidAgentConfig{Package: "com.fleetdm.agent", SigningSHA256: "abc123def456"}, batchSize, retryCounter)
	}
	agentInstall := func(hostID uint, status string, needsResend bool) *fleet.HostAndroidSoftwareInstall {
		return &fleet.HostAndroidSoftwareInstall{HostID: hostID, AdamID: "com.fleetdm.agent", InstallType: android.AppStatusForceInstalled, Status: status, NeedsResend: needsResend, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-2 * time.Hour)}
	}
	appOneInstall := func(hostID uint, status string, needsResend bool, updatedAt time.Time) *fleet.HostAndroidSoftwareInstall {
		return &fleet.HostAndroidSoftwareInstall{HostID: hostID, AdamID: "com.example.one", InstallType: android.AppStatusAvailable, VPPAppTeamID: new(firstVersionID), Status: status, NeedsResend: needsResend, PolicyBuiltAt: builtAt, UpdatedAt: updatedAt}
	}

	t.Run("a host with a new app in scope gets the app and the Fleet agent and another host with no changes gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA, hostB},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
				agentInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
		}

		// run the cron, only host A should get a request, its app and the Fleet agent recorded as pending in the returned
		// policy version
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Equal(t, []androidmgmt.PoliciesPatchOpts{{OnlyUpdateApps: true}}, state.sentOpts)
		require.Len(t, state.upserted, 2)
		require.Equal(t, hostA.ID, state.upserted[0].HostID)
		require.Equal(t, "com.example.one", state.upserted[0].AdamID)
		require.Equal(t, new(firstVersionID), state.upserted[0].VPPAppTeamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.upserted[0].Status)
		require.Equal(t, new(int64(5)), state.upserted[0].InPolicyVersion)
		require.False(t, state.upserted[0].PolicyBuiltAt.IsZero())
		require.Equal(t, hostA.ID, state.upserted[1].HostID)
		require.Equal(t, "com.fleetdm.agent", state.upserted[1].AdamID)
		require.Nil(t, state.upserted[1].VPPAppTeamID)
		require.Equal(t, android.AppStatusForceInstalled, state.upserted[1].InstallType)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.upserted[1].Status)
		require.Equal(t, new(int64(5)), state.upserted[1].InPolicyVersion)
	})

	t.Run("two hosts with an app whose configuration uses the host UUID variable each get their own UUID in the app's configuration", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:               []*fleet.Host{hostA, hostB},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			firstVersionConfig:  []byte(`{"managedConfiguration":{"deviceId":"$FLEET_VAR_HOST_UUID"}}`),
		}

		// run the cron, host A and host B should each get app one with their own UUID substituted in its configuration
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 2)
		require.JSONEq(t, `{"deviceId":"host-a"}`, state.sentConfigsByHostUUID["host-a"]["com.example.one"])
		require.JSONEq(t, `{"deviceId":"host-b"}`, state.sentConfigsByHostUUID["host-b"]["com.example.one"])
	})

	t.Run("a host with a failed app gets it as pending with a new app and a host with only the failed app gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA, hostB},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusFailed, false, time.Time{}),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
				appOneInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusFailed, false, time.Time{}),
				agentInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{
				hostA.ID: {appOne, {AdamID: "com.example.two", AppTeamID: secondVersionID}},
				hostB.ID: {appOne},
			},
		}

		// run the cron, only host A should get a request, with app one upserted as pending because its configuration
		// now resolves
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.example.two", "com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.upserted, 3)
		require.Equal(t, hostA.ID, state.upserted[0].HostID)
		require.Equal(t, "com.example.one", state.upserted[0].AdamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.upserted[0].Status)
		require.Equal(t, "com.example.two", state.upserted[1].AdamID)
		require.Equal(t, "com.fleetdm.agent", state.upserted[2].AdamID)
	})

	t.Run("a host with a new app in scope gets both apps and only the new app's install is upserted", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", InstallType: android.AppStatusAvailable, VPPAppTeamID: new(firstVersionID), Status: fleet.AndroidSoftwareInstallStatusPending, InPolicyVersion: new(int64(4)), PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {
				appOne,
				{AdamID: "com.example.two", AppTeamID: secondVersionID},
			}},
		}

		// run the cron, host A should get both apps and the unchanged app one should not be upserted, so a status report
		// that verifies it during the run is not overwritten
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.example.two", "com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.upserted, 2)
		require.Equal(t, "com.example.two", state.upserted[0].AdamID)
		require.Equal(t, "com.fleetdm.agent", state.upserted[1].AdamID)
		require.Empty(t, state.removed)
	})

	t.Run("a host with a row for an app out of scope gets a list without the app", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
			},
		}

		// run the cron, host A should get a list with only the Fleet agent and its install of the app should be removed
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.upserted, 1)
		require.Equal(t, "com.fleetdm.agent", state.upserted[0].AdamID)
		require.Len(t, state.removed, 1)
		require.Equal(t, hostA.ID, state.removed[0].HostID)
		require.Equal(t, "com.example.one", state.removed[0].AdamID)
	})

	t.Run("a host whose row has another version with a different configuration gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:  []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig: []byte(`{"managedConfiguration":{"v":"2"}}`),
		}

		// run the cron, host A should get the app and its row should have the second version
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Len(t, state.upserted, 2)
		require.Equal(t, new(secondVersionID), state.upserted[0].VPPAppTeamID)
	})

	t.Run("a host whose row has another version with the same configuration gets no request and its row is set to the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:  []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig: []byte(`{"managedConfiguration":{"v":"1"}}`),
		}

		// run the cron, nothing should be sent and host A's row should be updated to the second version
		err := reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Len(t, state.upserted, 1)
		require.Equal(t, new(secondVersionID), state.upserted[0].VPPAppTeamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusVerified, state.upserted[0].Status)
		require.True(t, builtAt.Equal(state.upserted[0].PolicyBuiltAt))
	})

	t.Run("a host whose row has another version with the same configuration and another install type gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:  []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig: []byte(`{"managedConfiguration":{"v":"1"}}`),
			setupApps:           []*fleet.HostAndroidVPPSoftwareInstall{{HostID: hostA.ID, AdamID: "com.example.one", VPPAppTeamID: secondVersionID}},
		}

		// run the cron, host A should get a request and its row should be set to the second version as preinstalled
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Equal(t, "com.example.one", state.upserted[0].AdamID)
		require.Equal(t, new(secondVersionID), state.upserted[0].VPPAppTeamID)
		require.Equal(t, android.AppStatusPreinstalled, state.upserted[0].InstallType)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.upserted[0].Status)
	})

	t.Run("a host whose row has another version with the same configuration edited after its policy was built gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
			firstVersionConfig:     []byte(`{"managedConfiguration":{"v":"1"}}`),
			secondVersionConfig:    []byte(`{"managedConfiguration":{"v":"1"}}`),
			configurationUpdatedAt: new(builtAt.Add(time.Minute)),
		}

		// run the cron, host A should get the app, the first version's configuration it was sent has changed since
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose row has a deleted version gets the version in scope", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.one", InstallType: android.AppStatusAvailable, Status: fleet.AndroidSoftwareInstallStatusVerified, PolicyBuiltAt: builtAt},
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOneSecondVersion}},
		}

		// run the cron, host A should get the app
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose version configuration changed after its policy was built gets the app again", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			configurationUpdatedAt: new(builtAt.Add(time.Minute)),
		}

		// run the cron, host A should get the app
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose version configuration changed before its policy was built gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			configurationUpdatedAt: new(builtAt.Add(-time.Minute)),
		}

		// run the cron, nothing should be sent
		err := reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
	})

	t.Run("a host flagged for resend gets the app again", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, true, time.Time{}),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
		}

		// run the cron, host A should get the app and its new row should not be flagged
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.False(t, state.upserted[0].NeedsResend)
	})

	t.Run("a host with a setup experience app out of scope keeps it preinstalled when it gets a new app", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				{HostID: hostA.ID, AdamID: "com.example.setup", InstallType: android.AppStatusPreinstalled, VPPAppTeamID: new(secondVersionID), Status: fleet.AndroidSoftwareInstallStatusPending, PolicyBuiltAt: builtAt},
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			setupApps:           []*fleet.HostAndroidVPPSoftwareInstall{{HostID: hostA.ID, AdamID: "com.example.setup", VPPAppTeamID: secondVersionID}},
		}

		// run the cron, host A should get the new app, the setup experience app and the Fleet agent, and the setup
		// experience app's row should be neither upserted nor removed
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.example.setup", "com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.upserted, 2)
		require.Equal(t, "com.example.one", state.upserted[0].AdamID)
		require.Equal(t, android.AppStatusAvailable, state.upserted[0].InstallType)
		require.Equal(t, "com.fleetdm.agent", state.upserted[1].AdamID)
		require.Empty(t, state.removed)
	})

	t.Run("a host with no apps in scope and no rows gets one list with only the Fleet agent", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
		}

		// run the cron, host A should get a list with only the Fleet agent, which replaces any apps older jobs left in its
		// policy, and the agent's row should be upserted as pending in the returned policy version
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.fleetdm.agent"}}}, state.sent)
		require.Len(t, state.upserted, 1)
		require.Equal(t, "com.fleetdm.agent", state.upserted[0].AdamID)
		require.Equal(t, android.AppStatusForceInstalled, state.upserted[0].InstallType)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.upserted[0].Status)
		require.Equal(t, new(int64(5)), state.upserted[0].InPolicyVersion)

		// read back the agent's row and run the cron again, nothing should be sent to host A
		upsertedAgentInstall := state.upserted[0]
		state.installs = []*fleet.HostAndroidSoftwareInstall{&upsertedAgentInstall}
		state.sent = nil
		err = reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
	})

	t.Run("a host rejected by the api is recorded as failed and another host still gets its request", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:                []*fleet.Host{hostA, hostB},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusBadRequest}},
		}

		// run the cron, host A's app and Fleet agent should be recorded as request failed and host B's as sent
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 2)
		require.Len(t, state.upserted, 4)
		require.Equal(t, hostA.ID, state.upserted[0].HostID)
		require.Equal(t, "com.example.one", state.upserted[0].AdamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusRequestFailed, state.upserted[0].Status)
		require.False(t, state.upserted[0].NeedsResend)
		require.Equal(t, hostA.ID, state.upserted[1].HostID)
		require.Equal(t, "com.fleetdm.agent", state.upserted[1].AdamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusRequestFailed, state.upserted[1].Status)
		require.Equal(t, hostB.ID, state.upserted[2].HostID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, state.upserted[2].Status)
	})

	t.Run("a host whose list gets a not modified response has its app and Fleet agent recorded as request failed", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:                []*fleet.Host{hostA},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusNotModified}},
		}

		// run the cron, host A's app and Fleet agent should be recorded as request failed with no policy version, so a
		// later run will retry them
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Len(t, state.upserted, 2)
		require.Equal(t, "com.example.one", state.upserted[0].AdamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusRequestFailed, state.upserted[0].Status)
		require.Nil(t, state.upserted[0].InPolicyVersion)
		require.Equal(t, "com.fleetdm.agent", state.upserted[1].AdamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusRequestFailed, state.upserted[1].Status)
	})

	t.Run("a host whose list was sent has the rows written during the run flagged and a rejected host doesn't", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:                []*fleet.Host{hostA, hostB},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusBadRequest}},
		}

		// run the cron with host A's request rejected, only host B's rows written during the run should be flagged
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []uint{hostB.ID}, state.flaggedSentHostIDs)
	})

	t.Run("a host whose app failed for a missing value and then flagged for resend gets the app right away", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusFailed, true, time.Now()),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
		}

		// run the cron, host A should get the app without waiting to retry and no retry should be counted
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a host whose request failed less than an hour ago gets a configuration edit right away", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false, time.Now().Add(-10*time.Minute)),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false),
			},
			appsInScopeByHostID:    map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			configurationUpdatedAt: new(builtAt.Add(time.Minute)),
		}

		// run the cron, host A should get the edited app without waiting to retry and no retry should be counted
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a quota error on a host's profile patch with its app list saves none of the host's apps", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:                []*fleet.Host{hostA},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			profilesToSend:       []*fleet.MDMAndroidProfilePayload{{HostUUID: "host-a", ProfileUUID: "profile-1", ProfileName: "Profile 1"}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusTooManyRequests}},
		}

		// run the cron, host A's profiles and apps should go in one patch, and none of host A's apps should be saved
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Empty(t, state.upserted)
	})

	t.Run("a host whose request failed less than an hour ago gets no request", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false, time.Now().Add(-10*time.Minute)),
				{HostID: hostA.ID, AdamID: "com.example.leaving", InstallType: android.AppStatusAvailable, VPPAppTeamID: new(secondVersionID), Status: fleet.AndroidSoftwareInstallStatusRequestFailed, PolicyBuiltAt: builtAt, UpdatedAt: time.Now().Add(-10 * time.Minute)},
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
		}

		// run the cron, nothing should be sent, including the removal of the app out of scope, and no retry counted
		err := reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a host whose request failed over an hour ago is sent again and the retry is counted", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false, time.Now().Add(-2*time.Hour)),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			retries:             1,
		}

		// run the cron, host A should get the app and one retry should be counted
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Equal(t, 1, state.recordedRetries)
	})

	t.Run("a host retried twice gets no request until its retry count expires", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false, time.Now().Add(-2*time.Hour)),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			retries:             2,
		}

		// run the cron, nothing should be sent and no retry counted
		err := reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Zero(t, state.recordedRetries)

		// expire the retry count and run the cron again, host A should get the app
		state.retries = 0
		err = reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
	})

	t.Run("a host whose retry the batch size leaves for the next run uses no retry", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA, hostB},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
				appOneInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false, time.Now().Add(-2*time.Hour)),
				agentInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			profilesToSend:      []*fleet.MDMAndroidProfilePayload{{HostUUID: "host-a", ProfileUUID: "profile-1", ProfileName: "Profile 1"}},
		}

		// run the cron with a batch size of 1, host A's profile patch should use the run's only patch and host B's retry
		// should wait for the next run without being counted
		err := reconcile(1)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a host with profile changes gets its app list in its profile patch when the batch has no room for another host", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:               []*fleet.Host{hostA, hostB},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			profilesToSend:      []*fleet.MDMAndroidProfilePayload{{HostUUID: "host-b", ProfileUUID: "profile-1", ProfileName: "Profile 1"}},
		}

		// run the cron with a batch size of 1, host B's profile patch should include its app list and host A, which has no
		// profile changes, should wait for the next run
		err := reconcile(1)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-b", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
		require.Equal(t, []androidmgmt.PoliciesPatchOpts{{}}, state.sentOpts)
	})

	t.Run("a host whose request failed less than an hour ago gets a new app in scope right away without counting a retry", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusRequestFailed, false, time.Now().Add(-10*time.Minute)),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {
				appOne,
				{AdamID: "com.example.two", AppTeamID: secondVersionID},
			}},
			retries: 2,
		}

		// run the cron, host A should get both apps
		err := reconcile(0)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-a", packages: []string{"com.example.one", "com.example.two", "com.fleetdm.agent"}}}, state.sent)
		require.Zero(t, state.recordedRetries)
	})

	t.Run("a quota error stops the run", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:                []*fleet.Host{hostA, hostB},
			appsInScopeByHostID:  map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}},
			setAppsErrByHostUUID: map[string]error{"host-a": &googleapi.Error{Code: http.StatusTooManyRequests}},
		}

		// run the cron, it should stop after host A, host B should not be sent and nothing should be recorded
		err := reconcile(0)
		require.NoError(t, err)
		require.Len(t, state.sent, 1)
		require.Empty(t, state.upserted)
	})

	t.Run("a Fleet agent that can't be built sends nothing for the host", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts:               []*fleet.Host{hostA},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}},
			agentErr:            sql.ErrConnDone,
		}

		// run the cron, nothing should be sent or recorded
		err := reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Empty(t, state.upserted)
	})

	t.Run("a host with an app whose configuration uses a value the host doesn't have gets no request and the app is upserted as failed", func(t *testing.T) {
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {
				appOne,
				{AdamID: "com.example.vital", AppTeamID: secondVersionID},
			}},
			secondVersionConfig: []byte(`{"managedConfiguration":{"tag":"$FLEET_HOST_VITAL_7"}}`),
			expandErr:           &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{7}, MissingNames: []string{"Host vital 1"}},
		}

		// run the cron, host A should get no request, and only the vital app should be upserted, as failed with no policy
		// version
		err := reconcile(0)
		require.NoError(t, err)
		require.Empty(t, state.sent)
		require.Len(t, state.upserted, 1)
		require.Equal(t, "com.example.vital", state.upserted[0].AdamID)
		require.Equal(t, fleet.AndroidSoftwareInstallStatusFailed, state.upserted[0].Status)
		require.Nil(t, state.upserted[0].InPolicyVersion)
	})

	t.Run("a full batch sends to the first host that needs a list, skipping a host with no changes, and the next run sends the rest", func(t *testing.T) {
		hostC := &fleet.Host{ID: 3, UUID: "host-c", Platform: "android"}
		*state = androidAppReconcileState{
			hosts: []*fleet.Host{hostA, hostB, hostC},
			installs: []*fleet.HostAndroidSoftwareInstall{
				appOneInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false, time.Time{}),
				agentInstall(hostA.ID, fleet.AndroidSoftwareInstallStatusVerified, false),
			},
			appsInScopeByHostID: map[uint][]fleet.VPPAppTeam{hostA.ID: {appOne}, hostB.ID: {appOne}, hostC.ID: {appOne}},
		}

		// run the cron with a batch size of 1, host A has no changes so host B should get the run's only list
		err := reconcile(1)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-b", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)

		// save host B's apps as sent and run the cron again, host C should get its list
		state.installs = append(state.installs,
			appOneInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusPending, false, time.Time{}),
			agentInstall(hostB.ID, fleet.AndroidSoftwareInstallStatusPending, false),
		)
		state.sent = nil
		err = reconcile(1)
		require.NoError(t, err)
		require.Equal(t, []androidAppReconcileSentList{{hostUUID: "host-c", packages: []string{"com.example.one", "com.fleetdm.agent"}}}, state.sent)
	})
}
