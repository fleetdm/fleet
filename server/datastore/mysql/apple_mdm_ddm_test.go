package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDMDDMApple(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"StoreDDMStatusReportSkipsRemoveRows", testStoreDDMStatusReportSkipsRemoveRows},
		{"CleanUpDuplicateRemoveInstallAcrossBatches", testCleanUpDuplicateRemoveInstallAcrossBatches},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// Helper function to set up device and enrollment records for a host
func setupMDMDeviceAndEnrollment(t *testing.T, ds *Datastore, ctx context.Context, hostUUID, hardwareSerial string) {
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate) VALUES (?, ?, ?)`,
			hostUUID, hardwareSerial, "test")
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			hostUUID, hostUUID, "Device", "topic", "push_magic", "token_hex", 1, time.Now())
		return err
	})
}

// Helper function to insert a host declaration
func insertHostDeclaration(t *testing.T, ds *Datastore, ctx context.Context, hostUUID, declarationUUID, token, status, operationType, identifier string) {
	var statusPtr *string
	if status != "" {
		statusPtr = ptr.String(status)
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_apple_declarations
			(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier)
			VALUES (?, ?, ?, ?, UNHEX(MD5(?)), ?)`,
			hostUUID, declarationUUID, statusPtr, operationType, token, identifier)
		return err
	})
}

func testStoreDDMStatusReportSkipsRemoveRows(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a test host
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-ddm-status",
		UUID:            "test-host-uuid-ddm-status",
		HardwareSerial:  "ABC123-DDM-STATUS",
		PrimaryIP:       "192.168.1.50",
		PrimaryMac:      "00:00:00:00:00:50",
		OsqueryHostID:   ptr.String("test-host-uuid-ddm-status"),
		NodeKey:         ptr.String("test-host-uuid-ddm-status"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	setupMDMDeviceAndEnrollment(t, ds, ctx, host.UUID, host.HardwareSerial)

	// Insert two rows into host_mdm_apple_declarations:
	// Row A: a remove-operation row (simulating a declaration pending removal)
	// Row B: a normal install-operation row
	insertHostDeclaration(t, ds, ctx, host.UUID, "decl-remove", "shared-token", "pending", "remove", "com.example.remove")
	insertHostDeclaration(t, ds, ctx, host.UUID, "decl-install", "install-token", "pending", "install", "com.example.install")

	// Query back the HEX tokens from the DB (MDMAppleStoreDDMStatusReport reads tokens as HEX)
	type tokenRow struct {
		DeclarationUUID string `db:"declaration_uuid"`
		Token           string `db:"token"`
	}
	var tokens []tokenRow
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &tokens, `
			SELECT declaration_uuid, HEX(token) as token
			FROM host_mdm_apple_declarations
			WHERE host_uuid = ?
			ORDER BY declaration_uuid`, host.UUID)
	})
	require.Len(t, tokens, 2)

	tokenByDecl := make(map[string]string, 2)
	for _, tok := range tokens {
		tokenByDecl[tok.DeclarationUUID] = tok.Token
	}
	tokenA := tokenByDecl["decl-remove"]
	tokenB := tokenByDecl["decl-install"]
	require.NotEmpty(t, tokenA)
	require.NotEmpty(t, tokenB)

	// Build the updates slice as if the device is reporting status.
	// The device always reports operation_type='install' for all declarations.
	updates := []*fleet.MDMAppleHostDeclaration{
		{
			Token:         tokenA,
			Status:        new(fleet.MDMDeliveryStatus),
			OperationType: fleet.MDMOperationTypeInstall,
		},
		{
			Token:         tokenB,
			Status:        new(fleet.MDMDeliveryStatus),
			OperationType: fleet.MDMOperationTypeInstall,
		},
	}
	*updates[0].Status = fleet.MDMDeliveryVerified
	*updates[1].Status = fleet.MDMDeliveryVerified

	// Call the method under test
	err = ds.MDMAppleStoreDDMStatusReport(ctx, host.UUID, updates)
	require.NoError(t, err)

	// Assert the end state.
	type resultRow struct {
		DeclarationUUID string `db:"declaration_uuid"`
		Token           string `db:"token"`
		Status          string `db:"status"`
		OperationType   string `db:"operation_type"`
	}
	var remaining []resultRow
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &remaining, `
			SELECT declaration_uuid, HEX(token) as token, status, operation_type
			FROM host_mdm_apple_declarations
			WHERE host_uuid = ?
			ORDER BY declaration_uuid`, host.UUID)
	})

	// With the fix: only the install row should remain (remove row was skipped then deleted)
	// With the bug: both rows remain, and the remove row was flipped to install/verified
	require.Len(t, remaining, 1, "expected remove row to not survive as install/verified — this is the token collision bug")
	assert.Equal(t, "decl-install", remaining[0].DeclarationUUID)
	assert.Equal(t, "install", remaining[0].OperationType)
	assert.Equal(t, "verified", remaining[0].Status)
}

func testCleanUpDuplicateRemoveInstallAcrossBatches(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// This exercises cleanUpDuplicateRemoveInstall, which runs inside
	// BulkUpsertMDMAppleHostDeclarations after each batch of host declaration
	// rows is written. The scenario:
	//   - D1 is the "old" declaration already marked for removal
	//     (operation_type=remove, status=pending) on each host.
	//   - D2 is the "new" declaration (same content/token, different UUID — e.g.
	//     after a name change caused a delete+reinsert) now being installed.
	// Because the remove and install share a token, the host has nothing to do, so
	// the cleanup deletes the stale remove row and marks the install as verified
	// with resync=1.
	//
	// The remove rows and install rows are written in SEPARATE bulk-upsert calls to
	// prove the cleanup matches against committed DB state, not just the rows in the
	// current batch.
	declJSON := []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.cleanup"}`)

	// Create the declaration so we have a realistic token to share between the
	// remove (D1) and install (D2) host rows.
	decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		DeclarationUUID: "decl-new",
		Name:            "New Declaration",
		Identifier:      "com.example.cleanup",
		RawJSON:         declJSON,
	}, nil)
	require.NoError(t, err)

	// Read the raw (binary) token. BulkUpsertMDMAppleHostDeclarations writes the
	// Token field straight into the token column (no UNHEX), so we need the binary
	// value rather than its hex representation.
	var token []byte
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &token,
			"SELECT token FROM mdm_apple_declarations WHERE declaration_uuid = ?", decl.DeclarationUUID)
	})

	// Create 3 hosts, all enrolled
	hosts := make([]*fleet.Host, 3)
	for i := range 3 {
		hostUUID := fmt.Sprintf("cleanup-host-%d", i)
		newHost, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        fmt.Sprintf("cleanup-host-%d", i),
			UUID:            hostUUID,
			HardwareSerial:  fmt.Sprintf("CLEANUP-%d", i),
			PrimaryIP:       fmt.Sprintf("192.168.10.%d", i+1),
			PrimaryMac:      fmt.Sprintf("00:00:00:00:10:%02d", i+1),
			OsqueryHostID:   ptr.String(hostUUID),
			NodeKey:         ptr.String(hostUUID),
			DetailUpdatedAt: time.Now(),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		require.NotNil(t, newHost)
		hosts[i] = newHost
		setupMDMDeviceAndEnrollment(t, ds, ctx, hostUUID, hosts[i].HardwareSerial)
	}

	pending := fleet.MDMDeliveryPending

	// First bulk upsert: write the stale "remove" rows for the old declaration (D1).
	// cleanUpDuplicateRemoveInstall is a no-op for this batch because it contains no
	// installs.
	removeRows := make([]*fleet.MDMAppleHostDeclaration, 0, len(hosts))
	for _, h := range hosts {
		removeRows = append(removeRows, &fleet.MDMAppleHostDeclaration{
			HostUUID:        h.UUID,
			DeclarationUUID: "decl-old",
			Name:            "Old Declaration",
			Identifier:      decl.Identifier,
			Status:          &pending,
			OperationType:   fleet.MDMOperationTypeRemove,
			Token:           string(token),
		})
	}
	require.NoError(t, ds.BulkUpsertMDMAppleHostDeclarations(ctx, removeRows))

	// Second bulk upsert: write the "install" rows for the new declaration (D2). The
	// cleanup runs after this batch and must find the matching remove rows committed
	// by the first batch.
	installRows := make([]*fleet.MDMAppleHostDeclaration, 0, len(hosts))
	for _, h := range hosts {
		installRows = append(installRows, &fleet.MDMAppleHostDeclaration{
			HostUUID:        h.UUID,
			DeclarationUUID: decl.DeclarationUUID,
			Name:            decl.Name,
			Identifier:      decl.Identifier,
			Status:          &pending,
			OperationType:   fleet.MDMOperationTypeInstall,
			Token:           string(token),
		})
	}
	require.NoError(t, ds.BulkUpsertMDMAppleHostDeclarations(ctx, installRows))

	// Assert: for each host, the cleanup should have:
	//   1. Deleted the D1 remove row (same token as D2 install — duplicate remove/install)
	//   2. Marked D2 install as verified with resync=1
	type declRow struct {
		DeclarationUUID string  `db:"declaration_uuid"`
		OperationType   string  `db:"operation_type"`
		Status          string  `db:"status"`
		Resync          bool    `db:"resync"`
		Token           *string `db:"token"`
	}
	for _, h := range hosts {
		var rows []declRow
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows, `
				SELECT declaration_uuid, operation_type, COALESCE(status, '') as status, resync, HEX(token) as token
				FROM host_mdm_apple_declarations
				WHERE host_uuid = ?
				ORDER BY declaration_uuid`, h.UUID)
		})

		// Build a map by declaration UUID for easier assertions
		rowByDecl := make(map[string]declRow, len(rows))
		for _, r := range rows {
			rowByDecl[r.DeclarationUUID] = r
		}

		// D1 remove row should NOT exist (cleaned up because same token as D2 install)
		_, d1Exists := rowByDecl["decl-old"]
		assert.False(t, d1Exists, "host %s: D1 remove row should have been cleaned up (same token as D2 install)", h.UUID)

		// D2 install row should exist as verified with resync=1
		d2Row, d2Exists := rowByDecl[decl.DeclarationUUID]
		if assert.True(t, d2Exists, "host %s: D2 install row should exist", h.UUID) {
			assert.Equal(t, "install", d2Row.OperationType, "host %s: D2 should be install", h.UUID)
			assert.Equal(t, "verified", d2Row.Status, "host %s: D2 should be marked verified by cleanup", h.UUID)
			assert.True(t, d2Row.Resync, "host %s: D2 should have resync=1", h.UUID)
		}
	}
}
