package service

// Apple recovery lock password tests for the MDM suite.
//
// Belongs here: the recovery lock password setting in app config and fleet config, and the end-to-end
// flow of setting, rotating, and viewing a host's recovery lock password.
//
// Does not belong here: disk encryption keys (integration_mdm_disk_encryption_test.go) and
// lock/wipe commands (integration_mdm_commands_test.go).

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestAppConfigMDMRecoveryLockPassword() {
	t := s.T()

	// by default, recovery lock password is disabled
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableRecoveryLockPassword.Value)

	// enable recovery lock password
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_recovery_lock_password": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableRecoveryLockPassword.Value)
	enabledActID := s.lastActivityMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

	// check that it's returned by GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableRecoveryLockPassword.Value)

	// patch without specifying recovery lock password should not alter it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"org_info": { "org_name": "test-org" }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableRecoveryLockPassword.Value)
	// verify no new recovery lock password activity was created
	s.lastActivityMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
		``, enabledActID)

	// patch with same value should not create new activity
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_recovery_lock_password": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableRecoveryLockPassword.Value)
	s.lastActivityMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
		``, enabledActID)

	// disable recovery lock password
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_recovery_lock_password": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableRecoveryLockPassword.Value)
	s.lastActivityMatches(fleet.ActivityTypeDisabledRecoveryLockPasswords{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

	// check that it's returned by GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableRecoveryLockPassword.Value)
}

func (s *integrationMDMTestSuite) TestTeamsMDMRecoveryLockPassword() {
	t := s.T()

	// create a team
	teamName := t.Name()
	team := &fleet.Team{
		Name:        teamName,
		Description: "team for testing recovery lock password",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// enable recovery lock password via modify team endpoint
	var modResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableRecoveryLockPassword: optjson.SetBool(true),
		},
	}, http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.MDM.EnableRecoveryLockPassword)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, team.ID, teamName, team.ID, teamName), 0)

	// check it's returned by GET
	teamResp := getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableRecoveryLockPassword)

	// patch with same value should not create new activity
	lastActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
		``, 0)
	modResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableRecoveryLockPassword: optjson.SetBool(true),
		},
	}, http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.MDM.EnableRecoveryLockPassword)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
		``, lastActID)

	// disable recovery lock password
	modResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableRecoveryLockPassword: optjson.SetBool(false),
		},
	}, http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.MDM.EnableRecoveryLockPassword)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledRecoveryLockPasswords{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, team.ID, teamName, team.ID, teamName), 0)

	// check it's returned by GET
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableRecoveryLockPassword)

	// test via apply team specs
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableRecoveryLockPassword: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// check it's enabled
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableRecoveryLockPassword)

	// apply with recovery lock password disabled
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableRecoveryLockPassword: optjson.SetBool(false),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// check it's disabled
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableRecoveryLockPassword)

	// create a new team via spec with recovery lock password enabled
	newTeamName := teamName + "2"
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: newTeamName,
		MDM: fleet.TeamSpecMDM{
			EnableRecoveryLockPassword: optjson.SetBool(true),
		},
	}}}
	var specResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &specResp)
	newTeamID := specResp.TeamIDsByName[newTeamName]
	require.NotZero(t, newTeamID)

	// check the new team has recovery lock password enabled
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", newTeamID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableRecoveryLockPassword)
}

// TestRecoveryLockPasswordIntegration is a comprehensive test for the recovery lock password feature.
// It tests various scenarios including:
// - MDM on, feature on/off transitions
// - Cron job sending commands and host state transitions
// - MDM command acknowledgment (verified) and error (failed) handling
// - Host details API returning correct recovery lock password information
// - Activities being reported correctly
// - Rotate password API (premium feature)
// - Feature toggled off with hosts in different states
func (s *integrationMDMTestSuite) TestRecoveryLockPasswordIntegration() {
	t := s.T()

	// Create a helper to create an Apple Silicon macOS host (required for recovery lock)
	createAppleSiliconHost := func(t *testing.T) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
		t.Helper()
		host, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		// Set the CPU type to ARM (Apple Silicon) - required for recovery lock
		host.CPUType = "arm64"
		err := s.ds.UpdateHost(t.Context(), host)
		require.NoError(t, err)
		return host, mdmClient
	}

	// Helper to run the recovery lock cron job
	runRecoveryLockCron := func(t *testing.T) {
		t.Helper()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		err := apple_mdm.SendRecoveryLockCommands(t.Context(), s.ds, s.mdmCommander, logger, s.fleetSvc.NewActivity)
		require.NoError(t, err)
	}

	// Helper that walks the two-command handshake a recovery lock operation now takes: the
	// device acknowledges SetRecoveryLock, Fleet responds by enqueuing VerifyRecoveryLock,
	// and only once the device acknowledges that too does the staged password become the
	// host's active one. Returns the SetRecoveryLock command that was acknowledged.
	ackSetThenVerify := func(t *testing.T, mdmClient *mdmtest.TestAppleMDMClient) *mdm.Command {
		t.Helper()
		setCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, setCmd)
		require.Equal(t, "SetRecoveryLock", setCmd.Command.RequestType)

		// The verify command is enqueued while this acknowledgment is being handled, so it
		// may not ride back on that same response; fall back to the next check-in.
		verifyCmd, err := mdmClient.Acknowledge(setCmd.CommandUUID)
		require.NoError(t, err)
		if verifyCmd == nil {
			verifyCmd, err = mdmClient.Idle()
			require.NoError(t, err)
		}
		require.NotNil(t, verifyCmd, "acknowledging SetRecoveryLock should enqueue VerifyRecoveryLock")
		require.Equal(t, "VerifyRecoveryLock", verifyCmd.Command.RequestType)

		// An acknowledgment alone isn't a pass: the verdict rides in PasswordVerified.
		_, err = mdmClient.AcknowledgeVerifyRecoveryLock(verifyCmd.CommandUUID, true)
		require.NoError(t, err)
		return setCmd
	}

	// Helper to get host details and return recovery lock password status
	getHostRecoveryLockStatus := func(hostID uint) *fleet.HostMDMRecoveryLockPassword {
		var getHostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &getHostResp)
		return &getHostResp.Host.MDM.OSSettings.RecoveryLockPassword
	}

	// archivedSetCommandUUIDs returns the SetRecoveryLock commands whose passwords Fleet
	// cannot rule out as the one the device holds, newest first. The archive is a DB-only
	// backstop with no read path, so this reads it directly.
	archivedSetCommandUUIDs := func(t *testing.T, hostUUID string) []string {
		t.Helper()
		var cmdUUIDs []string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(t.Context(), q, &cmdUUIDs, `
				SELECT COALESCE(set_command_uuid, '')
				FROM host_recovery_key_password_archive
				WHERE host_uuid = ? ORDER BY id DESC`, hostUUID)
		})
		return cmdUUIDs
	}

	// =========================================================================
	// Test 1: MDM on, feature off - hosts should not get recovery lock password
	// =========================================================================
	t.Run("MDM on, feature off - no recovery lock password set", func(t *testing.T) {
		// Ensure recovery lock password is disabled (default)
		acResp := appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.False(t, acResp.MDM.EnableRecoveryLockPassword.Value)

		host, _ := createAppleSiliconHost(t)

		// Run cron - should not send any commands
		runRecoveryLockCron(t)

		// Verify host has no recovery lock password status
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status, "status should be nil when feature is disabled")
		assert.False(t, rlpStatus.PasswordAvailable, "password should not be available when feature is disabled")
	})

	// =========================================================================
	// Test 2: MDM on, feature enabled - full lifecycle
	// =========================================================================
	t.Run("MDM on, feature enabled - full lifecycle", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		// Verify activity was created for enabling recovery lock
		s.lastActivityMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
			`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

		host, mdmClient := createAppleSiliconHost(t)

		// Initial state: no recovery lock password
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status)

		// Run cron - should send SetRecoveryLock command
		runRecoveryLockCron(t)

		// Host should now be in pending state
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status, "status should be set after cron runs")
		assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status, "status should be pending")

		// Simulate the device acknowledging SetRecoveryLock and then the VerifyRecoveryLock
		// that Fleet issues to confirm the device really took the password.
		setCmd := ackSetThenVerify(t, mdmClient)

		// Host should now be in verified state
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status, "status should be verified after the verify step")
		assert.True(t, rlpStatus.PasswordAvailable)

		// A host that confirms what it holds rules out every earlier candidate, so the
		// archive settles at one row: the set command that delivered the active password.
		assert.Equal(t, []string{setCmd.CommandUUID}, archivedSetCommandUUIDs(t, host.UUID))

		// Verify activity was created for setting password
		s.lastActivityOfTypeMatches(fleet.ActivityTypeSetHostRecoveryLockPassword{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 3: MDM command failure - host marked as failed
	// =========================================================================
	t.Run("MDM command failure - host marked as failed", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron - should send SetRecoveryLock command
		runRecoveryLockCron(t)

		// Host should be in pending state
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status)

		// This error is transient, so it is retried rather than failed outright. The host is
		// only marked failed once the retry budget is exhausted.
		for attempt := 0; attempt <= maxRecoveryLockRetries; attempt++ {
			if attempt > 0 {
				// A retry clears the status, so the next cron tick re-enqueues the command.
				runRecoveryLockCron(t)
			}
			cmd, err := mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd, "attempt %d should have a command queued", attempt)
			assert.Equal(t, "SetRecoveryLock", cmd.Command.RequestType)

			_, err = mdmClient.Err(cmd.CommandUUID, []mdm.ErrorChain{
				{ErrorCode: 12066, ErrorDomain: "MCMDMErrorDomain", LocalizedDescription: "Recovery lock password could not be set"},
			})
			require.NoError(t, err)
		}

		// Host should now be in failed state
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusFailed, *rlpStatus.Status, "status should be failed after error")
		assert.Contains(t, rlpStatus.Detail, "Recovery lock password could not be set")

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 4: Get recovery lock password API
	// =========================================================================
	t.Run("Get recovery lock password API", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron and complete the set + verify handshake
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		// Get recovery lock password via API
		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		assert.NotEmpty(t, getPasswordResp.RecoveryLockPassword.Password)
		assert.NotZero(t, getPasswordResp.RecoveryLockPassword.UpdatedAt)

		// Verify activity was created for viewing password
		s.lastActivityOfTypeMatches(fleet.ActivityTypeViewedHostRecoveryLockPassword{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 5: Rotate password API (enterprise feature - initiates rotation)
	// =========================================================================
	t.Run("Rotate password API initiates rotation", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron and complete the set + verify handshake to get to verified state
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		// Get the current password
		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		// Copy the value: unmarshalling the next response reuses this pointer.
		originalPassword := *getPasswordResp.RecoveryLockPassword.Password

		// Initiate rotation
		var rotateResp rotateRecoveryLockPasswordResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password/rotate", host.ID), nil, http.StatusOK, &rotateResp)

		// While the rotation is in flight the row is no longer 'verified', so the API
		// withholds the password rather than hand back one that may already be stale.
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		assert.Nil(t, getPasswordResp.RecoveryLockPassword.Password, "password is withheld until the rotation verifies")

		// The rotation is delivered and verified the same way an initial set is
		ackSetThenVerify(t, mdmClient)

		// Password should now be different (rotated)
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		assert.NotEqual(t, originalPassword, *getPasswordResp.RecoveryLockPassword.Password, "password should be different after rotation")

		// Verify activity was created for user-triggered rotation
		s.lastActivityOfTypeMatches(fleet.ActivityTypeRotatedHostRecoveryLockPassword{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 6: Feature disabled with host in pending state
	// =========================================================================
	t.Run("Feature disabled with host in pending state", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron - host goes to pending state
		runRecoveryLockCron(t)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status)

		// Disable the feature while host is in pending state
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})

		// Run cron again - hosts in pending state are NOT claimed for clear
		// (they haven't received the SetRecoveryLock command yet)
		runRecoveryLockCron(t)

		// Host should still be in pending state
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status)

		// Now simulate the device completing the set + verify handshake
		ackSetThenVerify(t, mdmClient)

		// Host is now verified
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		// Run cron again - now the verified host should be claimed for clear
		runRecoveryLockCron(t)

		// Host should be in removing_enforcement state
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusRemovingEnforcement, *rlpStatus.Status)
	})

	// =========================================================================
	// Test 7: Feature disabled with host in verified state
	// =========================================================================
	t.Run("Feature disabled with host in verified state", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron and complete the set + verify handshake to get to verified state
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		// Disable the feature
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})

		// Run cron - should claim host for clear
		runRecoveryLockCron(t)

		// Host should be in removing_enforcement state
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusRemovingEnforcement, *rlpStatus.Status)

		// The clear is delivered as a SetRecoveryLock with an empty new password, and is
		// confirmed by its own VerifyRecoveryLock before the row is dropped.
		ackSetThenVerify(t, mdmClient)

		// Host should no longer have a recovery lock password record
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status)
		assert.False(t, rlpStatus.PasswordAvailable)
	})

	// =========================================================================
	// Test 8: Feature re-enabled with host in removing_enforcement state
	// =========================================================================
	t.Run("Feature re-enabled with host in removing_enforcement state", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron and complete the set + verify handshake to get to verified state
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		// Disable feature to trigger clear
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
		runRecoveryLockCron(t)

		// Verify host is in removing_enforcement state
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusRemovingEnforcement, *rlpStatus.Status)

		// Re-enable feature before device acknowledges clear command
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		// Run cron - should restore host to verified state
		runRecoveryLockCron(t)

		// Host should be back to verified state (restored)
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status, "host should be restored to verified when feature is re-enabled")

		// Disable recovery lock password for cleanup
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 9: Non-Apple Silicon host should not get recovery lock
	// =========================================================================
	t.Run("Non-Apple Silicon host should not get recovery lock", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		// Create Intel Mac host (NOT Apple Silicon)
		host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		host.CPUType = "x86_64" // Intel, not ARM
		err := s.ds.UpdateHost(t.Context(), host)
		require.NoError(t, err)

		// Run cron - should not send commands for Intel host
		runRecoveryLockCron(t)

		// Intel host should not have recovery lock password
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status, "Intel Mac should not get recovery lock")

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 10: Team-specific recovery lock password
	// =========================================================================
	t.Run("Team-specific recovery lock password", func(t *testing.T) {
		// Ensure global recovery lock is disabled
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})

		// Create a team with recovery lock enabled
		teamName := "RecoveryLockTeam-" + uuid.NewString()[:8]
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: teamName}, http.StatusOK, &createTeamResp)
		team := createTeamResp.Team

		// Enable recovery lock for the team
		var modTeamResp teamResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
			MDM: &fleet.TeamPayloadMDM{
				EnableRecoveryLockPassword: optjson.SetBool(true),
			},
		}, http.StatusOK, &modTeamResp)
		require.True(t, modTeamResp.Team.Config.MDM.EnableRecoveryLockPassword)

		// Verify activity was created
		s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledRecoveryLockPasswords{}.ActivityName(),
			fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, team.ID, teamName, team.ID, teamName), 0)

		// Create host and add to team
		host, mdmClient := createAppleSiliconHost(t)
		err := s.ds.AddHostsToTeam(t.Context(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// Run cron - should send SetRecoveryLock for team host
		runRecoveryLockCron(t)

		// Host should be in pending state
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status)

		// Complete the set + verify handshake
		ackSetThenVerify(t, mdmClient)

		// Verify host is verified
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		// Create another host NOT in team - should not get recovery lock
		host2, _ := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		rlpStatus2 := getHostRecoveryLockStatus(host2.ID)
		assert.Nil(t, rlpStatus2.Status, "host not in team should not get recovery lock")

		// Disable team recovery lock
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
			MDM: &fleet.TeamPayloadMDM{
				EnableRecoveryLockPassword: optjson.SetBool(false),
			},
		}, http.StatusOK, &modTeamResp)
	})

	// =========================================================================
	// Test 11: Failed host state persists error message
	// =========================================================================
	t.Run("Failed host state persists error message", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		runRecoveryLockCron(t)

		// This error is transient, so it is retried; the message is only persisted as a
		// terminal failure once the retry budget is spent.
		errorMsg := "DeviceAlreadyHasPIN"
		for attempt := 0; attempt <= maxRecoveryLockRetries; attempt++ {
			if attempt > 0 {
				runRecoveryLockCron(t)
			}
			cmd, err := mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd, "attempt %d should have a command queued", attempt)

			_, err = mdmClient.Err(cmd.CommandUUID, []mdm.ErrorChain{
				{ErrorCode: 12066, ErrorDomain: "MCMDMErrorDomain", LocalizedDescription: errorMsg},
			})
			require.NoError(t, err)
		}

		// Verify error message is captured in detail
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusFailed, *rlpStatus.Status)
		assert.Contains(t, rlpStatus.Detail, errorMsg)

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 12: Get recovery lock password for non-Apple-Silicon host returns error
	// =========================================================================
	t.Run("Get recovery lock password for non-Apple-Silicon host returns error", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		// Create Intel Mac host
		host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		host.CPUType = "x86_64"
		err := s.ds.UpdateHost(t.Context(), host)
		require.NoError(t, err)

		// Try to get recovery lock password for Intel host - should fail with validation error
		res := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusUnprocessableEntity)
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, "Apple Silicon")

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 13: Multiple hosts in batch
	// =========================================================================
	t.Run("Multiple hosts processed in batch", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		// Create multiple hosts
		var hosts []*fleet.Host
		var mdmClients []*mdmtest.TestAppleMDMClient
		for range 3 {
			h, c := createAppleSiliconHost(t)
			hosts = append(hosts, h)
			mdmClients = append(mdmClients, c)
		}

		// Run cron - all hosts should be in pending state
		runRecoveryLockCron(t)

		for _, h := range hosts {
			rlpStatus := getHostRecoveryLockStatus(h.ID)
			require.NotNil(t, rlpStatus.Status, "host %d should have status", h.ID)
			assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status)
		}

		// Complete the set + verify handshake for all hosts
		for _, c := range mdmClients {
			ackSetThenVerify(t, c)
		}

		// All hosts should be verified
		for _, h := range hosts {
			rlpStatus := getHostRecoveryLockStatus(h.ID)
			require.NotNil(t, rlpStatus.Status)
			assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)
		}

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 11: Viewing password sets auto_rotate_at
	// =========================================================================
	t.Run("Viewing password sets auto_rotate_at", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron and complete the set + verify handshake to get to verified state
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		// View the password
		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		assert.NotEmpty(t, getPasswordResp.RecoveryLockPassword.Password)

		// auto_rotate_at should be set (approximately 1 hour from now)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.AutoRotateAt, "auto_rotate_at should be set after viewing password")
		expectedRotateAt := time.Now().Add(1 * time.Hour)
		assert.WithinDuration(t, expectedRotateAt, *getPasswordResp.RecoveryLockPassword.AutoRotateAt, 2*time.Minute)

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test 12: Auto-rotation triggers after password is viewed
	// =========================================================================
	t.Run("Auto-rotation triggers after password is viewed", func(t *testing.T) {
		// Enable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron and complete the set + verify handshake to get to verified state
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		// Get the original password
		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		// Copy the value: unmarshalling the next response reuses this pointer.
		originalPassword := *getPasswordResp.RecoveryLockPassword.Password

		// Manually set auto_rotate_at to the past to trigger auto-rotation
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(),
				`UPDATE host_recovery_key_passwords SET auto_rotate_at = ? WHERE host_uuid = ?`,
				time.Now().Add(-1*time.Hour), host.UUID)
			return err
		})

		// Run the cron job - should trigger auto-rotation
		runRecoveryLockCron(t)

		// The rotation is delivered as a SetRecoveryLock and confirmed by its own
		// VerifyRecoveryLock, exactly like an initial set.
		ackSetThenVerify(t, mdmClient)

		// Password should now be different (rotated)
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		assert.NotEqual(t, originalPassword, *getPasswordResp.RecoveryLockPassword.Password, "password should be different after auto-rotation")

		// auto_rotate_at should be set again (viewing the password schedules another rotation)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.AutoRotateAt, "auto_rotate_at should be set after viewing rotated password")
		expectedRotateAt := time.Now().Add(1 * time.Hour)
		assert.WithinDuration(t, expectedRotateAt, *getPasswordResp.RecoveryLockPassword.AutoRotateAt, 2*time.Minute)

		// Verify auto-rotation activity was created (Fleet-initiated, uses same activity type as manual rotation)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeRotatedHostRecoveryLockPassword{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

		// Disable recovery lock password
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: MDM re-enrollment invalidates the stored recovery lock password
	// =========================================================================
	t.Run("re-enrollment soft-deletes stored password and cron re-SETs", func(t *testing.T) {
		// Enable recovery lock password.
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		// Run cron → pending, then complete the set + verify handshake → verified.
		runRecoveryLockCron(t)
		originalCmdUUID := ackSetThenVerify(t, mdmClient).CommandUUID

		// Host detail API reports verified.
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)
		assert.True(t, rlpStatus.PasswordAvailable)

		// Capture the original password so we can assert it changes after re-SET.
		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		// Copy the value: unmarshalling the next response reuses this pointer.
		originalPassword := *getPasswordResp.RecoveryLockPassword.Password
		require.NotEmpty(t, originalPassword)

		// Simulate device re-enrollment (MDM profile was removed, device re-enrolls).
		// macOS clears the recovery lock password on MDM profile renewal or wipe.
		require.NoError(t, mdmClient.Reenroll())

		// Host detail API should now report no recovery lock state — same as a fresh host.
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status, "status must be absent after re-enrollment; row was soft-deleted")
		assert.False(t, rlpStatus.PasswordAvailable, "stored password must not be viewable after re-enrollment")
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusNotFound)

		// Run cron again: it must see the host as eligible and enqueue a fresh SetRecoveryLock.
		runRecoveryLockCron(t)

		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status, "cron must re-enqueue SetRecoveryLock after re-enrollment")
		assert.Equal(t, fleet.RecoveryLockStatusPending, *rlpStatus.Status)
		assert.False(t, rlpStatus.PasswordAvailable, "password should only be available when verified")

		// Ack the re-enrollment's fresh command.
		freshCmd := ackSetThenVerify(t, mdmClient)
		assert.NotEqual(t, originalCmdUUID, freshCmd.CommandUUID, "re-enrollment must enqueue a new command, not replay the old one")

		// Verified again, with a different password than before the re-enrollment.
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		assert.NotEmpty(t, getPasswordResp.RecoveryLockPassword.Password)
		assert.NotEqual(t, originalPassword, *getPasswordResp.RecoveryLockPassword.Password,
			"a new password must be generated after re-enrollment; the original is no longer on the device")

		// Disable recovery lock password.
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: the case the archive exists for. The device acknowledged the set, so it is
	// holding the new password, but Fleet never got a verify through — the row ends up
	// failed with the *old* password still active. Without the archive the password now
	// on the device would be unrecoverable.
	// =========================================================================
	t.Run("a password the device may hold survives a failed verify in the archive", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		runRecoveryLockCron(t)
		setCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, setCmd)
		require.Equal(t, "SetRecoveryLock", setCmd.Command.RequestType)

		// The device took the password. Every verify Fleet sends to confirm it fails
		// transiently, until the retry budget runs out.
		cmd, err := mdmClient.Acknowledge(setCmd.CommandUUID)
		require.NoError(t, err)
		for attempt := 0; attempt <= maxRecoveryLockRetries+1; attempt++ {
			if cmd == nil {
				cmd, err = mdmClient.Idle()
				require.NoError(t, err)
			}
			if cmd == nil {
				break
			}
			require.Equal(t, "VerifyRecoveryLock", cmd.Command.RequestType)
			cmd, err = mdmClient.Err(cmd.CommandUUID, []mdm.ErrorChain{
				{ErrorCode: 12345, ErrorDomain: "SomeTransientError", LocalizedDescription: "Temporary failure"},
			})
			require.NoError(t, err)
		}

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		require.Equal(t, fleet.RecoveryLockStatusFailed, *rlpStatus.Status)

		// Fleet never confirmed it, so it never became the active password and the host
		// detail API has nothing to offer — but the device is most likely locked with it,
		// and the archive is the only place it still exists.
		assert.False(t, rlpStatus.PasswordAvailable)
		assert.Equal(t, []string{setCmd.CommandUUID}, archivedSetCommandUUIDs(t, host.UUID),
			"a password the device acknowledged must stay recoverable after the verify fails")

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: a device acknowledges VerifyRecoveryLock even when the password did not
	// match, and reports the verdict in PasswordVerified. An acknowledgment on its own
	// must not be read as a successful verification.
	// =========================================================================
	t.Run("verify acknowledged with PasswordVerified false fails the host", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)

		runRecoveryLockCron(t)
		setCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, setCmd)
		require.Equal(t, "SetRecoveryLock", setCmd.Command.RequestType)

		verifyCmd, err := mdmClient.Acknowledge(setCmd.CommandUUID)
		require.NoError(t, err)
		if verifyCmd == nil {
			verifyCmd, err = mdmClient.Idle()
			require.NoError(t, err)
		}
		require.NotNil(t, verifyCmd)
		require.Equal(t, "VerifyRecoveryLock", verifyCmd.Command.RequestType)

		// Acknowledged, but the device says the password isn't the one it holds.
		nextCmd, err := mdmClient.AcknowledgeVerifyRecoveryLock(verifyCmd.CommandUUID, false)
		require.NoError(t, err)
		assert.Nil(t, nextCmd, "a device that answered no must not be asked the same thing again")

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusFailed, *rlpStatus.Status,
			"an acknowledgment carrying PasswordVerified=false must not verify the host")
		assert.Contains(t, rlpStatus.Detail, "does not match")
		assert.False(t, rlpStatus.PasswordAvailable)

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: re-enrollment soft-deletes the row but keeps the last known password.
	// If the device did NOT actually drop its lock, the fresh SetRecoveryLock is
	// rejected with a password mismatch (MDMClientError 70). Rather than failing
	// outright, Fleet verifies the password it still has on file.
	// =========================================================================

	// Drives the fallback: re-enroll a verified host, let the cron re-SET, and have the
	// device reject that set with a password mismatch. Returns the last known password
	// (the one stored before re-enrollment) and the VerifyRecoveryLock command Fleet
	// issued in response.
	mismatchAfterReenroll := func(t *testing.T, host *fleet.Host, mdmClient *mdmtest.TestAppleMDMClient) (string, *mdm.Command) {
		t.Helper()

		// Get to verified first so Fleet has a known-good password on file.
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		// Copy the value: unmarshalling the next response reuses this pointer.
		knownPassword := *getPasswordResp.RecoveryLockPassword.Password
		require.NotEmpty(t, knownPassword)

		// Re-enrollment soft-deletes the row. The stored password survives the delete,
		// which is what makes the fallback below possible.
		require.NoError(t, mdmClient.Reenroll())

		// The cron treats the host as fresh and enqueues a SetRecoveryLock carrying only
		// a NewPassword — no CurrentPassword, since Fleet assumes the lock is gone.
		runRecoveryLockCron(t)
		setCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, setCmd)
		require.Equal(t, "SetRecoveryLock", setCmd.Command.RequestType)

		// The device still holds the pre-re-enrollment lock, so it rejects the set.
		verifyCmd, err := mdmClient.Err(setCmd.CommandUUID, []mdm.ErrorChain{
			{ErrorCode: 70, ErrorDomain: "MDMClientError", LocalizedDescription: "Existing recovery lock password not provided"},
		})
		require.NoError(t, err)
		if verifyCmd == nil {
			verifyCmd, err = mdmClient.Idle()
			require.NoError(t, err)
		}
		require.NotNil(t, verifyCmd, "a mismatch with a known password on file must enqueue VerifyRecoveryLock")
		require.Equal(t, "VerifyRecoveryLock", verifyCmd.Command.RequestType)
		// The verify must carry the last known password, not the fresh one the device
		// just rejected.
		assert.Contains(t, string(verifyCmd.Raw), knownPassword,
			"verify must be sent with the last known password")

		return knownPassword, verifyCmd
	}

	t.Run("password mismatch after re-enrollment verifies the last known password", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		knownPassword, verifyCmd := mismatchAfterReenroll(t, host, mdmClient)

		// The device confirms it still has that password.
		_, err := mdmClient.AcknowledgeVerifyRecoveryLock(verifyCmd.CommandUUID, true)
		require.NoError(t, err)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status,
			"a verified last known password must not leave the host failed")
		assert.Empty(t, rlpStatus.Detail)
		assert.True(t, rlpStatus.PasswordAvailable)

		// The password the admin can read must be the one the device actually has, not
		// the fresh one it rejected.
		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		assert.Equal(t, knownPassword, *getPasswordResp.RecoveryLockPassword.Password)

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	t.Run("password mismatch after re-enrollment fails when the last known password does not verify", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		_, verifyCmd := mismatchAfterReenroll(t, host, mdmClient)

		// The device rejects the last known password too: whatever lock it holds is one
		// Fleet doesn't know. Nothing left to try, so this is terminal.
		nextCmd, err := mdmClient.Err(verifyCmd.CommandUUID, []mdm.ErrorChain{
			{ErrorCode: 70, ErrorDomain: "MDMClientError", LocalizedDescription: "Recovery lock password not set"},
		})
		require.NoError(t, err)
		assert.Nil(t, nextCmd, "a terminal verify failure must not enqueue another command")

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusFailed, *rlpStatus.Status)
		assert.Contains(t, rlpStatus.Detail, "Recovery lock password not set")
		assert.False(t, rlpStatus.PasswordAvailable)

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: Admin-initiated MDM unenroll soft-deletes the stored recovery lock
	// =========================================================================
	t.Run("admin API unenroll soft-deletes stored recovery lock password", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		// Verified before the unenroll.
		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		// PATCH /mdm/hosts/:id/unenroll → UnenrollMDM → MDMTurnOff.
		s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/unenroll", host.ID), nil, http.StatusNoContent)

		// Host detail API now reports no recovery lock; view-password endpoint 404s.
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status, "status must be absent after admin-initiated unenroll")
		assert.False(t, rlpStatus.PasswordAvailable, "stored password must not be viewable after unenroll")
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusNotFound)

		// Direct DB row is soft-deleted.
		var deleted bool
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &deleted,
				`SELECT deleted FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		})
		assert.True(t, deleted, "row must be soft-deleted")

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: Device-initiated CheckOut soft-deletes the stored recovery lock
	// =========================================================================
	t.Run("device CheckOut soft-deletes stored recovery lock password", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		// Device sends CheckOut (user removed the MDM profile and the device voluntarily
		// signaled it). nanomdm dispatches to MDMAppleCheckinAndCommandService.CheckOut →
		// mdmLifecycle.Do(HostActionTurnOff) → MDMTurnOff → soft-delete via the helper.
		require.NoError(t, mdmClient.Checkout())

		rlpStatus = getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status, "status must be absent after device CheckOut")
		assert.False(t, rlpStatus.PasswordAvailable)
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusNotFound)

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: manual profile removal (osquery reports enrolled=false) is caught
	// by the recovery-lock cron sweep on its next tick.
	// =========================================================================
	t.Run("cron sweep soft-deletes stored recovery lock password when host reports enrolled=false", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		// Helper: read the deleted flag directly from host_recovery_key_passwords,
		// bypassing the deleted=0 filter used by the datastore readers. This lets us
		// distinguish "row soft-deleted" from "row hard-deleted" — the cron sweep must
		// only soft-delete.
		readDeleted := func() bool {
			t.Helper()
			var deleted bool
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(t.Context(), q, &deleted,
					`SELECT deleted FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
			})
			return deleted
		}

		// Simulate the osquery ingest path that detects manual profile removal: the
		// device user removed the MDM profile but no CheckOut fired. osquery reports
		// enrolled=false via SetOrUpdateMDMData. The status is still stale until the
		// recovery-lock cron runs its sweep.
		require.NoError(t, s.ds.SetOrUpdateMDMData(t.Context(), host.ID, false, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone))

		// Pre-cron precondition: the row is live (deleted=0). SetOrUpdateMDMData only
		// flipped host_mdm.enrolled; the stale verified row still exists and the API
		// still returns it. The UI guard (hostIsMdmEnrolled check) hides it on the
		// client side until the next cron tick reconciles the backend.
		assert.False(t, readDeleted(), "precondition: row must still be live before cron sweep")

		// Cron tick runs SoftDeleteRecoveryLockPasswordsForUnenrolledHosts which sweeps
		// any live row whose host_mdm.enrolled=0.
		runRecoveryLockCron(t)

		rlpStatus = getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status, "cron sweep must soft-delete rows for unenrolled hosts")
		assert.False(t, rlpStatus.PasswordAvailable)
		assert.True(t, readDeleted(), "cron sweep must soft-delete the row (deleted=1), not hard-delete it")

		// A second cron tick is idempotent — the rkp.deleted=0 guard excludes the
		// already-swept row.
		runRecoveryLockCron(t)
		rlpStatus = getHostRecoveryLockStatus(host.ID)
		assert.Nil(t, rlpStatus.Status)
		assert.True(t, readDeleted(), "row still soft-deleted after a second cron tick")

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	t.Run("a command whose host secret can't be expanded doesn't block the queue", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		var setCmd string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &setCmd,
				`SELECT pending_set_command_uuid FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		})
		require.NotEmpty(t, setCmd)

		// The staged password is gone by the time the device fetches the command.
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(),
				`UPDATE host_recovery_key_passwords SET pending_encrypted_password = NULL WHERE host_uuid = ?`, host.UUID)
			return err
		})
		followUp := uuid.NewString()
		require.NoError(t, s.mdmCommander.EnqueueCommand(t.Context(), []string{host.UUID}, fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>ProfileList</string>
	</dict>
	<key>CommandUUID</key>
	<string>%s</string>
</dict>
</plist>`, followUp)))

		var delivered []string
		cmd, err := mdmClient.Idle()
		for i := 0; err == nil && i < 10; i++ {
			if cmd != nil {
				delivered = append(delivered, cmd.CommandUUID)
				cmd, err = mdmClient.Acknowledge(cmd.CommandUUID)
			} else {
				cmd, err = mdmClient.Idle()
			}
		}
		require.NoError(t, err)
		require.NotContains(t, delivered, setCmd)
		require.Contains(t, delivered, followUp, "the command behind the failed one is still delivered")

		var status string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &status,
				`SELECT status FROM nano_command_results WHERE id = ? AND command_uuid = ?`, host.UUID, setCmd)
		})
		require.Equal(t, fleet.MDMAppleStatusError, status)

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// =========================================================================
	// Test: a host with a SetRecoveryLock in flight when the server is upgraded from 4.92.
	// 4.92 didn't track which command a row was waiting on, so those rows reach 4.93 with
	// no command UUIDs. The device's acknowledgment must still complete set -> verify.
	// =========================================================================

	// Puts the host's row into the shape 4.92 left behind: still pending, with none of the
	// command UUIDs 4.93 matches results against.
	clearRecoveryLockCommandUUIDs := func(t *testing.T, hostUUID string) {
		t.Helper()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(), `
				UPDATE host_recovery_key_passwords
				SET pending_set_command_uuid = NULL, pending_verify_command_uuid = NULL,
					set_command_uuid = NULL, verify_command_uuid = NULL
				WHERE host_uuid = ?`, hostUUID)
			return err
		})
	}

	runUpgradeRepair := func(t *testing.T) {
		t.Helper()
		tx, err := s.ds.TestPrimaryDB().BeginTx(t.Context(), nil)
		require.NoError(t, err)
		require.NoError(t, tables.Up_20261026111120(tx))
		require.NoError(t, tx.Commit())
	}

	// Acknowledges the queued SetRecoveryLock and expects Fleet to follow with a verify,
	// which the device confirms. Returns the acknowledged SetRecoveryLock.
	ackInFlightSetThenVerify := func(t *testing.T, mdmClient *mdmtest.TestAppleMDMClient) *mdm.Command {
		t.Helper()
		setCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, setCmd, "the SetRecoveryLock queued before the upgrade should still be delivered")
		require.Equal(t, "SetRecoveryLock", setCmd.Command.RequestType)

		verifyCmd, err := mdmClient.Acknowledge(setCmd.CommandUUID)
		require.NoError(t, err)
		if verifyCmd == nil {
			verifyCmd, err = mdmClient.Idle()
			require.NoError(t, err)
		}
		require.NotNil(t, verifyCmd, "acknowledging a SetRecoveryLock queued before the upgrade should enqueue VerifyRecoveryLock")
		require.Equal(t, "VerifyRecoveryLock", verifyCmd.Command.RequestType)

		_, err = mdmClient.AcknowledgeVerifyRecoveryLock(verifyCmd.CommandUUID, true)
		require.NoError(t, err)
		return setCmd
	}

	t.Run("rotation in flight at upgrade from 4.92 completes", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		// Copy the value: unmarshalling the next response reuses this pointer.
		originalPassword := *getPasswordResp.RecoveryLockPassword.Password

		// The rotation command is queued but the device is offline when the server upgrades.
		// 4.92 queued the same rotation payload, so only the row needs to look like 4.92's.
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password/rotate", host.ID), nil, http.StatusOK, &rotateRecoveryLockPasswordResponse{})
		clearRecoveryLockCommandUUIDs(t, host.UUID)

		runUpgradeRepair(t)

		setCmd := ackInFlightSetThenVerify(t, mdmClient)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)
		assert.True(t, rlpStatus.PasswordAvailable)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		newPassword := *getPasswordResp.RecoveryLockPassword.Password
		assert.NotEqual(t, originalPassword, newPassword)
		assert.Contains(t, string(setCmd.Raw), "<string>"+newPassword+"</string>", "the readable password must be the one the rotation set")

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	t.Run("first set in flight at upgrade from 4.92 completes", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)

		var setCmdUUID string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &setCmdUUID,
				`SELECT pending_set_command_uuid FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		})
		require.NotEmpty(t, setCmdUUID)

		// 4.92 stored a first password in the active slot and its queued command read it from
		// there, where 4.93 stages it in the pending slot until the device verifies it.
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(t.Context(), `
				UPDATE host_recovery_key_passwords
				SET encrypted_password = pending_encrypted_password, pending_encrypted_password = NULL
				WHERE host_uuid = ?`, host.UUID); err != nil {
				return err
			}
			_, err := q.ExecContext(t.Context(), `UPDATE nano_commands SET command = REPLACE(command, ?, ?) WHERE command_uuid = ?`,
				"$"+fleet.HostSecretPrefix+fleet.HostSecretRecoveryLockPendingPassword,
				"$"+fleet.HostSecretPrefix+fleet.HostSecretRecoveryLockPassword,
				setCmdUUID)
			return err
		})
		clearRecoveryLockCommandUUIDs(t, host.UUID)

		runUpgradeRepair(t)

		setCmd := ackInFlightSetThenVerify(t, mdmClient)
		require.Equal(t, setCmdUUID, setCmd.CommandUUID)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)
		assert.True(t, rlpStatus.PasswordAvailable)

		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		assert.Contains(t, string(setCmd.Raw), "<string>"+*getPasswordResp.RecoveryLockPassword.Password+"</string>",
			"the readable password must be the one the first set delivered")

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	t.Run("rotation acknowledged before the repair from 4.92 completes", func(t *testing.T) {
		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": true},
		}, http.StatusOK, &appConfigResponse{})

		host, mdmClient := createAppleSiliconHost(t)
		runRecoveryLockCron(t)
		ackSetThenVerify(t, mdmClient)

		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password/rotate", host.ID), nil, http.StatusOK, &rotateRecoveryLockPasswordResponse{})
		clearRecoveryLockCommandUUIDs(t, host.UUID)

		// The device applies the rotation, and the result is dropped: the host is stuck.
		setCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, setCmd)
		require.Equal(t, "SetRecoveryLock", setCmd.Command.RequestType)
		next, err := mdmClient.Acknowledge(setCmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, next)
		runRecoveryLockCron(t)
		next, err = mdmClient.Idle()
		require.NoError(t, err)
		require.Nil(t, next)

		runUpgradeRepair(t)
		runRecoveryLockCron(t)

		verifyCmd, err := mdmClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, verifyCmd, "the cron should send the verify the dropped result never triggered")
		require.Equal(t, "VerifyRecoveryLock", verifyCmd.Command.RequestType)
		_, err = mdmClient.AcknowledgeVerifyRecoveryLock(verifyCmd.CommandUUID, true)
		require.NoError(t, err)

		rlpStatus := getHostRecoveryLockStatus(host.ID)
		require.NotNil(t, rlpStatus.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *rlpStatus.Status)

		var getPasswordResp getHostRecoveryLockPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/recovery_lock_password", host.ID), nil, http.StatusOK, &getPasswordResp)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword)
		require.NotNil(t, getPasswordResp.RecoveryLockPassword.Password)
		assert.Contains(t, string(setCmd.Raw), "<string>"+*getPasswordResp.RecoveryLockPassword.Password+"</string>",
			"the readable password must be the one the rotation set")

		// Nothing is left for the cron to send.
		runRecoveryLockCron(t)
		next, err = mdmClient.Idle()
		require.NoError(t, err)
		require.Nil(t, next)

		s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
			"mdm": map[string]any{"enable_recovery_lock_password": false},
		}, http.StatusOK, &appConfigResponse{})
	})

	// Final cleanup: ensure recovery lock is disabled
	s.DoJSON("PATCH", "/api/latest/fleet/config", map[string]any{
		"mdm": map[string]any{"enable_recovery_lock_password": false},
	}, http.StatusOK, &appConfigResponse{})
}
