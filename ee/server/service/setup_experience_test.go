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

// TestSetupExperienceNextStepPolicyGated covers the policy-gated (Windows/Linux) branch of SetupExperienceNextStep: the policy is
// used only as a gate (pass -> skip, fail -> install via the normal ForSetupExperience path), the item is held running while
// awaiting a fresh result, an out-of-scope gating policy falls back to installing, and the host policy clock is reset once when a
// gated setup finishes (not per gated result).
func TestSetupExperienceNextStepPolicyGated(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	hostUUID := "win-osquery"
	installerID := uint(7)
	policyID := uint(99)

	hostLastEnrolledAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	labelsReportedAt := hostLastEnrolledAt.Add(time.Minute) // labels reported after enrollment (the common, ready case)
	host := &fleet.Host{
		ID:             42,
		UUID:           "win-uuid",
		OsqueryHostID:  ptr.String(hostUUID),
		Platform:       "windows",
		LastEnrolledAt: hostLastEnrolledAt,
		LabelUpdatedAt: labelsReportedAt,
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
		// Assert the gating freshness plumbing forwards the right host, policy, and this-enrollment cutoff.
		require.Equal(t, host.ID, hostID)
		require.Equal(t, policyID, gotPolicyID)
		require.Equal(t, hostLastEnrolledAt, since)
		return policyResult, nil
	}
	var deliverable map[string]string
	ds.PolicyQueriesForHostFilteredFunc = func(ctx context.Context, host *fleet.Host, policyIDs []uint) (map[string]string, error) {
		return deliverable, nil
	}
	// The host policy clock is reset once, when a gated setup finishes (so the host's other policies re-run promptly post-setup).
	ds.ClearHostPolicyUpdatedAtFunc = func(ctx context.Context, hostID uint) error { return nil }

	reset := func() {
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false
		ds.ClearHostPolicyUpdatedAtFuncInvoked = false
		ds.PolicyQueriesForHostFilteredFuncInvoked = false
		ds.GetSetupExperiencePolicyResultFuncInvoked = false
		installs = nil
		updates = nil
		policyResult = nil
		deliverable = nil
		host.LabelUpdatedAt = labelsReportedAt // default to labels-ready; the "labels not reported yet" subtest overrides this
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
		deliverable = map[string]string{"99": "SELECT 1;"} // in scope

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked, "passing policy must not install")
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, updates[0].Status)
		require.Nil(t, updates[0].HostSoftwareInstallsExecutionID)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "clock is reset at setup completion, not per gated result")
	})

	t.Run("policy fails -> install via ForSetupExperience path (no PolicyID on the install)", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = &policyFails
		deliverable = map[string]string{"99": "SELECT 1;"} // in scope

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.Len(t, installs, 1)
		require.True(t, installs[0].ForSetupExperience, "gated install must run as a setup-experience install")
		require.Nil(t, installs[0].PolicyID, "setup experience owns the install; it must not be a policy-automation install")
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[0].Status)
		require.NotNil(t, updates[0].HostSoftwareInstallsExecutionID)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "clock is reset at setup completion, not per gated result")
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
	})

	t.Run("no result, policy out of scope (labels ready) -> falls back to installing", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = nil
		deliverable = map[string]string{} // out of scope: not deliverable (and labels are reported, so this is definitive)

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.True(t, ds.PolicyQueriesForHostFilteredFuncInvoked, "labels are ready, so scope is evaluated")
		require.Len(t, installs, 1)
		require.True(t, installs[0].ForSetupExperience)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[len(updates)-1].Status)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "out-of-scope fallback ran no gating policy; policy clock must not be reset")
	})

	t.Run("result present but policy out of scope -> installs, result not consulted (exclude-label edge)", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = &policyPasses      // a passing result was reported (e.g. before the exclude-label membership was computed)...
		deliverable = map[string]string{} // ...but with labels computed the policy is now out of scope for this host

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.GetSetupExperiencePolicyResultFuncInvoked, "scope is checked before the result; an out-of-scope policy's result must not be consulted")
		require.Len(t, installs, 1, "an out-of-scope policy must not gate the install even if it reported a pass")
		require.True(t, installs[0].ForSetupExperience)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[len(updates)-1].Status)
		require.NotNil(t, updates[len(updates)-1].HostSoftwareInstallsExecutionID)
	})

	t.Run("no result, labels not reported yet -> stays running (no premature out-of-scope install)", func(t *testing.T) {
		reset()
		items = gatedPending()
		policyResult = nil
		deliverable = map[string]string{}                        // would look out of scope...
		host.LabelUpdatedAt = hostLastEnrolledAt.Add(-time.Hour) // ...but the host hasn't reported labels for this enrollment yet

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.PolicyQueriesForHostFilteredFuncInvoked, "must not evaluate scope until labels are reported")
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked, "must not fail-open before labels are computed")
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusRunning, updates[0].Status)
		require.Nil(t, updates[0].HostSoftwareInstallsExecutionID)
	})

	t.Run("un-gated software is started before policy-gated", func(t *testing.T) {
		reset()
		ungatedInstallerID := uint(8)
		// Gated item is first (lower id / earlier alphabetically); the un-gated item is second. Un-gated-first selection must
		// still start the un-gated install so a gated item's policy/label wait does not delay it.
		items = []*fleet.SetupExperienceStatusResult{
			{HostUUID: hostUUID, Name: "A-GatedApp", SoftwareInstallerID: &installerID, PolicyID: &policyID, Status: fleet.SetupExperienceStatusPending},
			{HostUUID: hostUUID, Name: "Z-UngatedApp", SoftwareInstallerID: &ungatedInstallerID, Status: fleet.SetupExperienceStatusPending},
		}

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.GetSetupExperiencePolicyResultFuncInvoked, "the gated item must not be evaluated before the un-gated install starts")
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Len(t, updates, 1)
		require.Equal(t, "Z-UngatedApp", updates[0].Name, "the un-gated item must be the one started first")
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
		policyResult = &policyPasses                       // result now available
		deliverable = map[string]string{"99": "SELECT 1;"} // in scope

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.False(t, finished)
		require.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Len(t, updates, 1)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, updates[0].Status)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "clock is reset at setup completion, not per gated result")
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
	})

	t.Run("setup finishes with a gated item -> host policy clock reset once", func(t *testing.T) {
		reset()
		// All items terminal (the gated item resolved on a prior poll); this poll reaches the "finished" branch.
		items = []*fleet.SetupExperienceStatusResult{{
			HostUUID:            hostUUID,
			Name:                "GatedApp",
			SoftwareInstallerID: &installerID,
			PolicyID:            &policyID,
			Status:              fleet.SetupExperienceStatusSuccess,
		}}

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.True(t, finished, "all items terminal -> setup experience finished")
		require.True(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "a gated setup that finished must reset the host policy clock once")
	})

	t.Run("setup finishes with no gated item -> host policy clock not reset", func(t *testing.T) {
		reset()
		// A terminal, un-gated setup item (PolicyID nil): finishing must not touch the policy clock.
		items = []*fleet.SetupExperienceStatusResult{{
			HostUUID:            hostUUID,
			Name:                "UngatedApp",
			SoftwareInstallerID: &installerID,
			Status:              fleet.SetupExperienceStatusSuccess,
		}}

		finished, err := svc.SetupExperienceNextStep(ctx, host)
		require.NoError(t, err)
		require.True(t, finished)
		require.False(t, ds.ClearHostPolicyUpdatedAtFuncInvoked, "no gated item -> policy clock must not be reset")
	})
}
