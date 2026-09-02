package mysql

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	mathrand "math/rand/v2"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryLockPassword(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"RecoveryLockPasswordSetAndGet", testRecoveryLockPasswordSetAndGet},
		{"RecoveryLockPasswordBulkSet", testRecoveryLockPasswordBulkSet},
		{"RecoveryLockPasswordGetNotFound", testRecoveryLockPasswordGetNotFound},
		{"RecoveryLockPasswordSetOverwrite", testRecoveryLockPasswordSetOverwrite},
		{"RecoveryLockPasswordUpdatedAtChanges", testRecoveryLockPasswordUpdatedAtChanges},
		{"RecoveryLockStatusMethods", testRecoveryLockStatusMethods},
		{"GetHostsForRecoveryLockAction", testGetHostsForRecoveryLockAction},
		{"GetHostRecoveryLockPasswordStatus", testGetHostRecoveryLockPasswordStatus},
		{"ClaimHostsForRecoveryLockClear", testClaimHostsForRecoveryLockClear},
		{"RecoveryLockRotation", testRecoveryLockRotation},
		{"RecoveryLockAutoRotation", testRecoveryLockAutoRotation},
		{"RecoveryLockResetOnMDMReEnrollment", testRecoveryLockResetOnMDMReEnrollment},
		{"DeleteHostPreservesRecoveryLockPassword", testDeleteHostPreservesRecoveryLockPassword},
		{"HostRecoveryLockStatusMatrix", testHostRecoveryLockStatusMatrix},
		{"RecoveryLockReadersReturnNotFoundForSoftDeleted", testRecoveryLockReadersReturnNotFoundForSoftDeleted},
		{"MDMTurnOffSoftDeletesRecoveryLockPassword", testMDMTurnOffSoftDeletesRecoveryLockPassword},
		{"RecoveryLockPasswordArchive", testRecoveryLockPasswordArchive},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			t.Helper()
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testRecoveryLockPasswordSetAndGet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "test-host-1", "1.2.3.4", "h1key", "h1uuid", time.Now())

	// Generate and set password
	password := apple_mdm.GenerateRecoveryLockPassword()

	cmdUUID := uuid.NewString()
	err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: password, PendingSetCommandUUID: cmdUUID}})
	require.NoError(t, err)

	// Get the pending_password and verify it matches
	var lockResult struct {
		EncryptedPassword        []byte    `db:"encrypted_password"`
		PendingEncryptedPassword []byte    `db:"pending_encrypted_password"`
		UpdatedAt                time.Time `db:"updated_at"`
		PendingSetCommandUUID    string    `db:"pending_set_command_uuid"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &lockResult, `
			SELECT encrypted_password, pending_encrypted_password, updated_at,
			pending_set_command_uuid
			FROM host_recovery_key_passwords
			WHERE host_uuid = ? AND deleted = 0
		`, host.UUID)
	})
	require.NoError(t, err)
	require.Nil(t, lockResult.EncryptedPassword)
	require.NotNil(t, lockResult.PendingEncryptedPassword)
	decrypted, err := decrypt(lockResult.PendingEncryptedPassword, ds.serverPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, password, string(decrypted))
	assert.False(t, lockResult.UpdatedAt.IsZero())
	assert.Equal(t, cmdUUID, lockResult.PendingSetCommandUUID)
}

func testRecoveryLockPasswordBulkSet(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create multiple hosts
	host1 := test.NewHost(t, ds, "bulk-host-1", "1.2.3.10", "bulk1key", "bulk1uuid", time.Now())
	host2 := test.NewHost(t, ds, "bulk-host-2", "1.2.3.11", "bulk2key", "bulk2uuid", time.Now())
	host3 := test.NewHost(t, ds, "bulk-host-3", "1.2.3.12", "bulk3key", "bulk3uuid", time.Now())

	// Generate passwords for all hosts
	pw1 := apple_mdm.GenerateRecoveryLockPassword()
	pw2 := apple_mdm.GenerateRecoveryLockPassword()
	pw3 := apple_mdm.GenerateRecoveryLockPassword()

	// Bulk set passwords
	err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{
		{HostUUID: host1.UUID, Password: pw1},
		{HostUUID: host2.UUID, Password: pw2},
		{HostUUID: host3.UUID, Password: pw3},
	})
	require.NoError(t, err)

	// Each password is staged as pending until the device verifies it, so it only becomes
	// the host's active password once the verify step completes.
	for _, hostUUID := range []string{host1.UUID, host2.UUID, host3.UUID} {
		staged, err := ds.GetHostRecoveryLockPassword(ctx, hostUUID)
		require.NoError(t, err)
		assert.Nil(t, staged.Password, "password should not be active before it is verified")
		markRecoveryLockVerified(t, ds, hostUUID)
	}

	// Verify all passwords are stored correctly
	result1, err := ds.GetHostRecoveryLockPassword(ctx, host1.UUID)
	require.NoError(t, err)
	require.NotNil(t, result1.Password)
	assert.Equal(t, pw1, *result1.Password)

	result2, err := ds.GetHostRecoveryLockPassword(ctx, host2.UUID)
	require.NoError(t, err)
	require.NotNil(t, result2.Password)
	assert.Equal(t, pw2, *result2.Password)

	result3, err := ds.GetHostRecoveryLockPassword(ctx, host3.UUID)
	require.NoError(t, err)
	require.NotNil(t, result3.Password)
	assert.Equal(t, pw3, *result3.Password)

	// Verify all passwords are different
	assert.NotEqual(t, pw1, pw2)
	assert.NotEqual(t, pw2, pw3)
	assert.NotEqual(t, pw1, pw3)
}

func testRecoveryLockPasswordGetNotFound(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Try to get password for non-existent host
	_, err := ds.GetHostRecoveryLockPassword(ctx, "non-existent-uuid")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testRecoveryLockPasswordSetOverwrite(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "test-host-2", "1.2.3.5", "h2key", "h2uuid", time.Now())

	// Set password first time
	password1 := apple_mdm.GenerateRecoveryLockPassword()
	err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: password1}})
	require.NoError(t, err)

	// Set password second time (should overwrite)
	password2 := apple_mdm.GenerateRecoveryLockPassword()
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: password2}})
	require.NoError(t, err)

	// Passwords should be different (randomly generated)
	assert.NotEqual(t, password1, password2)

	markRecoveryLockVerified(t, ds, host.UUID)

	// Verify only the new password is stored
	result, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, result.Password)
	assert.Equal(t, password2, *result.Password)
}

func testRecoveryLockPasswordUpdatedAtChanges(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "test-host-3", "1.2.3.6", "h3key", "h3uuid", time.Now())

	// Set password first time
	password1 := apple_mdm.GenerateRecoveryLockPassword()
	err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: password1}})
	require.NoError(t, err)

	result1, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
	require.NoError(t, err)

	// Wait a bit to ensure timestamp changes
	time.Sleep(1 * time.Second)

	// Set password second time
	password2 := apple_mdm.GenerateRecoveryLockPassword()
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: password2}})
	require.NoError(t, err)

	result2, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
	require.NoError(t, err)

	// updated_at should have changed
	assert.True(t, result2.UpdatedAt.After(result1.UpdatedAt), "updated_at should increase after overwrite")
}

func testRecoveryLockStatusMethods(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Helper to create a host with a recovery lock password (status is set to 'pending'
	// atomically). Returns the pending SetRecoveryLock command UUID, which the status
	// transitions match on.
	setupHost := func(t *testing.T, name, ip, key, hostUUID string) (*fleet.Host, string) {
		t.Helper()
		host := test.NewHost(t, ds, name, ip, key, hostUUID, time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		cmdUUID := uuid.NewString()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw, PendingSetCommandUUID: cmdUUID}})
		require.NoError(t, err)
		return host, cmdUUID
	}

	t.Run("SetHostsRecoveryLockPasswords sets pending status atomically", func(t *testing.T) {
		host, _ := setupHost(t, "atomic-pending-host", "1.2.3.6", "atomickey", "atomicuuid")

		// Verify status is pending immediately after storing password
		var status string
		err := ds.writer(ctx).GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.Equal(t, string(fleet.MDMDeliveryPending), status)
	})

	t.Run("SetRecoveryLockVerified", func(t *testing.T) {
		host, setCmdUUID := setupHost(t, "verified-host", "1.2.3.9", "verifiedkey", "verifieduuid")

		// The pending password is only promoted once the verify command is acknowledged.
		verifyCmdUUID := uuid.NewString()
		err := ds.SetRecoveryLockVerifying(ctx, host.UUID, setCmdUUID, verifyCmdUUID)
		require.NoError(t, err)
		err = ds.SetRecoveryLockVerified(ctx, host.UUID, verifyCmdUUID)
		require.NoError(t, err)

		var result struct {
			Status                   string         `db:"status"`
			EncryptedPassword        []byte         `db:"encrypted_password"`
			PendingEncryptedPassword []byte         `db:"pending_encrypted_password"`
			VerifyCommandUUID        sql.NullString `db:"verify_command_uuid"`
		}
		err = ds.writer(ctx).GetContext(ctx, &result, `
			SELECT status, encrypted_password, pending_encrypted_password, verify_command_uuid
			FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		assert.Equal(t, string(fleet.MDMDeliveryVerified), result.Status)
		assert.NotEmpty(t, result.EncryptedPassword, "pending password should be promoted to active")
		assert.Empty(t, result.PendingEncryptedPassword)
		assert.Equal(t, verifyCmdUUID, result.VerifyCommandUUID.String)
	})

	t.Run("SetRecoveryLockVerifyingLastKnownPassword keeps the active password", func(t *testing.T) {
		host, firstSetCmdUUID := setupHost(t, "last-known-host", "1.2.3.20", "lastknownkey", "lastknownuuid")

		// Establish an active password, then stage a second one the device will reject.
		markRecoveryLockVerified(t, ds, host.UUID)
		var activePassword []byte
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &activePassword,
			"SELECT encrypted_password FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID))
		require.NotEmpty(t, activePassword)

		secondSetCmdUUID := uuid.NewString()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{
			{HostUUID: host.UUID, Password: apple_mdm.GenerateRecoveryLockPassword(), PendingSetCommandUUID: secondSetCmdUUID},
		}))
		require.NotEqual(t, firstSetCmdUUID, secondSetCmdUUID)

		// The set came back with a password mismatch, so the verify goes out with the
		// active password and the rejected pending one is dropped.
		verifyCmdUUID := uuid.NewString()
		require.NoError(t, ds.SetRecoveryLockVerifyingLastKnownPassword(ctx, host.UUID, secondSetCmdUUID, verifyCmdUUID))

		var row struct {
			Status                   string         `db:"status"`
			EncryptedPassword        []byte         `db:"encrypted_password"`
			PendingEncryptedPassword []byte         `db:"pending_encrypted_password"`
			PendingSetCommandUUID    sql.NullString `db:"pending_set_command_uuid"`
			PendingVerifyCommandUUID sql.NullString `db:"pending_verify_command_uuid"`
		}
		const selectStmt = `
			SELECT status, encrypted_password, pending_encrypted_password,
			       pending_set_command_uuid, pending_verify_command_uuid
			FROM host_recovery_key_passwords WHERE host_uuid = ?`
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &row, selectStmt, host.UUID))
		assert.Equal(t, string(fleet.MDMDeliveryVerifying), row.Status)
		assert.Empty(t, row.PendingEncryptedPassword, "the rejected password must not stay staged for promotion")
		assert.Equal(t, activePassword, row.EncryptedPassword)
		assert.False(t, row.PendingSetCommandUUID.Valid)
		assert.Equal(t, verifyCmdUUID, row.PendingVerifyCommandUUID.String)

		// The device confirms the active password; with nothing pending, it stays put.
		require.NoError(t, ds.SetRecoveryLockVerified(ctx, host.UUID, verifyCmdUUID))
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &row, selectStmt, host.UUID))
		assert.Equal(t, string(fleet.MDMDeliveryVerified), row.Status)
		assert.Equal(t, activePassword, row.EncryptedPassword,
			"verifying the active password must not overwrite it with a NULL pending one")
	})

	t.Run("SetRecoveryLockVerified ignores a stale verify command", func(t *testing.T) {
		host, setCmdUUID := setupHost(t, "stale-verify-host", "1.2.3.15", "stalekey", "staleuuid")

		err := ds.SetRecoveryLockVerifying(ctx, host.UUID, setCmdUUID, uuid.NewString())
		require.NoError(t, err)
		// A result for a command the host is no longer waiting on must not promote.
		err = ds.SetRecoveryLockVerified(ctx, host.UUID, uuid.NewString())
		require.NoError(t, err)

		var status string
		err = ds.writer(ctx).GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.Equal(t, string(fleet.MDMDeliveryVerifying), status)
	})

	t.Run("SetRecoveryLockFailed", func(t *testing.T) {
		host, setCmdUUID := setupHost(t, "failed-host", "1.2.3.10", "failedkey", "faileduuid")

		// Set failed status, matched on the in-flight SetRecoveryLock command
		err := ds.SetRecoveryLockFailed(ctx, host.UUID, setCmdUUID, "test error message")
		require.NoError(t, err)

		// Verify status and error message
		var status, errorMsg string
		err = ds.writer(ctx).GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.Equal(t, string(fleet.MDMDeliveryFailed), status)

		err = ds.writer(ctx).GetContext(ctx, &errorMsg, "SELECT error_message FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.Equal(t, "test error message", errorMsg)
	})

	t.Run("ClearRecoveryLockPendingStatus", func(t *testing.T) {
		host, _ := setupHost(t, "clear-pending-host", "1.2.3.11", "clearkey", "clearuuid")

		// Verify status is pending
		var status sql.NullString
		err := ds.writer(ctx).GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.Equal(t, string(fleet.MDMDeliveryPending), status.String)

		// Clear pending status
		err = ds.ClearRecoveryLockPendingStatus(ctx, []string{host.UUID})
		require.NoError(t, err)

		// Verify status is now NULL
		var checkResult struct {
			Status                   sql.NullString `db:"status"`
			PendingSetCommandUUID    sql.NullString `db:"pending_set_command_uuid"`
			PendingEncryptedPassword sql.NullString `db:"pending_encrypted_password"`
		}
		err = ds.writer(ctx).GetContext(ctx, &checkResult, "SELECT status, pending_set_command_uuid, pending_encrypted_password FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.False(t, checkResult.Status.Valid, "status should be NULL after clearing")
		assert.False(t, checkResult.PendingSetCommandUUID.Valid, "pending_set_command_uuid should be NULL after clearing")
		assert.False(t, checkResult.PendingEncryptedPassword.Valid, "pending_encrypted_password should be NULL after clearing")
	})

	t.Run("ClearRecoveryLockPendingStatus only clears pending", func(t *testing.T) {
		host, _ := setupHost(t, "no-clear-verified-host", "1.2.3.12", "ncvkey", "ncvuuid")

		// Set to verified
		markRecoveryLockVerified(t, ds, host.UUID)

		// Try to clear - should not affect verified status
		err := ds.ClearRecoveryLockPendingStatus(ctx, []string{host.UUID})
		require.NoError(t, err)

		// Verify status is still verified
		var status string
		err = ds.writer(ctx).GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", host.UUID)
		require.NoError(t, err)
		assert.Equal(t, string(fleet.MDMDeliveryVerified), status)
	})

	t.Run("RetryRecoveryLock", func(t *testing.T) {
		host, setCmdUUID := setupHost(t, "retry-host", "1.2.3.13", "rrkey", "rruuid")

		// A transient failure clears the status so the cron picks the host up again,
		// and bumps the retry counter that bounds how often that can happen.
		err := ds.RetryRecoveryLock(ctx, host.UUID, setCmdUUID)
		require.NoError(t, err)

		var result struct {
			Status sql.NullString `db:"status"`
			Retry  int            `db:"retry"`
		}
		err = ds.writer(ctx).GetContext(ctx, &result, `
			SELECT status, retry FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		assert.False(t, result.Status.Valid, "status should be NULL so the host is retried")
		assert.Equal(t, 1, result.Retry)

		// no-ops if no in-flight command exists.
		require.NoError(t, ds.RetryRecoveryLock(ctx, host.UUID, setCmdUUID))
		err = ds.writer(ctx).GetContext(ctx, &result, `
			SELECT status, retry FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Retry)
	})

	t.Run("RetryRecoveryLockVerify re-arms the verify without touching the set", func(t *testing.T) {
		host, setCmdUUID := setupHost(t, "retry-verify-host", "1.2.3.16", "rvkey", "rvuuid")

		verifyCmdUUID := uuid.NewString()
		require.NoError(t, ds.SetRecoveryLockVerifying(ctx, host.UUID, setCmdUUID, verifyCmdUUID))

		newVerifyCmdUUID := uuid.NewString()
		require.NoError(t, ds.RetryRecoveryLockVerify(ctx, host.UUID, verifyCmdUUID, newVerifyCmdUUID))

		var result struct {
			Status                   string         `db:"status"`
			Retry                    int            `db:"retry"`
			PendingVerifyCommandUUID sql.NullString `db:"pending_verify_command_uuid"`
			PendingEncryptedPassword []byte         `db:"pending_encrypted_password"`
		}
		const selectStmt = `
			SELECT status, retry, pending_verify_command_uuid, pending_encrypted_password
			FROM host_recovery_key_passwords WHERE host_uuid = ?`
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &result, selectStmt, host.UUID))

		// The row must stay in 'verifying': a NULL status would hand it to
		// GetHostsForRecoveryLockAction, which re-runs the set with the un-promoted
		// encrypted_password as CurrentPassword and fails on every attempt.
		assert.Equal(t, string(fleet.MDMDeliveryVerifying), result.Status)
		assert.Equal(t, 1, result.Retry)
		assert.Equal(t, newVerifyCmdUUID, result.PendingVerifyCommandUUID.String)
		assert.NotEmpty(t, result.PendingEncryptedPassword, "the pending password is still what we are verifying")

		// The host stays out of the cron's hands.
		hosts, err := ds.GetHostsForRecoveryLockAction(ctx)
		require.NoError(t, err)
		assert.False(t, hostNeedsRecoveryLockAction(hosts, host.UUID),
			"a verify awaiting retry must not be picked up for a SetRecoveryLock")

		// A result for the superseded verify command no longer matches.
		require.NoError(t, ds.RetryRecoveryLockVerify(ctx, host.UUID, verifyCmdUUID, uuid.NewString()))
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &result, selectStmt, host.UUID))
		assert.Equal(t, 1, result.Retry, "a stale verify UUID must not bump the counter")
		assert.Equal(t, newVerifyCmdUUID, result.PendingVerifyCommandUUID.String)
	})
}

// hostNeedsRecoveryLockAction reports whether GetHostsForRecoveryLockAction returned the
// host. The map value is whether the host already has a password, not membership.
func hostNeedsRecoveryLockAction(hosts map[string]bool, hostUUID string) bool {
	_, ok := hosts[hostUUID]
	return ok
}

// markRecoveryLockVerified drives a host through the verify step the way the MDM result
// handlers do: the SetRecoveryLock ack promotes the row to verifying, then the
// VerifyRecoveryLock ack marks it verified and promotes the pending password to active.
func markRecoveryLockVerified(t *testing.T, ds *Datastore, hostUUID string) {
	t.Helper()
	ctx := t.Context()
	// Both transitions are guarded on the command UUID the row is actually waiting on, so
	// read it rather than inventing one.
	pending, err := ds.GetPendingRecoveryLock(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, pending, "host has no recovery lock row to verify")
	require.NotNil(t, pending.PendingSetCommandUUID, "host has no in-flight set command to verify")

	verifyCmdUUID := uuid.NewString()
	require.NoError(t, ds.SetRecoveryLockVerifying(ctx, hostUUID, *pending.PendingSetCommandUUID, verifyCmdUUID))
	require.NoError(t, ds.SetRecoveryLockVerified(ctx, hostUUID, verifyCmdUUID))
}

// markRecoveryLockFailed fails the host's in-flight command, whichever step it is on.
// SetRecoveryLockFailed matches on the pending set or verify command UUID, so the caller
// does not have to track which one is outstanding.
func markRecoveryLockFailed(t *testing.T, ds *Datastore, hostUUID, errorMsg string) {
	t.Helper()
	ctx := t.Context()
	pending, err := ds.GetPendingRecoveryLock(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, pending, "host has no recovery lock row to fail")
	cmdUUID := pending.PendingVerifyCommandUUID
	if cmdUUID == nil {
		cmdUUID = pending.PendingSetCommandUUID
	}
	require.NotNil(t, cmdUUID, "host has no in-flight command to fail")
	require.NoError(t, ds.SetRecoveryLockFailed(ctx, hostUUID, *cmdUUID, errorMsg))
}

func testGetHostsForRecoveryLockAction(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Helper to create a team with recovery lock setting
	createTeamWithRecoveryLock := func(name string, enabled bool) *fleet.Team {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: name})
		require.NoError(t, err)

		team.Config.MDM.EnableRecoveryLockPassword = enabled
		team, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)
		return team
	}

	// Helper to set app config recovery lock setting
	setAppConfigRecoveryLock := func(enabled bool) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableRecoveryLockPassword = optjson.SetBool(enabled)
		err = ds.SaveAppConfig(ctx, ac)
		require.NoError(t, err)
	}

	// Helper to set host CPU type
	setHostCPUType := func(hostID uint, cpuType string) {
		_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET cpu_type = ? WHERE id = ?`, cpuType, hostID)
		require.NoError(t, err)
	}

	// Initially no eligible hosts
	hosts, err := ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.Empty(t, hosts)

	// Create eligible Apple Silicon host in team with recovery lock enabled
	teamARM := createTeamWithRecoveryLock("team-arm", true)
	hostARM := test.NewHost(t, ds, "arm-host", "1.2.5.1", "armkey", "armuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamARM.ID))
	setHostCPUType(hostARM.ID, "arm64")
	nanoEnrollAndSetHostMDMData(t, ds, hostARM, false)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.True(t, hostNeedsRecoveryLockAction(hosts, hostARM.UUID), "Apple Silicon (ARM) host should be eligible")

	// Create ineligible Intel host
	teamIntel := createTeamWithRecoveryLock("team-intel", true)
	hostIntel := test.NewHost(t, ds, "intel-host", "1.2.5.2", "intelkey", "inteluuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamIntel.ID))
	setHostCPUType(hostIntel.ID, "x86_64")
	nanoEnrollAndSetHostMDMData(t, ds, hostIntel, false)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostIntel.UUID), "Intel host should NOT be eligible")

	// Create host in team with recovery lock DISABLED
	teamDisabled := createTeamWithRecoveryLock("team-disabled", false)
	hostDisabled := test.NewHost(t, ds, "disabled-team-host", "1.2.5.4", "dtkey", "dtuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamDisabled.ID))
	setHostCPUType(hostDisabled.ID, "arm64e")
	nanoEnrollAndSetHostMDMData(t, ds, hostDisabled, false)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostDisabled.UUID), "host in disabled team should NOT be eligible")

	// Create host without MDM enrollment
	teamNotEnrolled := createTeamWithRecoveryLock("team-not-enrolled", true)
	hostNotEnrolled := test.NewHost(t, ds, "not-enrolled-host", "1.2.5.5", "nekey", "neuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamNotEnrolled.ID))
	setHostCPUType(hostNotEnrolled.ID, "arm64e")
	// No nano enrollment

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostNotEnrolled.UUID), "non-enrolled host should NOT be eligible")

	// Create Windows host (not darwin)
	teamNotDarwin := createTeamWithRecoveryLock("team-not-darwin", true)
	hostWindows := test.NewHost(t, ds, "windows-host", "1.2.5.6", "wkey", "wuuid", time.Now(),
		test.WithPlatform("windows"), test.WithTeamID(teamNotDarwin.ID))
	nanoEnrollAndSetHostMDMData(t, ds, hostWindows, false)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostWindows.UUID), "Windows host should NOT be eligible")

	// Create host with pending status (already has SetRecoveryLock in progress)
	// Note: SetHostsRecoveryLockPasswords now sets status to 'pending' atomically
	teamPending := createTeamWithRecoveryLock("team-pending", true)
	hostPending := test.NewHost(t, ds, "pending-host2", "1.2.5.7", "pkey2", "puuid2", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamPending.ID))
	setHostCPUType(hostPending.ID, "arm64e")
	nanoEnrollAndSetHostMDMData(t, ds, hostPending, false)
	pendingPW := apple_mdm.GenerateRecoveryLockPassword()
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: hostPending.UUID, Password: pendingPW}})
	require.NoError(t, err)
	// Status is already 'pending' from SetHostsRecoveryLockPasswords - no need to call SetRecoveryLockPending

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostPending.UUID), "pending host should NOT be eligible")

	// Create host with verified status (already has recovery lock set)
	teamVerified := createTeamWithRecoveryLock("team-verified", true)
	hostVerified := test.NewHost(t, ds, "verified-host2", "1.2.5.8", "vkey2", "vuuid2", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamVerified.ID))
	setHostCPUType(hostVerified.ID, "arm64e")
	nanoEnrollAndSetHostMDMData(t, ds, hostVerified, false)
	verifiedPW := apple_mdm.GenerateRecoveryLockPassword()
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: hostVerified.UUID, Password: verifiedPW}})
	require.NoError(t, err)
	markRecoveryLockVerified(t, ds, hostVerified.UUID)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostVerified.UUID), "verified host should NOT be eligible")

	// Create BYOD (personally-owned) enrolled host. Personal enrollments have the
	// DeviceLock/DeviceErase rights stripped, so SetRecoveryLock would fail on them.
	teamPersonal := createTeamWithRecoveryLock("team-personal", true)
	hostPersonal := test.NewHost(t, ds, "personal-host", "1.2.5.11", "perskey", "persuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamPersonal.ID))
	setHostCPUType(hostPersonal.ID, "arm64e")
	nanoEnroll(t, ds, hostPersonal, false)
	err = ds.SetOrUpdateMDMData(ctx, hostPersonal.ID, false, true, "https://fleetdm.com", false, fleet.WellKnownMDMFleet, "", true)
	require.NoError(t, err)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostPersonal.UUID), "personally-owned (BYOD) host should NOT be eligible")

	// Test no-team host with app config recovery lock enabled
	setAppConfigRecoveryLock(true)
	hostNoTeam := test.NewHost(t, ds, "no-team-host", "1.2.5.9", "ntkey", "ntuuid", time.Now(),
		test.WithPlatform("darwin"))
	setHostCPUType(hostNoTeam.ID, "arm64e")
	nanoEnrollAndSetHostMDMData(t, ds, hostNoTeam, false)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.True(t, hostNeedsRecoveryLockAction(hosts, hostNoTeam.UUID), "no-team host should be eligible when app config enabled")

	// Clean up - disable app config recovery lock
	setAppConfigRecoveryLock(false)

	// Now the no-team host should not be eligible
	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostNoTeam.UUID), "no-team host should NOT be eligible when app config disabled")

	// Create host with nano enrollment but MDM turned off (host_mdm.enrolled = 0)
	// This tests that hosts are properly excluded after MDMTurnOff is called
	teamUnenrolled := createTeamWithRecoveryLock("team-unenrolled", true)
	hostUnenrolled := test.NewHost(t, ds, "unenrolled-host", "1.2.5.10", "uekey", "ueuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamUnenrolled.ID))
	setHostCPUType(hostUnenrolled.ID, "arm64e")
	nanoEnroll(t, ds, hostUnenrolled, false)
	// Set host_mdm with enrolled = false (simulates MDM turn off)
	err = ds.SetOrUpdateMDMData(ctx, hostUnenrolled.ID, false, false, "", false, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)

	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostUnenrolled.UUID), "host with MDM turned off should NOT be eligible")

	// Test host in "pending remove" state is NOT picked up by GetHostsForRecoveryLockAction
	// Instead, RestoreRecoveryLockForReenabledHosts should handle this case
	// This tests the scenario where:
	// 1. Feature is disabled, host goes to operation_type='remove', status='pending'
	// 2. Feature is re-enabled
	// 3. RestoreRecoveryLockForReenabledHosts restores it to "verified install"
	// 4. GetHostsForRecoveryLockAction should NOT pick it up (it's already verified)
	teamReEnable := createTeamWithRecoveryLock("team-reenable", true)
	hostReEnable := test.NewHost(t, ds, "reenable-host", "1.2.5.11", "rekey", "reuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamReEnable.ID))
	setHostCPUType(hostReEnable.ID, "arm64e")
	nanoEnrollAndSetHostMDMData(t, ds, hostReEnable, false)

	// Set and verify the password
	reEnablePW := apple_mdm.GenerateRecoveryLockPassword()
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: hostReEnable.UUID, Password: reEnablePW}})
	require.NoError(t, err)
	markRecoveryLockVerified(t, ds, hostReEnable.UUID)

	// Disable recovery lock for team (triggers pending remove state)
	teamReEnable.Config.MDM.EnableRecoveryLockPassword = false
	_, err = ds.SaveTeam(ctx, teamReEnable)
	require.NoError(t, err)

	// Claim for clear - this sets operation_type to "remove" and status to "pending"
	_, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
	require.NoError(t, err)

	// Host should NOT be eligible while feature is disabled
	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostReEnable.UUID), "host in pending remove state should NOT be eligible while feature is disabled")

	// Re-enable recovery lock for team
	teamReEnable.Config.MDM.EnableRecoveryLockPassword = true
	_, err = ds.SaveTeam(ctx, teamReEnable)
	require.NoError(t, err)

	// Host should still NOT be eligible for GetHostsForRecoveryLockAction
	// (it needs to be restored first by RestoreRecoveryLockForReenabledHosts)
	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostReEnable.UUID), "host in pending remove state should NOT be picked up by GetHostsForRecoveryLockAction")

	// set auto_rotate_at to a past time to simulate due rotation
	_, err = ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_recovery_key_passwords
		SET auto_rotate_at = ?
		WHERE host_uuid = ?
		  AND deleted = 0
	`, time.Now().Add(-1*time.Hour), hostReEnable.UUID)
	require.NoError(t, err)

	// RestoreRecoveryLockForReenabledHosts should restore the host to "verified install"
	restored, err := ds.RestoreRecoveryLockForReenabledHosts(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), restored, "should restore one host")

	// Verify the host is now in "verified install" state
	var opType, status string
	err = ds.writer(ctx).GetContext(ctx, &opType, "SELECT operation_type FROM host_recovery_key_passwords WHERE host_uuid = ?", hostReEnable.UUID)
	require.NoError(t, err)
	assert.Equal(t, string(fleet.MDMOperationTypeInstall), opType)

	err = ds.writer(ctx).GetContext(ctx, &status, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", hostReEnable.UUID)
	require.NoError(t, err)
	assert.Equal(t, string(fleet.MDMDeliveryVerified), status)

	var autoRotateAt sql.NullTime
	err = ds.writer(ctx).GetContext(ctx, &autoRotateAt, "SELECT auto_rotate_at FROM host_recovery_key_passwords WHERE host_uuid = ?", hostReEnable.UUID)
	require.NoError(t, err)
	assert.False(t, autoRotateAt.Valid, "auto_rotate_at should be NULL")

	// Host should STILL not be eligible (it's verified, not pending)
	hosts, err = ds.GetHostsForRecoveryLockAction(ctx)
	require.NoError(t, err)
	assert.False(t, hostNeedsRecoveryLockAction(hosts, hostReEnable.UUID), "verified host should NOT be eligible")

	// Test that RestoreRecoveryLockForReenabledHosts does NOT restore failed records
	// This tests the scenario where:
	// 1. Feature is disabled, host goes to operation_type='remove'
	// 2. ClearRecoveryLock fails with terminal error (e.g., password mismatch)
	// 3. Host is now in (remove, failed) state with error_message
	// 4. Feature is re-enabled
	// 5. RestoreRecoveryLockForReenabledHosts should NOT restore this host
	//    because it's a terminal error requiring admin intervention
	teamFailed := createTeamWithRecoveryLock("team-failed", true)
	hostFailed := test.NewHost(t, ds, "failed-host", "1.2.5.12", "failkey", "failuuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(teamFailed.ID))
	setHostCPUType(hostFailed.ID, "arm64e")
	nanoEnrollAndSetHostMDMData(t, ds, hostFailed, false)

	// Set and verify the password
	failedPW := apple_mdm.GenerateRecoveryLockPassword()
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: hostFailed.UUID, Password: failedPW}})
	require.NoError(t, err)
	markRecoveryLockVerified(t, ds, hostFailed.UUID)

	// Disable recovery lock for team
	teamFailed.Config.MDM.EnableRecoveryLockPassword = false
	_, err = ds.SaveTeam(ctx, teamFailed)
	require.NoError(t, err)

	// Claim for clear - sets operation_type to "remove" and status to "pending"
	_, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
	require.NoError(t, err)

	// Simulate ClearRecoveryLock failing with terminal error (password mismatch)
	markRecoveryLockFailed(t, ds, hostFailed.UUID, "Password mismatch: The provided recovery password failed to validate.")

	// Verify host is in (remove, failed) state
	var failedOpType, failedStatus, failedErrorMsg string
	err = ds.writer(ctx).GetContext(ctx, &failedOpType, "SELECT operation_type FROM host_recovery_key_passwords WHERE host_uuid = ?", hostFailed.UUID)
	require.NoError(t, err)
	assert.Equal(t, string(fleet.MDMOperationTypeRemove), failedOpType)
	err = ds.writer(ctx).GetContext(ctx, &failedStatus, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", hostFailed.UUID)
	require.NoError(t, err)
	assert.Equal(t, string(fleet.MDMDeliveryFailed), failedStatus)
	err = ds.writer(ctx).GetContext(ctx, &failedErrorMsg, "SELECT error_message FROM host_recovery_key_passwords WHERE host_uuid = ?", hostFailed.UUID)
	require.NoError(t, err)
	assert.Contains(t, failedErrorMsg, "Password mismatch")

	// Re-enable recovery lock for team
	teamFailed.Config.MDM.EnableRecoveryLockPassword = true
	_, err = ds.SaveTeam(ctx, teamFailed)
	require.NoError(t, err)

	// RestoreRecoveryLockForReenabledHosts should NOT restore the failed host
	restored, err = ds.RestoreRecoveryLockForReenabledHosts(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), restored, "should NOT restore failed hosts")

	// Verify host is STILL in (remove, failed) state with error_message preserved
	err = ds.writer(ctx).GetContext(ctx, &failedOpType, "SELECT operation_type FROM host_recovery_key_passwords WHERE host_uuid = ?", hostFailed.UUID)
	require.NoError(t, err)
	assert.Equal(t, string(fleet.MDMOperationTypeRemove), failedOpType, "operation_type should still be 'remove'")
	err = ds.writer(ctx).GetContext(ctx, &failedStatus, "SELECT status FROM host_recovery_key_passwords WHERE host_uuid = ?", hostFailed.UUID)
	require.NoError(t, err)
	assert.Equal(t, string(fleet.MDMDeliveryFailed), failedStatus, "status should still be 'failed'")
	err = ds.writer(ctx).GetContext(ctx, &failedErrorMsg, "SELECT error_message FROM host_recovery_key_passwords WHERE host_uuid = ?", hostFailed.UUID)
	require.NoError(t, err)
	assert.Contains(t, failedErrorMsg, "Password mismatch", "error_message should be preserved")
}

func testClaimHostsForRecoveryLockClear(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Helper to create a team with recovery lock setting
	createTeamWithRecoveryLock := func(t *testing.T, name string, enabled bool) *fleet.Team {
		t.Helper()
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: name})
		require.NoError(t, err)

		team.Config.MDM.EnableRecoveryLockPassword = enabled
		team, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)
		return team
	}

	// Helper to set app config recovery lock setting
	setAppConfigRecoveryLock := func(t *testing.T, enabled bool) {
		t.Helper()
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableRecoveryLockPassword = optjson.SetBool(enabled)
		err = ds.SaveAppConfig(ctx, ac)
		require.NoError(t, err)
	}

	// Helper to set host CPU type
	setHostCPUType := func(t *testing.T, hostID uint, cpuType string) {
		t.Helper()
		_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET cpu_type = ? WHERE id = ?`, cpuType, hostID)
		require.NoError(t, err)
	}

	// Helper to get password record (excludes soft-deleted records)
	getPasswordRecord := func(t *testing.T, hostUUID string) (opType, status string, found bool) {
		t.Helper()
		var rec struct {
			OperationType string  `db:"operation_type"`
			Status        *string `db:"status"`
		}
		err := sqlx.GetContext(ctx, ds.reader(ctx), &rec,
			`SELECT operation_type, status FROM host_recovery_key_passwords WHERE host_uuid = ? AND deleted = 0`, hostUUID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", false
			}
			t.Fatalf("getPasswordRecord query failed: %v", err)
		}
		if rec.Status != nil {
			status = *rec.Status
		}
		return rec.OperationType, status, true
	}

	t.Run("no hosts to clear returns empty", func(t *testing.T) {
		hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.Empty(t, hosts)
	})

	t.Run("claims verified host when config disabled", func(t *testing.T) {
		// Create team with recovery lock enabled initially
		team := createTeamWithRecoveryLock(t, "removing-status-team", true)
		host := test.NewHost(t, ds, "removing-rlp-host", "1.2.6.4", "removingrlpkey", "removingrlpuuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		// Set password and verify
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Disable recovery lock for team to trigger clear
		team.Config.MDM.EnableRecoveryLockPassword = false
		_, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)

		// Claim for clear - this sets operation_type to "remove" and status to "pending"
		_, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)

		// Verify state is now operation_type=remove, status=pending
		opType, status, found := getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "remove", opType)
		assert.Equal(t, "pending", status)
	})

	t.Run("does not claim personally-owned (BYOD) host", func(t *testing.T) {
		// Personal enrollments have DeviceLock/DeviceErase rights stripped, so
		// recovery lock commands (including clear) are rejected by the device.
		team := createTeamWithRecoveryLock(t, "personal-clear-team", true)
		host := test.NewHost(t, ds, "personal-clear-host", "1.2.6.8", "perscleerkey", "perscleeruuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnroll(t, ds, host, false)
		err := ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://fleetdm.com", false, fleet.WellKnownMDMFleet, "", true)
		require.NoError(t, err)

		// Give it a verified password record that would otherwise be claimed for clear.
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Disable recovery lock for team to trigger clear.
		team.Config.MDM.EnableRecoveryLockPassword = false
		_, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)

		uuids, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.NotContains(t, uuids, host.UUID, "personally-owned (BYOD) host should NOT be claimed for clear")
	})

	t.Run("clears stale auto_rotate_at when flipping to remove", func(t *testing.T) {
		team := createTeamWithRecoveryLock(t, "stale-rotation-team", true)
		host := test.NewHost(t, ds, "stale-rotation-host", "1.2.6.7", "stalerotkey", "stalerotuuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Simulate the user viewing the password under the install-state row,
		// which schedules a rotation.
		priorRotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)
		require.False(t, priorRotateAt.IsZero())

		// Disabling the team's setting and running the clear claim must wipe
		// the stale view-deadline — auto_rotate_at is meaningful only for
		// install-state rows, and leaving it would cause a subsequent view to
		// return a rotation time the cron will never honor.
		team.Config.MDM.EnableRecoveryLockPassword = false
		_, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)

		uuids, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		require.Contains(t, uuids, host.UUID)

		var autoRotateAt *time.Time
		err = sqlx.GetContext(ctx, ds.reader(ctx), &autoRotateAt,
			`SELECT auto_rotate_at FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		assert.Nil(t, autoRotateAt, "auto_rotate_at should be cleared when flipping operation_type to remove")
	})

	t.Run("returns pending when operation_type is install and status is NULL", func(t *testing.T) {
		host := test.NewHost(t, ds, "install-null-host", "1.2.6.5", "installnullkey", "installnulluuid", time.Now())

		// Insert a record with operation_type=install and status=NULL directly
		_, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, operation_type, status)
			 VALUES (?, 'test', 'install', NULL)`, host.UUID)
		require.NoError(t, err)

		// Verify state is operation_type=install, status=NULL
		opType, status, found := getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "install", opType)
		assert.Empty(t, status, "status should be empty (NULL) when operation_type is install and status is NULL")
	})

	t.Run("returns verified status", func(t *testing.T) {
		// Create team with recovery lock enabled
		team := createTeamWithRecoveryLock(t, "verified-status-team", true)
		host := test.NewHost(t, ds, "verified-rlp-host", "1.2.6.3", "verifiedrlpkey", "verifiedrlpuuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		// Set password and mark as verified
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Verify initial state
		opType, status, found := getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "install", opType)
		assert.Equal(t, "verified", status)

		// Should not be claimed while config is enabled
		hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.NotContains(t, hosts, host.UUID)

		// Disable recovery lock for team
		team.Config.MDM.EnableRecoveryLockPassword = false
		_, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)

		// Now host should be claimed
		hosts, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.Contains(t, hosts, host.UUID)

		// Verify state changed to remove/pending
		opType, status, found = getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "remove", opType)
		assert.Equal(t, "pending", status)

		// Should not be claimed again (already pending)
		hosts, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.NotContains(t, hosts, host.UUID)
	})

	t.Run("does not claim pending or failed hosts", func(t *testing.T) {
		team := createTeamWithRecoveryLock(t, "clear-pending-team", false)

		// Host with pending status
		hostPending := test.NewHost(t, ds, "pending-clear", "1.2.6.2", "pendkey", "penduuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, hostPending.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, hostPending, false)
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: hostPending.UUID, Password: pw}})
		require.NoError(t, err)
		// Status is pending from SetHostsRecoveryLockPasswords

		// Host with failed status
		hostFailed := test.NewHost(t, ds, "failed-clear", "1.2.6.3", "failkey", "failuuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, hostFailed.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, hostFailed, false)
		pw2 := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: hostFailed.UUID, Password: pw2}})
		require.NoError(t, err)
		markRecoveryLockFailed(t, ds, hostFailed.UUID, "test error")

		hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.NotContains(t, hosts, hostPending.UUID, "pending host should not be claimed")
		assert.NotContains(t, hosts, hostFailed.UUID, "failed host should not be claimed")
	})

	t.Run("claims no-team host when appconfig disabled", func(t *testing.T) {
		// Enable recovery lock in appconfig
		setAppConfigRecoveryLock(t, true)

		host := test.NewHost(t, ds, "noteam-clear", "1.2.6.4", "ntkey", "ntuuid", time.Now(),
			test.WithPlatform("darwin"))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Should not be claimed while appconfig enabled
		hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.NotContains(t, hosts, host.UUID)

		// Disable recovery lock in appconfig
		setAppConfigRecoveryLock(t, false)

		// Now should be claimed
		hosts, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.Contains(t, hosts, host.UUID)
	})

	t.Run("delete removes the password record once the clear is verified", func(t *testing.T) {
		team := createTeamWithRecoveryLock(t, "delete-test-team", true)
		host := test.NewHost(t, ds, "delete-host", "1.2.6.5", "delkey", "deluuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Verify record exists
		_, _, found := getPasswordRecord(t, host.UUID)
		require.True(t, found)

		// Disabling the feature claims the host for clear (remove/pending).
		team.Config.MDM.EnableRecoveryLockPassword = false
		_, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)
		clearCmdUUID := uuid.NewString()
		claimed, err := ds.ClaimHostsForRecoveryLockClear(ctx, clearCmdUUID)
		require.NoError(t, err)
		require.Contains(t, claimed, host.UUID)

		// The clear command is acknowledged, then verified with an empty password.
		verifyCmdUUID := uuid.NewString()
		err = ds.SetRecoveryLockVerifying(ctx, host.UUID, clearCmdUUID, verifyCmdUUID)
		require.NoError(t, err)
		err = ds.DeleteHostRecoveryLockPassword(ctx, host.UUID, verifyCmdUUID)
		require.NoError(t, err)

		// The row is removed outright once the device confirms the lock is gone.
		_, _, found = getPasswordRecord(t, host.UUID)
		assert.False(t, found)

		var count int
		err = sqlx.GetContext(ctx, ds.reader(ctx), &count,
			`SELECT COUNT(*) FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		assert.Zero(t, count, "record should be hard deleted")
	})

	t.Run("get pending recovery lock", func(t *testing.T) {
		team := createTeamWithRecoveryLock(t, "optype-test-team", true)
		host := test.NewHost(t, ds, "optype-host", "1.2.6.6", "optkey", "optuuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		// No record - returns nil rather than an error
		pending, err := ds.GetPendingRecoveryLock(ctx, host.UUID)
		require.NoError(t, err)
		assert.Nil(t, pending)

		// Create record with install type
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		pending, err = ds.GetPendingRecoveryLock(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pending)
		assert.Equal(t, fleet.MDMOperationTypeInstall, pending.OperationType)
		assert.True(t, pending.HasCurrentPassword)

		// Claim for clear - changes to remove type
		team.Config.MDM.EnableRecoveryLockPassword = false
		_, err = ds.SaveTeam(ctx, team)
		require.NoError(t, err)
		_, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)

		pending, err = ds.GetPendingRecoveryLock(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pending)
		assert.Equal(t, fleet.MDMOperationTypeRemove, pending.OperationType)
	})

	t.Run("retries failed clear attempts", func(t *testing.T) {
		team := createTeamWithRecoveryLock(t, "retry-test-team", false)
		host := test.NewHost(t, ds, "retry-host", "1.2.6.7", "retrykey", "retryuuid", time.Now(),
			test.WithPlatform("darwin"), test.WithTeamID(team.ID))
		setHostCPUType(t, host.ID, "arm64")
		nanoEnrollAndSetHostMDMData(t, ds, host, false)

		// Set password and mark as verified
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		// Claim for clear
		hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.Contains(t, hosts, host.UUID)

		// Verify state is remove/pending
		opType, status, found := getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "remove", opType)
		assert.Equal(t, "pending", status)

		// Simulate failed enqueue - clear pending status back to NULL
		err = ds.ClearRecoveryLockPendingStatus(ctx, []string{host.UUID})
		require.NoError(t, err)

		// Verify state is remove/NULL (retry state)
		opType, status, found = getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "remove", opType)
		assert.Empty(t, status) // NULL becomes empty string

		// Should be claimed again on retry
		hosts, err = ds.ClaimHostsForRecoveryLockClear(ctx, uuid.NewString())
		require.NoError(t, err)
		assert.Contains(t, hosts, host.UUID, "host with remove/NULL should be retried")

		// Verify state is back to remove/pending
		opType, status, found = getPasswordRecord(t, host.UUID)
		require.True(t, found)
		assert.Equal(t, "remove", opType)
		assert.Equal(t, "pending", status)
	})
}

func testGetHostRecoveryLockPasswordStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("returns nil for host without recovery lock password", func(t *testing.T) {
		host := test.NewHost(t, ds, "no-rlp-host", "1.2.6.1", "norlpkey", "norlpuuid", time.Now())

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		assert.Nil(t, status)
	})

	t.Run("returns enforcing status for pending install", func(t *testing.T) {
		host := test.NewHost(t, ds, "pending-rlp-host", "1.2.6.2", "pendingrlpkey", "pendingrlpuuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *status.Status)
		assert.Empty(t, status.Detail)
		// The password is still staged as pending, so it is not offered to the admin until
		// the device has accepted and verified it.
		assert.False(t, status.PasswordAvailable)
	})

	t.Run("returns verified status", func(t *testing.T) {
		host := test.NewHost(t, ds, "verified-rlp-host", "1.2.6.3", "verifiedrlpkey", "verifiedrlpuuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *status.Status)
		assert.Empty(t, status.Detail)
		assert.True(t, status.PasswordAvailable)
	})

	t.Run("returns failed status with error message", func(t *testing.T) {
		host := test.NewHost(t, ds, "failed-rlp-host", "1.2.6.4", "failedrlpkey", "failedrlpuuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		errMsg := "SetRecoveryLock command failed: device rejected"
		markRecoveryLockFailed(t, ds, host.UUID, errMsg)

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusFailed, *status.Status)
		assert.Equal(t, errMsg, status.Detail)
	})

	t.Run("returns verifying status", func(t *testing.T) {
		host := test.NewHost(t, ds, "verifying-rlp-host", "1.2.6.5", "verifyingrlpkey", "verifyingrlpuuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		setCmdUUID := "set-cmd-uuid"
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw, PendingSetCommandUUID: setCmdUUID}})
		require.NoError(t, err)
		require.NoError(t, ds.SetRecoveryLockVerifying(ctx, host.UUID, setCmdUUID, "pending-verify-cmd-uuid"))

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *status.Status)
		assert.Empty(t, status.Detail)
	})

	t.Run("returns enforcing status when status column is NULL (retry state)", func(t *testing.T) {
		host := test.NewHost(t, ds, "null-status-host", "1.2.6.6", "nullstatuskey", "nullstatusuuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		// Clear status to NULL (simulates retry state after failed enqueue)
		err = ds.ClearRecoveryLockPendingStatus(ctx, []string{host.UUID})
		require.NoError(t, err)

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		// NULL status is coalesced to pending, which becomes enforcing
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *status.Status)
		assert.Empty(t, status.Detail)
	})

	t.Run("returns removing_enforcement status for pending removal after PopulateStatus", func(t *testing.T) {
		host := test.NewHost(t, ds, "remove-pending-host", "1.2.6.7", "removependingkey", "removependinguuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)
		// Set operation_type to 'remove' and status to 'pending' (simulates pending removal)
		_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_recovery_key_passwords SET operation_type = ?, status = ? WHERE host_uuid = ?`,
			fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending, host.UUID)
		require.NoError(t, err)

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)

		// Before PopulateStatus, Status is nil (raw status is internal)
		assert.Nil(t, status.Status)

		// After PopulateStatus, Status is removing_enforcement
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusRemovingEnforcement, *status.Status)
	})

	t.Run("returns failed status when operation_type is remove and status is failed", func(t *testing.T) {
		host := test.NewHost(t, ds, "remove-failed-host", "1.2.6.8", "removefailedkey", "removefaileduuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)
		// Set operation_type to 'remove' and status to 'failed'
		errMsg := "ClearRecoveryLock command failed"
		_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_recovery_key_passwords SET operation_type = ?, status = ?, error_message = ? WHERE host_uuid = ?`,
			fleet.MDMOperationTypeRemove, fleet.MDMDeliveryFailed, errMsg, host.UUID)
		require.NoError(t, err)

		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, errMsg, status.Detail)

		// After PopulateStatus, Status is failed
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusFailed, *status.Status)
	})
}

func testRecoveryLockRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Helper to set up a host with a verified recovery lock password
	setupHostWithVerifiedPassword := func(t *testing.T, name, uuid string) *fleet.Host {
		t.Helper()
		host := test.NewHost(t, ds, name, "1.2.3."+uuid[:3], name+"key", uuid, time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)
		return host
	}

	// Helper to get pending rotation state
	getPendingRotationState := func(t *testing.T, hostUUID string) (hasPending bool, pendingErr *string) {
		t.Helper()
		var result struct {
			HasPending bool    `db:"has_pending"`
			PendingErr *string `db:"pending_err"`
		}
		err := ds.writer(ctx).GetContext(ctx, &result, `
			SELECT
				pending_encrypted_password IS NOT NULL AS has_pending,
				error_message AS pending_err
			FROM host_recovery_key_passwords
			WHERE host_uuid = ? AND deleted = 0`, hostUUID)
		if err == sql.ErrNoRows {
			return false, nil
		}
		require.NoError(t, err)
		return result.HasPending, result.PendingErr
	}

	t.Run("InitiateRecoveryLockRotation success", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "rotate-host", "rotateuuid1")

		// Initiate rotation
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Verify pending password is set
		hasPending, _ := getPendingRotationState(t, host.UUID)
		assert.True(t, hasPending, "pending password should be set")

		// The row now carries the pending SetRecoveryLock command
		pending, err := ds.GetPendingRecoveryLock(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pending)
		assert.NotNil(t, pending.PendingSetCommandUUID)
	})

	t.Run("InitiateRecoveryLockRotation rejects if already pending", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "double-rotate-host", "doublerotuuid")

		// Initiate first rotation
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Try to initiate second rotation - should fail
		err = ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), "another-password")
		require.Error(t, err)
		assert.ErrorIs(t, err, fleet.ErrRecoveryLockRotationPending)
	})

	t.Run("InitiateRecoveryLockRotation rejects pending status", func(t *testing.T) {
		host := test.NewHost(t, ds, "pending-rotate-host", "1.2.3.100", "pendingrotkey", "pendingrotuuid", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		// Status is pending after SetHostsRecoveryLockPasswords

		// Try to initiate rotation on pending status - should fail. The initial set has
		// already staged a pending password, so this reports as a pending rotation.
		err = ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), "new-password")
		require.Error(t, err)
		assert.ErrorIs(t, err, fleet.ErrRecoveryLockRotationPending)
	})

	t.Run("InitiateRecoveryLockRotation allows failed status", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "failed-rotate-host", "failedrotuuid")

		// Reach 'failed' the only way a row can: an in-flight rotation that failed.
		require.NoError(t, ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), apple_mdm.GenerateRecoveryLockPassword()))
		markRecoveryLockFailed(t, ds, host.UUID, "previous failure")

		// Should be able to initiate rotation on failed status
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		hasPending, _ := getPendingRotationState(t, host.UUID)
		assert.True(t, hasPending)
	})

	t.Run("CompleteRecoveryLockRotation success", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "complete-rotate-host", "completerotuuid")

		// Get original password
		origPw, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
		require.NoError(t, err)

		// Initiate rotation with new password
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// The rotation completes through the normal verify path
		markRecoveryLockVerified(t, ds, host.UUID)

		// Verify new password is now the active password
		currentPw, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, currentPw.Password)
		assert.NotEqual(t, origPw.Password, currentPw.Password)
		assert.Equal(t, newPassword, *currentPw.Password)

		// Verify pending is cleared
		hasPending, _ := getPendingRotationState(t, host.UUID)
		assert.False(t, hasPending)

		// Verify status is verified
		status, err := ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status.Status)
		assert.Equal(t, string(fleet.MDMDeliveryVerified), *status.Status)
	})

	t.Run("a failed rotation clears the pending password", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "fail-rotate-host", "failrotuuid")

		// Initiate rotation
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Fail the rotation
		markRecoveryLockFailed(t, ds, host.UUID, "rotation failed due to device error")

		// The pending password is dropped: a retry generates a fresh one rather than
		// re-sending a password the device may or may not have accepted.
		hasPending, pendingErr := getPendingRotationState(t, host.UUID)
		assert.False(t, hasPending, "pending password should be cleared")
		require.NotNil(t, pendingErr)
		assert.Equal(t, "rotation failed due to device error", *pendingErr)

		status, err := ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
		require.NoError(t, err)
		assert.False(t, status.HasPendingRotation)
	})

	t.Run("ClearRecoveryLockRotation removes pending", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "clear-rotate-host", "clearrotuuid")

		// Initiate rotation
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Clear rotation
		err = ds.ClearRecoveryLockRotation(ctx, host.UUID)
		require.NoError(t, err)

		// Verify pending is cleared
		hasPending, _ := getPendingRotationState(t, host.UUID)
		assert.False(t, hasPending)

		// The pending SetRecoveryLock command is cleared too
		pending, err := ds.GetPendingRecoveryLock(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pending)
		assert.Nil(t, pending.PendingSetCommandUUID)

		// Verify status restored to verified (since it was verified before rotation)
		status, err := ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status.Status)
		assert.Equal(t, string(fleet.MDMDeliveryVerified), *status.Status)
	})

	t.Run("ClearRecoveryLockRotation keeps failed after clearing on previous failed attempt", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "clear-failed-rotate-host", "clearfailedrotuuid")

		// Reach 'failed' the only way a row can: an in-flight rotation that failed.
		require.NoError(t, ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), apple_mdm.GenerateRecoveryLockPassword()))
		markRecoveryLockFailed(t, ds, host.UUID, "previous failure")

		// Initiate rotation from failed state
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Clear rotation
		err = ds.ClearRecoveryLockRotation(ctx, host.UUID)
		require.NoError(t, err)

		// Verify pending is cleared
		hasPending, _ := getPendingRotationState(t, host.UUID)
		assert.False(t, hasPending)

		// Verify that the status remains 'failed' after clearing the rotation, reflecting the previous failed attempt.
		status, err := ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status.Status)
		assert.Equal(t, string(fleet.MDMDeliveryFailed), *status.Status)
	})

	t.Run("GetRecoveryLockRotationStatus returns all fields", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "status-rotate-host", "statusrotuuid")

		// Get initial status
		status, err := ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
		require.NoError(t, err)
		assert.Equal(t, host.UUID, status.HostUUID)
		assert.True(t, status.HasPassword)
		require.NotNil(t, status.Status)
		assert.Equal(t, string(fleet.MDMDeliveryVerified), *status.Status)
		assert.Equal(t, string(fleet.MDMOperationTypeInstall), status.OperationType)
		assert.False(t, status.HasPendingRotation)

		// Initiate rotation
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Check status now shows pending rotation
		status, err = ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
		require.NoError(t, err)
		assert.True(t, status.HasPendingRotation)
	})

	t.Run("GetRecoveryLockRotationStatus not found", func(t *testing.T) {
		_, err := ds.GetRecoveryLockRotationStatus(ctx, "non-existent-uuid")
		require.Error(t, err)
		assert.True(t, fleet.IsNotFound(err))
	})

	t.Run("GetPendingRecoveryLock returns nil for no record", func(t *testing.T) {
		pending, err := ds.GetPendingRecoveryLock(ctx, "non-existent-uuid")
		require.NoError(t, err)
		assert.Nil(t, pending)
	})
}

func testRecoveryLockAutoRotation(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Helper to set up a host with a verified recovery lock password
	setupHostWithVerifiedPassword := func(t *testing.T, name, uuid string) *fleet.Host {
		t.Helper()
		host := test.NewHost(t, ds, name, "2.3.4."+uuid[:3], name+"key", uuid, time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		markRecoveryLockVerified(t, ds, host.UUID)
		return host
	}

	// Helper to get auto_rotate_at directly from DB
	getAutoRotateAt := func(t *testing.T, hostUUID string) *time.Time {
		t.Helper()
		var autoRotateAt *time.Time
		err := ds.writer(ctx).GetContext(ctx, &autoRotateAt, `
			SELECT auto_rotate_at FROM host_recovery_key_passwords
			WHERE host_uuid = ? AND deleted = 0`, hostUUID)
		if err == sql.ErrNoRows {
			return nil
		}
		require.NoError(t, err)
		return autoRotateAt
	}

	t.Run("MarkRecoveryLockPasswordViewed sets auto_rotate_at", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "view-host1", "viewuuid0001")

		// Initially no auto_rotate_at
		autoRotateAt := getAutoRotateAt(t, host.UUID)
		assert.Nil(t, autoRotateAt)

		// Mark as viewed
		rotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)
		assert.False(t, rotateAt.IsZero())

		// Verify auto_rotate_at is approximately 1 hour from now
		expectedRotateAt := time.Now().Add(1 * time.Hour)
		assert.WithinDuration(t, expectedRotateAt, rotateAt, 1*time.Minute)

		// Verify via direct DB query
		autoRotateAt = getAutoRotateAt(t, host.UUID)
		require.NotNil(t, autoRotateAt)
		assert.WithinDuration(t, expectedRotateAt, *autoRotateAt, 1*time.Minute)
	})

	t.Run("MarkRecoveryLockPasswordViewed updates existing auto_rotate_at", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "view-host2", "viewuuid0002")

		// First view
		firstRotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamp

		// Second view should update auto_rotate_at
		secondRotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		// Second rotation time should be after first
		assert.True(t, secondRotateAt.After(firstRotateAt), "second view should update auto_rotate_at")

		// Verify the value was persisted in the database
		pw, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pw.AutoRotateAt, "auto_rotate_at should be persisted")
		assert.True(t, pw.AutoRotateAt.After(firstRotateAt), "persisted auto_rotate_at should be after first rotation time")
	})

	t.Run("MarkRecoveryLockPasswordViewed returns zero time for non-existent host", func(t *testing.T) {
		// Callers are expected to verify existence via GetHostRecoveryLockPassword
		// before scheduling rotation, so a missing row here is treated the same
		// as a non-install-state row: skip scheduling without erroring.
		rotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, "non-existent-uuid")
		require.NoError(t, err)
		assert.True(t, rotateAt.IsZero())
	})

	t.Run("MarkRecoveryLockPasswordViewed returns zero time for remove operation", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "view-host-remove", "viewuuidremove1")

		// Realistic sequence: password is viewed under install state (sets
		// auto_rotate_at), then the row is flipped to remove by the cleanup
		// path. A second view must not fail and must not (re-)schedule a
		// rotation that the auto-rotation cron won't honor.
		priorRotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)
		require.False(t, priorRotateAt.IsZero(), "view under install state should schedule rotation")

		_, err = ds.writer(ctx).ExecContext(ctx, `
			UPDATE host_recovery_key_passwords
			SET operation_type = 'remove'
			WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)

		rotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)
		assert.True(t, rotateAt.IsZero(), "expected zero rotateAt for remove-state row")

		// MarkRecoveryLockPasswordViewed itself must not touch a remove-state
		// row's auto_rotate_at — clearing that stale deadline is the
		// responsibility of ClaimHostsForRecoveryLockClear (covered in
		// testClaimHostsForRecoveryLockClear) so this assertion pins the
		// no-op contract.
		var autoRotateAt *time.Time
		err = sqlx.GetContext(ctx, ds.reader(ctx), &autoRotateAt,
			`SELECT auto_rotate_at FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, autoRotateAt)
		assert.WithinDuration(t, priorRotateAt, *autoRotateAt, time.Second,
			"MarkRecoveryLockPasswordViewed must not modify auto_rotate_at on a remove-state row")
	})

	// Helper to check if a host UUID is in the rotation info list
	containsHostUUID := func(hosts []fleet.HostAutoRotationInfo, uuid string) bool {
		for _, h := range hosts {
			if h.HostUUID == uuid {
				return true
			}
		}
		return false
	}

	t.Run("GetHostsForAutoRotation returns due hosts", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "auto-rotate-host1", "autorotateuuid1")

		// Set auto_rotate_at to 2 hours ago (past due)
		_, err := ds.writer(ctx).ExecContext(ctx, `
			UPDATE host_recovery_key_passwords
			SET auto_rotate_at = DATE_SUB(NOW(6), INTERVAL 2 HOUR)
			WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)

		// Should be returned
		hosts, err := ds.GetHostsForAutoRotation(ctx)
		require.NoError(t, err)
		assert.True(t, containsHostUUID(hosts, host.UUID), "host should be in auto-rotation list")
	})

	t.Run("GetHostsForAutoRotation excludes future auto_rotate_at", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "auto-rotate-host2", "autorotateuuid2")

		// Set auto_rotate_at to 1 hour in the future
		_, err := ds.writer(ctx).ExecContext(ctx, `
			UPDATE host_recovery_key_passwords
			SET auto_rotate_at = DATE_ADD(NOW(6), INTERVAL 1 HOUR)
			WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)

		// Should NOT be returned
		hosts, err := ds.GetHostsForAutoRotation(ctx)
		require.NoError(t, err)
		assert.False(t, containsHostUUID(hosts, host.UUID), "host should not be in auto-rotation list")
	})

	t.Run("GetHostsForAutoRotation excludes hosts with pending rotation", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "auto-rotate-host3", "autorotateuuid3")

		// Set auto_rotate_at to past due
		_, err := ds.writer(ctx).ExecContext(ctx, `
			UPDATE host_recovery_key_passwords
			SET auto_rotate_at = DATE_SUB(NOW(6), INTERVAL 2 HOUR)
			WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)

		// Initiate rotation (sets pending_encrypted_password)
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		// Should NOT be returned because pending rotation exists
		hosts, err := ds.GetHostsForAutoRotation(ctx)
		require.NoError(t, err)
		assert.False(t, containsHostUUID(hosts, host.UUID), "host should not be in auto-rotation list")
	})

	t.Run("GetHostsForAutoRotation excludes non-verified hosts", func(t *testing.T) {
		host := test.NewHost(t, ds, "auto-rotate-host4", "2.3.4.104", "autorotate4key", "autorotateuuid4", time.Now())
		pw := apple_mdm.GenerateRecoveryLockPassword()
		err := ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}})
		require.NoError(t, err)
		// Status is "pending" after SetHostsRecoveryLockPasswords, NOT verified

		// Set auto_rotate_at to past due
		_, err = ds.writer(ctx).ExecContext(ctx, `
			UPDATE host_recovery_key_passwords
			SET auto_rotate_at = DATE_SUB(NOW(6), INTERVAL 2 HOUR)
			WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)

		// Should NOT be returned because status is not verified
		hosts, err := ds.GetHostsForAutoRotation(ctx)
		require.NoError(t, err)
		assert.False(t, containsHostUUID(hosts, host.UUID), "host should not be in auto-rotation list")
	})

	t.Run("CompleteRecoveryLockRotation clears auto_rotate_at", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "complete-auto-rotate", "completeautorot")

		// Mark as viewed to set auto_rotate_at
		_, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		// Verify auto_rotate_at is set
		autoRotateAt := getAutoRotateAt(t, host.UUID)
		require.NotNil(t, autoRotateAt)

		// Initiate and complete rotation
		newPassword := apple_mdm.GenerateRecoveryLockPassword()
		err = ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), newPassword)
		require.NoError(t, err)

		markRecoveryLockVerified(t, ds, host.UUID)

		// auto_rotate_at should be cleared
		autoRotateAt = getAutoRotateAt(t, host.UUID)
		assert.Nil(t, autoRotateAt)
	})

	t.Run("GetHostRecoveryLockPassword includes auto_rotate_at", func(t *testing.T) {
		host := setupHostWithVerifiedPassword(t, "get-pw-auto-rotate", "getpwautorot")

		// Initially no auto_rotate_at
		pw, err := ds.GetHostRecoveryLockPassword(ctx, host.UUID)
		require.NoError(t, err)
		assert.Nil(t, pw.AutoRotateAt)

		// Mark as viewed
		rotateAt, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		// Now auto_rotate_at should be returned
		pw, err = ds.GetHostRecoveryLockPassword(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pw.AutoRotateAt)
		assert.WithinDuration(t, rotateAt, *pw.AutoRotateAt, 1*time.Second)
	})
}

// recoveryLockRawRow is the full row shape used by testRecoveryLockResetOnMDMReEnrollment's
// raw reader, which bypasses the deleted=0 filter applied by production readers.
type recoveryLockRawRow struct {
	Status        *string    `db:"status"`
	OperationType string     `db:"operation_type"`
	HasPassword   bool       `db:"has_password"`
	HasPendingPw  bool       `db:"has_pending_pw"`
	ErrorMessage  *string    `db:"error_message"`
	AutoRotateAt  *time.Time `db:"auto_rotate_at"`
	Deleted       bool       `db:"deleted"`
}

// testRecoveryLockResetOnMDMReEnrollment verifies that MDMResetEnrollment soft-deletes
// the host's host_recovery_key_passwords row and nulls out rotation/view state that would
// otherwise leak into a future re-enrolled password. The row is kept (deleted=1) as a
// troubleshooting safeguard; all live readers filter deleted=0 so it behaves as absent.
func testRecoveryLockResetOnMDMReEnrollment(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// readRaw returns the full row bypassing the deleted=0 filter used by production readers.
	readRaw := func(t *testing.T, hostUUID string) *recoveryLockRawRow {
		t.Helper()
		var row recoveryLockRawRow
		err := ds.writer(ctx).GetContext(ctx, &row, `
			SELECT
				status,
				operation_type,
				encrypted_password IS NOT NULL AS has_password,
				pending_encrypted_password IS NOT NULL AS has_pending_pw,
				error_message,
				auto_rotate_at,
				deleted
			FROM host_recovery_key_passwords
			WHERE host_uuid = ?`, hostUUID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		require.NoError(t, err)
		return &row
	}

	setupHost := func(t *testing.T, name, uuid string) *fleet.Host {
		t.Helper()
		host := test.NewHost(t, ds, name, "1.2.7."+uuid[:3], name+"key", uuid, time.Now())
		nanoEnroll(t, ds, host, false)
		return host
	}

	t.Run("soft-deletes verified install row", func(t *testing.T) {
		host := setupHost(t, "reset-verified", "resetverifuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)

		require.NoError(t, ds.MDMResetEnrollment(ctx, host.UUID, false))

		// Live reader sees nothing.
		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		assert.Nil(t, status)

		// Raw row persists with deleted=1 for support diagnostics.
		row := readRaw(t, host.UUID)
		require.NotNil(t, row)
		assert.True(t, row.Deleted)
		assert.True(t, row.HasPassword, "encrypted_password preserved for diagnostics")
	})

	t.Run("soft-deletes stuck-pending install row (ClearQueue scenario)", func(t *testing.T) {
		host := setupHost(t, "reset-stuck-pending", "resetstuckuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		// status is 'pending' after SetHostsRecoveryLockPasswords — simulates the command
		// that was abandoned by nanomdm's ClearQueue before being acked.

		require.NoError(t, ds.MDMResetEnrollment(ctx, host.UUID, false))

		row := readRaw(t, host.UUID)
		require.NotNil(t, row)
		assert.True(t, row.Deleted)
	})

	t.Run("nulls pending rotation fields on soft-delete", func(t *testing.T) {
		host := setupHost(t, "reset-pending-rotation", "resetrotuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)
		require.NoError(t, ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), apple_mdm.GenerateRecoveryLockPassword()))

		// Sanity: the in-flight rotation has staged a pending password.
		before := readRaw(t, host.UUID)
		require.NotNil(t, before)
		require.True(t, before.HasPendingPw)

		require.NoError(t, ds.MDMResetEnrollment(ctx, host.UUID, false))

		after := readRaw(t, host.UUID)
		require.NotNil(t, after)
		assert.True(t, after.Deleted)
		assert.False(t, after.HasPendingPw, "pending_encrypted_password must be nulled to prevent re-animation leak")
	})

	t.Run("nulls auto_rotate_at on soft-delete", func(t *testing.T) {
		host := setupHost(t, "reset-auto-rotate", "resetautorotuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)
		_, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		before := readRaw(t, host.UUID)
		require.NotNil(t, before)
		require.NotNil(t, before.AutoRotateAt, "auto_rotate_at set after MarkRecoveryLockPasswordViewed")

		require.NoError(t, ds.MDMResetEnrollment(ctx, host.UUID, false))

		after := readRaw(t, host.UUID)
		require.NotNil(t, after)
		assert.True(t, after.Deleted)
		assert.Nil(t, after.AutoRotateAt, "auto_rotate_at must be nulled so cron does not fire auto-rotation against a freshly re-set password")
	})

	t.Run("re-animation after soft-delete yields clean state", func(t *testing.T) {
		host := setupHost(t, "reset-reanimate", "resetreanuuid"[:13])
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)
		require.NoError(t, ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), apple_mdm.GenerateRecoveryLockPassword()))
		markRecoveryLockFailed(t, ds, host.UUID, "boom")
		_, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		// Re-enroll wipes the row.
		require.NoError(t, ds.MDMResetEnrollment(ctx, host.UUID, false))

		// Simulate the next recovery-lock cron tick re-enqueuing a fresh SetRecoveryLock.
		newPw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: newPw}}))

		after := readRaw(t, host.UUID)
		require.NotNil(t, after)
		assert.False(t, after.Deleted, "row re-animated with deleted=0")
		assert.True(t, after.HasPassword)
		assert.True(t, after.HasPendingPw, "the fresh password is staged as pending")
		assert.Nil(t, after.AutoRotateAt, "view state must not leak across re-enrollment")
		assert.Nil(t, after.ErrorMessage, "old error_message is cleared by ON DUPLICATE KEY UPDATE")
		require.NotNil(t, after.Status)
		assert.Equal(t, string(fleet.MDMDeliveryPending), *after.Status, "re-animated row is pending awaiting the new command")
		assert.Equal(t, string(fleet.MDMOperationTypeInstall), after.OperationType)

		// Live reader now sees the re-animated row.
		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusPending, *status.Status)
		assert.False(t, status.PasswordAvailable, "the re-animated password is pending, not yet verified on the device")
		assert.Empty(t, status.Detail)
	})

	t.Run("preserves row during SCEP renewal", func(t *testing.T) {
		host := setupHost(t, "reset-scep", "resetscepuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)

		require.NoError(t, ds.MDMResetEnrollment(ctx, host.UUID, true /* scepRenewalInProgress */))

		// Row untouched: still visible to live readers as verified.
		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, status)
		status.PopulateStatus()
		require.NotNil(t, status.Status)
		assert.Equal(t, fleet.RecoveryLockStatusVerified, *status.Status)

		row := readRaw(t, host.UUID)
		require.NotNil(t, row)
		assert.False(t, row.Deleted)
		assert.True(t, row.HasPassword)
	})
}

// testDeleteHostPreservesRecoveryLockPassword locks in the intentional non-cascade of
// host_recovery_key_passwords across host deletion. The device may still be enrolled in MDM
// with the password intact, and Orbit re-enrollment recreates the host row and reuses the
// existing password record.
func testDeleteHostPreservesRecoveryLockPassword(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host := test.NewHost(t, ds, "delete-rlp", "1.2.7.200", "deleterlpkey", "deletelppuuid", time.Now())
	pw := apple_mdm.GenerateRecoveryLockPassword()
	require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
	markRecoveryLockVerified(t, ds, host.UUID)

	type rawRow struct {
		Encrypted []byte `db:"encrypted_password"`
		Deleted   bool   `db:"deleted"`
	}

	// Capture the encrypted bytes so we can assert the row survives untouched.
	var before rawRow
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &before,
		`SELECT encrypted_password, deleted FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID))
	require.NotEmpty(t, before.Encrypted)
	require.False(t, before.Deleted)

	require.NoError(t, ds.DeleteHost(ctx, host.ID))

	// Row still there (bypass deleted=0 filter), and the encrypted_password is byte-identical.
	var after rawRow
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &after,
		`SELECT encrypted_password, deleted FROM host_recovery_key_passwords WHERE host_uuid = ?`, host.UUID))
	assert.Equal(t, before.Encrypted, after.Encrypted, "encrypted_password must survive host deletion byte-for-byte")
	assert.False(t, after.Deleted, "deleted flag must not be flipped by DeleteHost")
}

// testHostRecoveryLockStatusMatrix locks in the host-detail API contract for every
// observable (status, operation_type, encrypted_password, pending_encrypted_password, deleted)
// state. This protects the UI from silent regressions in GetHostRecoveryLockPasswordStatus or
// PopulateStatus.
func testHostRecoveryLockStatusMatrix(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	type matrixCase struct {
		name                   string
		status                 *fleet.MDMDeliveryStatus // nil = SQL NULL
		operationType          fleet.MDMOperationType
		hasPassword            bool
		hasPendingPw           bool
		deleted                bool
		errorMessage           string
		expectNilFromDatastore bool
		expectPopulatedStatus  *fleet.RecoveryLockStatus
		// Only a verified row offers its password: until the device confirms the
		// SetRecoveryLock command, the stored password may not be the one on the device.
		expectPasswordAvailable bool
		expectDetail            string
	}

	cases := []matrixCase{
		{
			name:                   "soft-deleted row is invisible to readers",
			status:                 &fleet.MDMDeliveryVerified,
			operationType:          fleet.MDMOperationTypeInstall,
			hasPassword:            true,
			deleted:                true,
			expectNilFromDatastore: true,
		},
		{
			name:                    "NULL status, install, password stored -> pending",
			status:                  nil,
			operationType:           fleet.MDMOperationTypeInstall,
			hasPassword:             true,
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusPending),
			expectPasswordAvailable: false,
		},
		{
			name:                    "pending install, no rotation -> pending",
			status:                  &fleet.MDMDeliveryPending,
			operationType:           fleet.MDMOperationTypeInstall,
			hasPassword:             true,
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusPending),
			expectPasswordAvailable: false,
		},
		{
			name:                    "verified install -> verified",
			status:                  &fleet.MDMDeliveryVerified,
			operationType:           fleet.MDMOperationTypeInstall,
			hasPassword:             true,
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusVerified),
			expectPasswordAvailable: true,
		},
		{
			name:                    "failed install -> failed with detail",
			status:                  &fleet.MDMDeliveryFailed,
			operationType:           fleet.MDMOperationTypeInstall,
			hasPassword:             true,
			errorMessage:            "device rejected",
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusFailed),
			expectPasswordAvailable: false,
			expectDetail:            "device rejected",
		},
		{
			name:                    "pending install + rotation in flight -> pending",
			status:                  &fleet.MDMDeliveryPending,
			operationType:           fleet.MDMOperationTypeInstall,
			hasPassword:             true,
			hasPendingPw:            true,
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusPending),
			expectPasswordAvailable: false,
		},
		{
			name:                    "failed install + rotation in flight -> failed",
			status:                  &fleet.MDMDeliveryFailed,
			operationType:           fleet.MDMOperationTypeInstall,
			hasPassword:             true,
			hasPendingPw:            true,
			errorMessage:            "set failed",
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusFailed),
			expectPasswordAvailable: false,
			expectDetail:            "set failed",
		},
		{
			name:                    "pending remove -> removing_enforcement",
			status:                  &fleet.MDMDeliveryPending,
			operationType:           fleet.MDMOperationTypeRemove,
			hasPassword:             true,
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusRemovingEnforcement),
			expectPasswordAvailable: false,
		},
		{
			name:                    "NULL status remove -> removing_enforcement (clear retry)",
			status:                  nil,
			operationType:           fleet.MDMOperationTypeRemove,
			hasPassword:             true,
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusRemovingEnforcement),
			expectPasswordAvailable: false,
		},
		{
			name:                    "failed remove -> failed",
			status:                  &fleet.MDMDeliveryFailed,
			operationType:           fleet.MDMOperationTypeRemove,
			hasPassword:             true,
			errorMessage:            "clear failed",
			expectPopulatedStatus:   new(fleet.RecoveryLockStatusFailed),
			expectPasswordAvailable: false,
			expectDetail:            "clear failed",
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uuid := fmt.Sprintf("matrixuuid%d", i)
			host := test.NewHost(t, ds, fmt.Sprintf("matrix-host-%d", i), fmt.Sprintf("1.2.8.%d", i+1), fmt.Sprintf("matrixkey%d", i), uuid, time.Now())

			var encryptedPw, pendingPw any
			if tc.hasPassword {
				var err error
				encryptedPw, err = encrypt([]byte("password-bytes"), ds.serverPrivateKey)
				require.NoError(t, err)
			}
			if tc.hasPendingPw {
				var err error
				pendingPw, err = encrypt([]byte("pending-bytes"), ds.serverPrivateKey)
				require.NoError(t, err)
			}

			var statusArg any
			if tc.status != nil {
				statusArg = string(*tc.status)
			}

			var errMsgArg any
			if tc.errorMessage != "" {
				errMsgArg = tc.errorMessage
			}

			_, err := ds.writer(ctx).ExecContext(ctx, `
				INSERT INTO host_recovery_key_passwords
					(host_uuid, encrypted_password, pending_encrypted_password, status, operation_type, error_message, deleted)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				host.UUID, encryptedPw, pendingPw, statusArg, string(tc.operationType), errMsgArg, tc.deleted)
			require.NoError(t, err)

			got, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
			require.NoError(t, err)

			if tc.expectNilFromDatastore {
				require.Nil(t, got, "readers must filter deleted=0 and return nil")
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tc.expectPasswordAvailable, got.PasswordAvailable)
			assert.Equal(t, tc.expectDetail, got.Detail)

			got.PopulateStatus()
			if tc.expectPopulatedStatus == nil {
				assert.Nil(t, got.Status)
			} else {
				require.NotNil(t, got.Status)
				assert.Equal(t, *tc.expectPopulatedStatus, *got.Status)
			}
		})
	}
}

// testMDMTurnOffSoftDeletesRecoveryLockPassword verifies that MDMTurnOff (the explicit
// per-host MDM unenroll path used by both device CheckOut and the admin API) soft-deletes
// the recovery-lock row. Apple removes the device-side lock when the MDM profile is
// removed, so Fleet's stored copy is no longer valid.
func testMDMTurnOffSoftDeletesRecoveryLockPassword(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	type rawRow struct {
		Encrypted    []byte     `db:"encrypted_password"`
		HasPendingPw bool       `db:"has_pending_pw"`
		AutoRotateAt *time.Time `db:"auto_rotate_at"`
		Deleted      bool       `db:"deleted"`
	}
	readRaw := func(t *testing.T, hostUUID string) *rawRow {
		t.Helper()
		var row rawRow
		err := ds.writer(ctx).GetContext(ctx, &row, `
			SELECT
				encrypted_password,
				pending_encrypted_password IS NOT NULL AS has_pending_pw,
				auto_rotate_at,
				deleted
			FROM host_recovery_key_passwords
			WHERE host_uuid = ?`, hostUUID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		require.NoError(t, err)
		return &row
	}

	setupEnrolledHost := func(t *testing.T, name, uuid string) *fleet.Host {
		t.Helper()
		host := test.NewHost(t, ds, name, "1.2.9."+uuid[:3], name+"key", uuid, time.Now())
		nanoEnroll(t, ds, host, false)
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://mdm.example.com", false, "Fleet", "", false))
		return host
	}

	t.Run("soft-deletes verified recovery lock and clears volatile state", func(t *testing.T) {
		host := setupEnrolledHost(t, "turnoff-verified", "turnoffverifuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)
		require.NoError(t, ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), apple_mdm.GenerateRecoveryLockPassword()))
		markRecoveryLockFailed(t, ds, host.UUID, "boom")
		_, err := ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID)
		require.NoError(t, err)

		_, _, err = ds.MDMTurnOff(ctx, host.UUID)
		require.NoError(t, err)

		// Live reader sees nothing.
		status, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
		require.NoError(t, err)
		assert.Nil(t, status)

		// Raw row persists with deleted=1 and volatile state nulled.
		row := readRaw(t, host.UUID)
		require.NotNil(t, row)
		assert.True(t, row.Deleted)
		assert.NotEmpty(t, row.Encrypted, "encrypted_password preserved for diagnostics")
		assert.False(t, row.HasPendingPw, "pending rotation must be nulled")
		assert.Nil(t, row.AutoRotateAt, "view state must be nulled")
	})

	t.Run("idempotent: second MDMTurnOff is a no-op on the already-soft-deleted row", func(t *testing.T) {
		host := setupEnrolledHost(t, "turnoff-idempotent", "turnoffidempuuid")
		pw := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
		markRecoveryLockVerified(t, ds, host.UUID)

		_, _, err := ds.MDMTurnOff(ctx, host.UUID)
		require.NoError(t, err)

		first := readRaw(t, host.UUID)
		require.NotNil(t, first)
		require.True(t, first.Deleted)
		firstBytes := first.Encrypted

		// Without re-enrolling (just calling MDMTurnOff again), the deleted=0 guard in
		// the helper makes the soft-delete a true no-op — the row is unchanged.
		_, _, err = ds.MDMTurnOff(ctx, host.UUID)
		require.NoError(t, err)

		second := readRaw(t, host.UUID)
		require.NotNil(t, second)
		assert.True(t, second.Deleted)
		assert.Equal(t, firstBytes, second.Encrypted, "second turn-off must not modify the encrypted_password kept for diagnostics")
	})

	t.Run("no-op when host has no recovery lock row", func(t *testing.T) {
		host := setupEnrolledHost(t, "turnoff-no-row", "turnoffnorouuid")
		// No SetHostsRecoveryLockPasswords call — row never existed.

		_, _, err := ds.MDMTurnOff(ctx, host.UUID)
		require.NoError(t, err)

		row := readRaw(t, host.UUID)
		assert.Nil(t, row, "no row should be created by MDMTurnOff")
	})
}

// testMDMTurnOffSoftDeletesMDMCertificates verifies unenroll clears MDM-origin
// certs but leaves osquery-origin certs intact.
func testMDMTurnOffSoftDeletesMDMCertificates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	mkCert := func(hostID uint, commonName string) *fleet.HostCertificateRecord {
		template := x509.Certificate{
			Subject:               pkix.Name{CommonName: commonName},
			Issuer:                pkix.Name{CommonName: "issuer.test.example.com"},
			SerialNumber:          big.NewInt(mathrand.Int64()), // nolint:gosec
			SignatureAlgorithm:    x509.SHA256WithRSA,
			NotBefore:             time.Now().Add(-time.Hour).Truncate(time.Second).UTC(),
			NotAfter:              time.Now().Add(24 * time.Hour).Truncate(time.Second).UTC(),
			BasicConstraintsValid: true,
		}
		certBytes, _, err := GenerateTestCertBytes(&template)
		require.NoError(t, err)
		block, _ := pem.Decode(certBytes)
		parsed, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)
		return fleet.NewHostCertificateRecord(hostID, parsed)
	}

	host := test.NewHost(t, ds, "turnoff-certs", "1.2.3.45", "turnoffcertskey", "turnoffcertsuuid", time.Now())
	nanoEnroll(t, ds, host, false)
	require.NoError(t, ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://mdm.example.com", false, "Fleet", "", false))

	require.NoError(t, ds.UpdateHostCertificates(ctx, host.ID, host.UUID,
		[]*fleet.HostCertificateRecord{mkCert(host.ID, "osquery.example.com")}, fleet.HostCertificateOriginOsquery, nil))
	require.NoError(t, ds.UpdateHostCertificates(ctx, host.ID, host.UUID,
		[]*fleet.HostCertificateRecord{mkCert(host.ID, "mdm-acme.example.com")}, fleet.HostCertificateOriginMDM, nil))

	certs, _, err := ds.ListHostCertificates(ctx, host.ID, fleet.ListOptions{OrderKey: "common_name"})
	require.NoError(t, err)
	require.Len(t, certs, 2)

	_, _, err = ds.MDMTurnOff(ctx, host.UUID)
	require.NoError(t, err)

	// Only the osquery-origin cert remains; the MDM-origin cert is soft-deleted.
	certs, _, err = ds.ListHostCertificates(ctx, host.ID, fleet.ListOptions{OrderKey: "common_name"})
	require.NoError(t, err)
	require.Len(t, certs, 1)
	require.Equal(t, "osquery.example.com", certs[0].CommonName)
}

// testRecoveryLockReadersReturnNotFoundForSoftDeleted verifies that view-password and
// rotation-status readers surface notFound for soft-deleted rows. The EE rotate endpoint
// depends on the notFound-from-GetRecoveryLockRotationStatus branch to return
// "Host does not have a recovery lock password to rotate."
func testRecoveryLockReadersReturnNotFoundForSoftDeleted(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host := test.NewHost(t, ds, "softdel-read", "1.2.8.250", "softdelreadkey", "softdelreaduuid", time.Now())
	pw := apple_mdm.GenerateRecoveryLockPassword()
	require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{{HostUUID: host.UUID, Password: pw}}))
	markRecoveryLockVerified(t, ds, host.UUID)

	// Soft-delete directly (simulates MDMResetEnrollment without the full lifecycle setup).
	_, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_recovery_key_passwords SET deleted = 1 WHERE host_uuid = ?`, host.UUID)
	require.NoError(t, err)

	// GetHostRecoveryLockPassword (view password endpoint source) must surface notFound.
	_, err = ds.GetHostRecoveryLockPassword(ctx, host.UUID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "view-password reader must surface notFound so the UI treats it as absent")

	// GetRecoveryLockRotationStatus (rotation endpoint prerequisite) must surface notFound so
	// the EE service returns BadRequest "Host does not have a recovery lock password to rotate."
	_, err = ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "rotation-status reader must surface notFound")

	// GetPendingRecoveryLock returns (nil, nil) for missing/deleted rows.
	pending, err := ds.GetPendingRecoveryLock(ctx, host.UUID)
	require.NoError(t, err)
	assert.Nil(t, pending)

	// GetHostRecoveryLockPasswordStatus (host detail API source) returns nil so the JSON
	// field is omitted entirely.
	got, err := ds.GetHostRecoveryLockPasswordStatus(ctx, host.UUID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// testRecoveryLockPasswordArchive covers the archive's contract: it records every password
// Fleet puts on the wire (not just the ones a device confirms), and it shrinks only when a
// device proves which password it holds.
func testRecoveryLockPasswordArchive(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// sendPassword does what the cron does: generate, store as pending, archive.
	sendPassword := func(t *testing.T, hostUUID string) (string, string) {
		t.Helper()
		pw, cmdUUID := apple_mdm.GenerateRecoveryLockPassword(), uuid.NewString()
		require.NoError(t, ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{
			{HostUUID: hostUUID, Password: pw, PendingSetCommandUUID: cmdUUID},
		}))
		return pw, cmdUUID
	}

	// The archive has no read path in the datastore on purpose — it is a backstop queried
	// directly against the DB when a host's recovery lock has gone wrong.
	type archivedRow struct {
		EncryptedPassword []byte         `db:"encrypted_password"`
		SetCommandUUID    sql.NullString `db:"set_command_uuid"`
	}
	archivedRows := func(t *testing.T, hostUUID string) []archivedRow {
		t.Helper()
		var rows []archivedRow
		require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &rows, `
			SELECT encrypted_password, set_command_uuid
			FROM host_recovery_key_password_archive
			WHERE host_uuid = ? ORDER BY id DESC`, hostUUID))
		return rows
	}

	archivedPasswords := func(t *testing.T, hostUUID string) []string {
		t.Helper()
		rows := archivedRows(t, hostUUID)
		out := make([]string, 0, len(rows))
		for _, row := range rows {
			decrypted, err := decrypt(row.EncryptedPassword, ds.serverPrivateKey)
			require.NoError(t, err)
			out = append(out, string(decrypted))
		}
		return out
	}

	t.Run("every generated password is archived, including ones no device confirmed", func(t *testing.T) {
		host := test.NewHost(t, ds, "archive-host", "1.2.9.1", "archivekey", "archiveuuid", time.Now())

		// A set, then two retries: the cron generates a fresh password each time, and the
		// two discarded ones may already be on the device.
		first, firstCmdUUID := sendPassword(t, host.UUID)
		second, _ := sendPassword(t, host.UUID)
		third, thirdCmdUUID := sendPassword(t, host.UUID)

		assert.Equal(t, []string{third, second, first}, archivedPasswords(t, host.UUID),
			"a password overwritten by a retry is still a password the device may hold")

		// Each row carries the command that would have delivered it, which is how a
		// support query ties a candidate back to what the device was asked to do.
		rows := archivedRows(t, host.UUID)
		require.Len(t, rows, 3)
		assert.Equal(t, thirdCmdUUID, rows[0].SetCommandUUID.String)
		assert.Equal(t, firstCmdUUID, rows[2].SetCommandUUID.String)
	})

	t.Run("a confirmed password prunes everything older", func(t *testing.T) {
		host := test.NewHost(t, ds, "archive-prune-host", "1.2.9.2", "prunekey", "pruneuuid", time.Now())

		sendPassword(t, host.UUID)
		sendPassword(t, host.UUID)
		confirmed, _ := sendPassword(t, host.UUID)
		require.Len(t, archivedPasswords(t, host.UUID), 3)

		// The device answers that it holds the third one, which rules out the first two.
		markRecoveryLockVerified(t, ds, host.UUID)
		assert.Equal(t, []string{confirmed}, archivedPasswords(t, host.UUID))

		// A rotation adds a candidate; confirming it collapses the archive again. This is
		// what keeps a healthy host at one row however often it rotates.
		rotated := apple_mdm.GenerateRecoveryLockPassword()
		require.NoError(t, ds.InitiateRecoveryLockRotation(ctx, host.UUID, uuid.NewString(), rotated))
		assert.Equal(t, []string{rotated, confirmed}, archivedPasswords(t, host.UUID))

		markRecoveryLockVerified(t, ds, host.UUID)
		assert.Equal(t, []string{rotated}, archivedPasswords(t, host.UUID))
	})

	t.Run("a stale verify result confirms nothing and prunes nothing", func(t *testing.T) {
		host := test.NewHost(t, ds, "archive-stale-host", "1.2.9.3", "stalearchkey", "stalearchuuid", time.Now())

		sendPassword(t, host.UUID)
		pending, err := ds.GetPendingRecoveryLock(ctx, host.UUID)
		require.NoError(t, err)
		require.NotNil(t, pending.PendingSetCommandUUID)
		require.NoError(t, ds.SetRecoveryLockVerifying(ctx, host.UUID, *pending.PendingSetCommandUUID, uuid.NewString()))
		sendPassword(t, host.UUID)
		require.Len(t, archivedPasswords(t, host.UUID), 2)

		// A result for a command the row is no longer waiting on must not be read as proof.
		require.NoError(t, ds.SetRecoveryLockVerified(ctx, host.UUID, uuid.NewString()))
		assert.Len(t, archivedPasswords(t, host.UUID), 2)
	})

	t.Run("verifying the last known password keeps the rejected candidate", func(t *testing.T) {
		host := test.NewHost(t, ds, "archive-fallback-host", "1.2.9.4", "fbkey", "fbuuid", time.Now())

		sendPassword(t, host.UUID)
		markRecoveryLockVerified(t, ds, host.UUID)
		active := archivedPasswords(t, host.UUID)
		require.Len(t, active, 1)

		// Re-enrollment path: the row is soft-deleted, the cron re-SETs as if the host were
		// fresh, and the device rejects it for a password mismatch.
		require.NoError(t, softDeleteHostRecoveryLockPassword(ctx, ds.writer(ctx), host.UUID))
		rejected, rejectedCmdUUID := sendPassword(t, host.UUID)

		verifyCmdUUID := uuid.NewString()
		require.NoError(t, ds.SetRecoveryLockVerifyingLastKnownPassword(ctx, host.UUID, rejectedCmdUUID, verifyCmdUUID))
		require.NoError(t, ds.SetRecoveryLockVerified(ctx, host.UUID, verifyCmdUUID))

		// The confirmed password is the older one, so the prune rules out nothing. The
		// rejected candidate is kept on purpose: the archive only drops what a device has
		// positively ruled out, and it is bounded by the per-host cap either way.
		assert.Equal(t, []string{rejected, active[0]}, archivedPasswords(t, host.UUID))
	})

	t.Run("a host with no archive reads empty", func(t *testing.T) {
		assert.Empty(t, archivedRows(t, "no-such-host-uuid"))
	})

	t.Run("a host that never confirms is capped", func(t *testing.T) {
		host := test.NewHost(t, ds, "archive-cap-host", "1.2.9.5", "capkey", "capuuid", time.Now())

		var newest string
		for range maxArchivedRecoveryLockPasswordsPerHost + 5 {
			newest, _ = sendPassword(t, host.UUID)
		}

		archived := archivedPasswords(t, host.UUID)
		require.Len(t, archived, maxArchivedRecoveryLockPasswordsPerHost)
		assert.Equal(t, newest, archived[0], "the cap must drop the oldest candidates, not the newest")
	})

	t.Run("the archive outlives the host", func(t *testing.T) {
		host := test.NewHost(t, ds, "archive-outlives-host", "1.2.9.6", "outliveskey", "outlivesuuid", time.Now())

		sent, _ := sendPassword(t, host.UUID)

		// Unenroll soft-deletes the live row, which is what makes the password unreadable
		// through the normal API — while the device may still be locked with it.
		require.NoError(t, softDeleteHostRecoveryLockPassword(ctx, ds.writer(ctx), host.UUID))
		assert.Equal(t, []string{sent}, archivedPasswords(t, host.UUID))

		require.NoError(t, ds.DeleteHost(ctx, host.ID))
		assert.Equal(t, []string{sent}, archivedPasswords(t, host.UUID),
			"a deleted host can still be holding a lock Fleet handed it")
	})
}
