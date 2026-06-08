package service

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupExperienceNextStep(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	requestedInstalls := make(map[uint][]uint)
	requestedUpdateSetupExperience := []*fleet.SetupExperienceStatusResult{}
	requestedScriptExecution := []*fleet.HostScriptRequestPayload{}
	resetIndicators := func() {
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		ds.InsertHostVPPSoftwareInstallFuncInvoked = false
		ds.NewHostScriptExecutionRequestFuncInvoked = false
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false
		clear(requestedInstalls)
		requestedUpdateSetupExperience = []*fleet.SetupExperienceStatusResult{}
		requestedScriptExecution = []*fleet.HostScriptRequestPayload{}
	}

	host1UUID := "123"
	host1ID := uint(1)
	installerID1 := uint(2)
	scriptID1 := uint(3)
	scriptContentID1 := uint(4)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				EnabledAndConfigured: true,
			},
		}, nil
	}

	ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
		return true, nil
	}

	var mockListSetupExperience []*fleet.SetupExperienceStatusResult
	ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hostUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
		return mockListSetupExperience, nil
	}

	var mockListHostsLite []*fleet.Host
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		return mockListHostsLite, nil
	}

	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		requestedInstalls[hostID] = append(requestedInstalls[hostID], softwareInstallerID)
		return "install-uuid", nil
	}

	ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
		requestedUpdateSetupExperience = append(requestedUpdateSetupExperience, status)
		return nil
	}

	ds.NewHostScriptExecutionRequestFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
		requestedScriptExecution = append(requestedScriptExecution, request)
		return &fleet.HostScriptResult{
			ExecutionID: "script-uuid",
		}, nil
	}

	mockListHostsLite = append(mockListHostsLite, &fleet.Host{UUID: host1UUID, ID: host1ID})

	finished, err := svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)
	assert.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.False(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	resetIndicators()

	// Only installer queued
	mockListSetupExperience = []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:            host1UUID,
			SoftwareInstallerID: &installerID1,
			Status:              fleet.SetupExperienceStatusPending,
		},
	}

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedInstalls, 1)
	assert.Len(t, requestedUpdateSetupExperience, 1)
	assert.Equal(t, "install-uuid", *requestedUpdateSetupExperience[0].HostSoftwareInstallsExecutionID)

	mockListSetupExperience[0].Status = fleet.SetupExperienceStatusSuccess
	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)

	resetIndicators()

	// TODO VPP app queueing is better done in an integration
	// test, the setup required would be too much

	// Only script queued
	mockListSetupExperience = []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:                host1UUID,
			SetupExperienceScriptID: &scriptID1,
			ScriptContentID:         &scriptContentID1,
			Status:                  fleet.SetupExperienceStatusPending,
		},
	}

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.True(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedScriptExecution, 1)
	assert.Len(t, requestedUpdateSetupExperience, 1)
	assert.Equal(t, "script-uuid", *requestedUpdateSetupExperience[0].ScriptExecutionID)

	mockListSetupExperience[0].Status = fleet.SetupExperienceStatusSuccess
	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)

	resetIndicators()

	// Both installer and script
	mockListSetupExperience = []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:            host1UUID,
			SoftwareInstallerID: &installerID1,
			Status:              fleet.SetupExperienceStatusPending,
		},
		{
			HostUUID:                host1UUID,
			SetupExperienceScriptID: &scriptID1,
			ScriptContentID:         &scriptContentID1,
			Status:                  fleet.SetupExperienceStatusPending,
		},
	}

	// Only installer is queued
	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedInstalls, 1)
	assert.Len(t, requestedScriptExecution, 0)
	assert.Len(t, requestedUpdateSetupExperience, 1)

	// install finished, call it again. This time script is queued
	mockListSetupExperience[0].Status = fleet.SetupExperienceStatusSuccess

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.True(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedInstalls, 1)
	assert.Len(t, requestedScriptExecution, 1)
	assert.Len(t, requestedUpdateSetupExperience, 2)

	// both finished, now we're done
	mockListSetupExperience[1].Status = fleet.SetupExperienceStatusFailure

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)
}

func TestSetupExperienceSetWithManualAgentInstall(t *testing.T) {
	ctx := test.UserContext(context.Background(), test.UserAdmin)
	ds := new(mock.Store)
	svc, baseSvc := newTestServiceWithMock(t, ds)

	appConfig := fleet.AppConfig{}
	team := fleet.TeamLite{}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &appConfig, nil
	}

	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		return &team, nil
	}

	ds.SetSetupExperienceSoftwareTitlesFunc = func(ctx context.Context, platform string, teamID uint, titleIDs []uint) error {
		return nil
	}

	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}

	ds.SetSetupExperienceScriptFunc = func(ctx context.Context, script *fleet.Script) error {
		return nil
	}

	baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	// No manual agent install, we good
	// No team
	err := svc.SetSetupExperienceSoftware(ctx, "darwin", 0, []uint{1, 2})
	require.NoError(t, err)

	scriptReader := bytes.NewReader([]byte("hello"))
	err = svc.SetSetupExperienceScript(ctx, nil, "potato.sh", scriptReader)
	require.NoError(t, err)
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// Team
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 1, []uint{1, 2})
	require.NoError(t, err)

	err = svc.SetSetupExperienceScript(ctx, ptr.Uint(1), "potato.sh", scriptReader)
	require.NoError(t, err)
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// Manual agent install
	appConfig.MDM.MacOSSetup.ManualAgentInstall = optjson.SetBool(true)
	team.Config.MDM.MacOSSetup.ManualAgentInstall = optjson.SetBool(true)

	// No team
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 0, []uint{1, 2})
	require.ErrorContains(t, err, "Couldn’t add setup experience software. To add software, first disable macos_manual_agent_install.")

	err = svc.SetSetupExperienceScript(ctx, nil, "potato.sh", scriptReader)
	require.ErrorContains(t, err, "Couldn’t add setup experience script. To add script, first disable macos_manual_agent_install.")
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// Team
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 1, []uint{1, 2})
	require.ErrorContains(t, err, "Couldn’t add setup experience software. To add software, first disable macos_manual_agent_install.")

	err = svc.SetSetupExperienceScript(ctx, ptr.Uint(1), "potato.sh", scriptReader)
	require.ErrorContains(t, err, "Couldn’t add setup experience script. To add script, first disable macos_manual_agent_install.")
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// We can still set software to none though
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 0, []uint{})
	require.NoError(t, err)

	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 1, []uint{})
	require.NoError(t, err)

	t.Run("should not block for non darwin hosts", func(t *testing.T) {
		for _, platform := range fleet.SetupExperienceSupportedPlatforms {
			if platform == "darwin" {
				continue
			}

			err := svc.SetSetupExperienceSoftware(ctx, platform, 0, []uint{1, 2})
			require.NoError(t, err)
		}
	})
}

// fakePolicyReportClock records the host IDs whose async "policies last reported" Redis epoch was reset, so tests can assert the
// epoch is cleared exactly when a real gating result is consumed.
type fakePolicyReportClock struct{ resetHostIDs []uint }

func (f *fakePolicyReportClock) ResetHostPolicyReportedAt(_ context.Context, hostID uint) error {
	f.resetHostIDs = append(f.resetHostIDs, hostID)
	return nil
}

// TestSetupExperienceNextStepPolicyGated covers the policy-gated (Windows/Linux) branch of SetupExperienceNextStep: the policy is
// used only as a gate (pass -> skip, fail -> install via the normal ForSetupExperience path), the item is held running while
// awaiting a fresh result, and an out-of-scope gating policy falls back to installing.
func TestSetupExperienceNextStepPolicyGated(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	policyClock := &fakePolicyReportClock{}
	svc.policyReportClock = policyClock

	hostUUID := "win-osquery"
	installerID := uint(7)
	policyID := uint(99)

	host := &fleet.Host{
		ID:            42,
		UUID:          "win-uuid",
		OsqueryHostID: ptr.String(hostUUID),
		Platform:      "windows",
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) { return true, nil }

	var items []*fleet.SetupExperienceStatusResult
	ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hostUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
		return items, nil
	}

	var installs []fleet.HostSoftwareInstallOptions
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		installs = append(installs, opts)
		return "gated-install-uuid", nil
	}
	var updates []*fleet.SetupExperienceStatusResult
	ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
		updates = append(updates, status)
		return nil
	}

	// Defaults; individual subtests override.
	policyPasses, policyFails := true, false
	var policyResult *bool
	ds.GetSetupExperiencePolicyResultFunc = func(ctx context.Context, hostID, gotPolicyID uint, since time.Time) (*bool, error) {
		return policyResult, nil
	}
	var deliverable map[string]string
	ds.PolicyQueriesForHostFilteredFunc = func(ctx context.Context, host *fleet.Host, policyIDs []uint) (map[string]string, error) {
		return deliverable, nil
	}
	// Resetting the host policy clock after a gating result is consumed (so the host's other policies re-run promptly post-setup).
	ds.ClearHostPolicyUpdatedAtFunc = func(ctx context.Context, hostID uint) error { return nil }

	reset := func() {
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false
		ds.ClearHostPolicyUpdatedAtFuncInvoked = false
		installs = nil
		updates = nil
		policyResult = nil
		deliverable = nil
		policyClock.resetHostIDs = nil
	}

	gatedPending := func() []*fleet.SetupExperienceStatusResult {
		return []*fleet.SetupExperienceStatusResult{{
			HostUUID:            hostUUID,
			Name:                "GatedApp",
			SoftwareInstallerID: &installerID,
			PolicyID:            &policyID,
			Status:              fleet.SetupExperienceStatusPending,
		}}
	}

	t.Run("policy passes -> skipped (success, no install)", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = &policyPasses

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked, "passing policy must not install")
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, updates[0].Status)
		require.Nil(t, updates[0].HostSoftwareInstallsExecutionID)
		require.True(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "consuming a gating result must reset the host policy clock")
		require.Equal(t, []uint{host.ID}, policyClock.resetHostIDs, "consuming a gating result must also clear the async redis epoch")
	})

	t.Run("policy fails -> install via ForSetupExperience path (no PolicyID on the install)", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = &policyFails

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.Len(t, installs, 1)
		require.True(t, installs[0].ForSetupExperience, "gated install must run as a setup-experience install")
		require.Nil(t, installs[0].PolicyID, "setup experience owns the install; it must not be a policy-automation install")
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[0].Status)
		require.NotNil(t, updates[0].HostSoftwareInstallsExecutionID)
		require.True(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "consuming a gating result must reset the host policy clock")
		require.Equal(t, []uint{host.ID}, policyClock.resetHostIDs, "consuming a gating result must also clear the async redis epoch")
	})

	t.Run("no result yet, policy in scope -> stays running, no install", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = nil
		deliverable = map[string]string{"99": "SELECT 1;"} // in scope

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[0].Status)
		require.Nil(t, updates[0].HostSoftwareInstallsExecutionID)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "no result consumed yet; policy clock must not be reset")
		require.Empty(t, policyClock.resetHostIDs, "no result consumed yet; redis epoch must not be cleared")
	})

	t.Run("no result, policy out of scope -> falls back to installing", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = nil
		deliverable = map[string]string{} // out of scope: not deliverable

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.Len(t, installs, 1)
		require.True(t, installs[0].ForSetupExperience)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[len(updates)-1].Status)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "out-of-scope fallback ran no gating policy; policy clock must not be reset")
		require.Empty(t, policyClock.resetHostIDs, "out-of-scope fallback ran no gating policy; redis epoch must not be cleared")
	})

	t.Run("running gated item awaiting policy is re-checked each poll", func(t *testing.T) {
		reset()
		// Already running, no install execution yet -> awaiting-policy phase.
		items = []*fleet.SetupExperienceStatusResult{{
			HostUUID:            hostUUID,
			Name:                "GatedApp",
			SoftwareInstallerID: &installerID,
			PolicyID:            &policyID,
			Status:              fleet.SetupExperienceStatusRunning,
		}}
		policyResult = &policyPasses // result now available

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, updates[0].Status)
		require.Equal(t, []uint{host.ID}, policyClock.resetHostIDs, "consuming a gating result must also clear the async redis epoch")
	})

	t.Run("already-running awaiting item with no result yet does not write again", func(t *testing.T) {
		reset()
		// Already running, no install execution yet, and still no policy result -> nothing changed, so we must not re-persist
		// the same running state on every poll (avoids write amplification while waiting).
		items = []*fleet.SetupExperienceStatusResult{{
			HostUUID:            hostUUID,
			Name:                "GatedApp",
			SoftwareInstallerID: &installerID,
			PolicyID:            &policyID,
			Status:              fleet.SetupExperienceStatusRunning,
		}}
		policyResult = nil
		deliverable = map[string]string{"99": "SELECT 1;"} // in scope, just not reported yet

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.False(t, ds.UpdateSetupExperienceStatusResultFuncInvoked, "must not re-write unchanged running state on every poll")
		require.Empty(t, policyClock.resetHostIDs, "no result consumed; redis epoch must not be cleared")
	})
}
