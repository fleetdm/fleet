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
		{"TestMDMAppleBatchSetHostDeclarationState", testMDMAppleBatchSetHostDeclarationState},
		{"StoreDDMStatusReportSkipsRemoveRows", testStoreDDMStatusReportSkipsRemoveRows},
		{"CleanUpDuplicateRemoveInstallAcrossBatches", testCleanUpDuplicateRemoveInstallAcrossBatches},
		{"CleanUpOrphanedPendingRemoves", testCleanUpOrphanedPendingRemoves},
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

// Helper function to check declaration status
func checkDeclarationStatus(t *testing.T, ds *Datastore, ctx context.Context, hostUUID, declarationUUID, expectedStatus, operation string) {
	var status string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		db := q.(*sqlx.DB)
		return db.QueryRowContext(ctx, `
			SELECT status FROM host_mdm_apple_declarations
			WHERE host_uuid = ? AND declaration_uuid = ? AND operation_type = ?`,
			hostUUID, declarationUUID, operation).Scan(&status)
	})
	assert.Equal(t, expectedStatus, status)
}

func testMDMAppleBatchSetHostDeclarationState(t *testing.T, ds *Datastore) {
	t.Run("BasicTest", func(t *testing.T) {
		ctx := t.Context()

		// Create a test host
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        "test-host-ddm",
			UUID:            "test-host-uuid-ddm",
			HardwareSerial:  "ABC123-DDM",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "00:00:00:00:00:00",
			OsqueryHostID:   ptr.String("test-host-uuid-ddm"),
			NodeKey:         ptr.String("test-host-uuid-ddm"),
			DetailUpdatedAt: time.Now(),
			Platform:        "darwin",
		})
		require.NoError(t, err)

		// Set up device and enrollment records (required for foreign key constraints)
		setupMDMDeviceAndEnrollment(t, ds, ctx, host.UUID, host.HardwareSerial)

		// Create 6 declarations (3 for install, 3 for remove)
		declarations := make([]*fleet.MDMAppleDeclaration, 3)
		for i := 0; i < 3; i++ {
			declarations[i], err = ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
				DeclarationUUID: "test-declaration-uuid-" + string(rune('A'+i)),
				Name:            "Test Declaration " + string(rune('A'+i)),
				Identifier:      "com.example.test.declaration." + string(rune('A'+i)),
				RawJSON:         []byte(`{"Type":"com.apple.test.declaration","Identifier":"com.example.test.declaration.` + string(rune('A'+i)) + `"}`),
			}, nil)
			require.NoError(t, err)
		}
		removeDeclarations := make([]*fleet.MDMAppleDeclaration, 3)
		for i := 0; i < 3; i++ {
			removeDeclarations[i] = &fleet.MDMAppleDeclaration{
				DeclarationUUID: "test-remove-declaration-uuid-" + string(rune('A'+i)),
				Name:            "Test Remove Declaration " + string(rune('A'+i)),
				Identifier:      "com.example.test.remove.declaration." + string(rune('A'+i)),
				RawJSON:         []byte(`{"Type":"com.apple.test.declaration","Identifier":"com.example.test.remove.declaration.` + string(rune('A'+i)) + `"}`),
			}
		}

		// Don't insert the install declarations in host_mdm_apple_declarations
		// so they get picked up as new declarations to install

		// Insert 3 remove declarations with verified status
		// These simulate declarations that were previously installed but no longer
		// exist in mdm_apple_declarations (hence should be removed)
		for i := 0; i < 3; i++ {
			// Use a proper hex token for each remove declaration
			token := fmt.Sprintf("%032x", i+1000) // 32 hex chars = 16 bytes when unhexed
			insertHostDeclaration(
				t, ds, ctx,
				host.UUID,
				removeDeclarations[i].DeclarationUUID,
				token,
				"verified",
				"install", // should get converted to "remove"
				removeDeclarations[i].Identifier,
			)
		}

		// Call the method under test
		hostUUIDs, err := ds.MDMAppleBatchSetHostDeclarationState(ctx)
		require.NoError(t, err)
		require.Contains(t, hostUUIDs, host.UUID)

		// Also verify that the 3 remove declarations have been marked as pending
		for i := 0; i < 3; i++ {
			checkDeclarationStatus(t, ds, ctx, host.UUID, removeDeclarations[i].DeclarationUUID, "pending", "remove")
		}
		// Verify that the 3 install declarations have been marked as pending
		for i := 0; i < 3; i++ {
			checkDeclarationStatus(t, ds, ctx, host.UUID, declarations[i].DeclarationUUID, "pending", "install")
		}
	})

	t.Run("MultipleHostsSharedTokens", func(t *testing.T) {
		ctx := t.Context()

		// Create 3 test hosts
		hosts := make([]*fleet.Host, 3)
		for i := 0; i < 3; i++ {
			hostUUID := "test-host-uuid-" + string(rune('A'+i))
			hardwareSerial := "ABC123-" + string(rune('A'+i))

			var err error
			hosts[i], err = ds.NewHost(ctx, &fleet.Host{
				Hostname:        "test-host-" + string(rune('A'+i)),
				UUID:            hostUUID,
				HardwareSerial:  hardwareSerial,
				PrimaryIP:       "192.168.1." + string(rune('1'+i)),
				PrimaryMac:      "00:00:00:00:00:0" + string(rune('1'+i)),
				OsqueryHostID:   ptr.String(hostUUID),
				NodeKey:         ptr.String(hostUUID),
				DetailUpdatedAt: time.Now(),
				Platform:        "darwin",
			})
			require.NoError(t, err)

			// Set up device and enrollment records for each host
			setupMDMDeviceAndEnrollment(t, ds, ctx, hostUUID, hardwareSerial)
		}

		// Create 3 declarations for install operations
		installDeclarations := make([]*fleet.MDMAppleDeclaration, 3)
		for i := 0; i < 3; i++ {
			var err error
			installDeclarations[i], err = ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
				DeclarationUUID: "test-install-decl-" + string(rune('A'+i)),
				Name:            "Test Install Declaration " + string(rune('A'+i)),
				Identifier:      "com.example.test.install." + string(rune('A'+i)),
				RawJSON:         []byte(`{"Type":"com.apple.test.declaration","Identifier":"com.example.test.install.` + string(rune('A'+i)) + `"}`),
			}, nil)
			require.NoError(t, err)
		}

		// Create 3 declarations for remove operations (without calling NewMDMAppleDeclaration)
		removeDeclarations := make([]*fleet.MDMAppleDeclaration, 3)
		for i := 0; i < 3; i++ {
			removeDeclarations[i] = &fleet.MDMAppleDeclaration{
				DeclarationUUID: "test-remove-decl-" + string(rune('A'+i)),
				Name:            "Test Remove Declaration " + string(rune('A'+i)),
				Identifier:      "com.example.test.remove." + string(rune('A'+i)),
				RawJSON:         []byte(`{"Type":"com.apple.test.declaration","Identifier":"com.example.test.remove.` + string(rune('A'+i)) + `"}`),
			}
		}

		// Get tokens for all declarations
		getToken := func(declarationUUID string) string {
			var token []byte
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				err := sqlx.GetContext(ctx, q, &token,
					"SELECT token FROM mdm_apple_declarations WHERE declaration_uuid = ?", declarationUUID)
				return err
			})
			return fmt.Sprintf("%x", token)
		}

		installTokens := make([]string, 3)
		for i := 0; i < 3; i++ {
			installTokens[i] = getToken(installDeclarations[i].DeclarationUUID)
			installDeclarations[i].Token = installTokens[i]
		}

		removeTokens := make([]string, 3)
		for i := 0; i < 3; i++ {
			if i < 2 {
				// First 2 remove operations use the same tokens as the first 2 install operations
				removeTokens[i] = installTokens[i]
			} else {
				// Last remove operation uses a different token
				removeTokens[i] = fmt.Sprintf("%032x", i+1000)
			}
		}

		// For each host, insert 3 install declarations and 3 remove declarations
		for _, host := range hosts {
			// We don't add install declarations because they will be added automatically

			// Insert remove declarations
			for j := 0; j < 3; j++ {
				insertHostDeclaration(
					t, ds, ctx,
					host.UUID,
					removeDeclarations[j].DeclarationUUID,
					removeTokens[j],
					"verified", // verified status
					"install",  // should get converted to "remove"
					removeDeclarations[j].Identifier,
				)
			}
		}

		// Call the method under test
		hostUUIDs, err := ds.MDMAppleBatchSetHostDeclarationState(ctx)
		require.NoError(t, err)

		// Verify that all host UUIDs are returned
		for _, host := range hosts {
			require.Contains(t, hostUUIDs, host.UUID)
		}

		// Verify that all declarations for all hosts have been marked as pending
		for _, host := range hosts {
			// Check remove declarations first
			for _, decl := range removeDeclarations {
				// All remove declarations should be marked as pending since they were inserted
				// with verified status and should be converted to remove operations
				checkDeclarationStatus(t, ds, ctx, host.UUID, decl.DeclarationUUID, "pending", "remove")
			}

			// Check install declarations
			for _, decl := range installDeclarations {
				// All install declarations should be marked as pending
				checkDeclarationStatus(t, ds, ctx, host.UUID, decl.DeclarationUUID, "pending", "install")
			}
		}
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

	// D1 simulates the "old" declaration that was already installed on hosts.
	// D2 simulates the "new" declaration (same content/identifier, different UUID — e.g. after a name change caused delete+reinsert).
	// They share the same identifier, so D1 must be deleted before D2 is created (unique constraint on team_id+identifier).
	declJSON := []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.cleanup"}`)

	d1, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		DeclarationUUID: "decl-old",
		Name:            "Old Declaration",
		Identifier:      "com.example.cleanup",
		RawJSON:         declJSON,
	}, nil)
	require.NoError(t, err)

	// Query the token for D1 from mdm_apple_declarations (generated column)
	var d1Token string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &d1Token,
			"SELECT HEX(token) FROM mdm_apple_declarations WHERE declaration_uuid = ?", d1.DeclarationUUID)
	})

	// Create 3 hosts, all enrolled
	hosts := make([]*fleet.Host, 3)
	for i := range 3 {
		hostUUID := fmt.Sprintf("cleanup-host-%d", i)
		hosts[i], err = ds.NewHost(ctx, &fleet.Host{
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
		setupMDMDeviceAndEnrollment(t, ds, ctx, hostUUID, hosts[i].HardwareSerial)
	}

	// For each host, insert a host_declaration row for D1 with status=verified, operation_type=install.
	// This simulates D1 being currently installed on each host.
	// NOTE: We use UNHEX(?) with the hex token directly (not insertHostDeclaration which does
	// UNHEX(MD5(?))) because we need the token to match the generated column in mdm_apple_declarations exactly.
	for _, h := range hosts {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				INSERT INTO host_mdm_apple_declarations
				(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier)
				VALUES (?, ?, 'verified', 'install', UNHEX(?), ?)`,
				h.UUID, d1.DeclarationUUID, d1Token, d1.Identifier)
			return err
		})
	}

	// Delete D1 from mdm_apple_declarations (simulating IT admin removing the old declaration).
	// The host_mdm_apple_declarations rows for D1 remain — the reconciler will mark them for removal.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_declarations WHERE declaration_uuid = ?", d1.DeclarationUUID)
		return err
	})

	// Now create D2 with the same identifier (possible since D1 was just deleted) and same raw_json → same token.
	d2, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		DeclarationUUID: "decl-new",
		Name:            "New Declaration",
		Identifier:      "com.example.cleanup",
		RawJSON:         declJSON,
	}, nil)
	require.NoError(t, err)

	var d2Token string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &d2Token,
			"SELECT HEX(token) FROM mdm_apple_declarations WHERE declaration_uuid = ?", d2.DeclarationUUID)
	})
	require.Equal(t, d1Token, d2Token, "declarations with same raw_json should have the same token")

	// Force a small batch size so installs and removes end up in different batches.
	// The UNION ALL query returns removes first, then installs. With 3 hosts:
	//   - 3 remove rows (D1 for each host) + 3 install rows (D2 for each host) = 6 rows
	//   - batch size 2 → batch 1: 2 removes, batch 2: 1 remove + 1 install, batch 3: 2 installs
	// When batch 1 runs cleanUpDuplicateRemoveInstall, the matching installs haven't been upserted to
	// status='pending' yet — they still have status='verified' — so the cleanup finds no match.
	ds.testUpsertMDMDesiredProfilesBatchSize = 2
	t.Cleanup(func() { ds.testUpsertMDMDesiredProfilesBatchSize = 0 })

	// Run the reconciler
	_, err = ds.MDMAppleBatchSetHostDeclarationState(ctx)
	require.NoError(t, err)

	// Assert: for each host, the cleanup should have:
	//   1. Deleted the D1 remove row (same token as D2 install — duplicate remove/install)
	//   2. Marked D2 install as verified with resync=1
	// With the bug: the D1 remove row survives because cleanup ran per-batch and missed
	// cross-batch matches. Both D1 (remove/pending) and D2 (install/pending) exist.
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
		_, d1Exists := rowByDecl[d1.DeclarationUUID]
		assert.False(t, d1Exists, "host %s: D1 remove row should have been cleaned up (same token as D2 install)", h.UUID)

		// D2 install row should exist as verified with resync=1
		d2Row, d2Exists := rowByDecl[d2.DeclarationUUID]
		if assert.True(t, d2Exists, "host %s: D2 install row should exist", h.UUID) {
			assert.Equal(t, "install", d2Row.OperationType, "host %s: D2 should be install", h.UUID)
			assert.Equal(t, "verified", d2Row.Status, "host %s: D2 should be marked verified by cleanup", h.UUID)
			assert.True(t, d2Row.Resync, "host %s: D2 should have resync=1", h.UUID)
		}
	}
}

func testCleanUpOrphanedPendingRemoves(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// This test simulates the "already stuck" scenario: a host has an orphaned
	// remove/pending row and a matching install/verified row with the same token
	// and identifier. No new declarations are changed, so the reconciler's
	// changedDeclarations is empty. The cleanUpOrphanedPendingRemoves safety net
	// in MDMAppleBatchSetHostDeclarationState should still clean it up.

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-orphan",
		UUID:            "test-host-uuid-orphan",
		HardwareSerial:  "ORPHAN-001",
		PrimaryIP:       "192.168.20.1",
		PrimaryMac:      "00:00:00:00:20:01",
		OsqueryHostID:   ptr.String("test-host-uuid-orphan"),
		NodeKey:         ptr.String("test-host-uuid-orphan"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	setupMDMDeviceAndEnrollment(t, ds, ctx, host.UUID, host.HardwareSerial)

	// Create a declaration so we have a valid token to work with.
	decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		DeclarationUUID: "decl-orphan-test",
		Name:            "Orphan Test Declaration",
		Identifier:      "com.example.orphan",
		RawJSON:         []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.orphan"}`),
	}, nil)
	require.NoError(t, err)

	// Get the token from mdm_apple_declarations
	var hexToken string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &hexToken,
			"SELECT HEX(token) FROM mdm_apple_declarations WHERE declaration_uuid = ?", decl.DeclarationUUID)
	})

	// Simulate the stuck state: insert both an install/verified row (new UUID)
	// and a remove/pending row (old UUID) with the same token and identifier.
	// Use UNHEX(?) directly to get the exact same binary token.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_apple_declarations
			(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier)
			VALUES (?, ?, 'verified', 'install', UNHEX(?), ?)`,
			host.UUID, decl.DeclarationUUID, hexToken, decl.Identifier)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_apple_declarations
			(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier)
			VALUES (?, ?, 'pending', 'remove', UNHEX(?), ?)`,
			host.UUID, "decl-orphan-old-uuid", hexToken, decl.Identifier)
		return err
	})

	// Verify both rows exist before the reconciler runs.
	var countBefore int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &countBefore, `
			SELECT COUNT(*) FROM host_mdm_apple_declarations WHERE host_uuid = ?`, host.UUID)
	})
	require.Equal(t, 2, countBefore)

	// Run the reconciler. There are no changed declarations (the install row
	// matches the desired state, and the remove row is excluded by the
	// reconciler's query filter). The safety net should still clean up.
	_, err = ds.MDMAppleBatchSetHostDeclarationState(ctx)
	require.NoError(t, err)

	// Assert: the orphaned remove/pending row should be deleted.
	type resultRow struct {
		DeclarationUUID string `db:"declaration_uuid"`
		OperationType   string `db:"operation_type"`
		Status          string `db:"status"`
	}
	var remaining []resultRow
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &remaining, `
			SELECT declaration_uuid, operation_type, status
			FROM host_mdm_apple_declarations
			WHERE host_uuid = ?
			ORDER BY declaration_uuid`, host.UUID)
	})

	require.Len(t, remaining, 1, "orphaned remove/pending row should have been cleaned up")
	assert.Equal(t, decl.DeclarationUUID, remaining[0].DeclarationUUID)
	assert.Equal(t, "install", remaining[0].OperationType)
	assert.Equal(t, "verified", remaining[0].Status)
}
