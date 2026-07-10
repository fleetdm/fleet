package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
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
		{"ListAppleDDMAssets", testListAppleDDMAssets},
		{"GetAppleDDMAsset", testGetAppleDDMAsset},
		{"GetAppleDDMAssetForDownload", testGetAppleDDMAssetForDownload},
		{"CreateAppleDDMAsset", testCreateAppleDDMAsset},
		{"DeleteAppleDDMAsset", testDeleteAppleDDMAsset},
		{"BatchSetAppleDDMAssets", testBatchSetAppleDDMAssets},
		{"StoreDDMStatusReportSkipsRemoveRows", testStoreDDMStatusReportSkipsRemoveRows},
		{"CleanUpDuplicateRemoveInstallAcrossBatches", testCleanUpDuplicateRemoveInstallAcrossBatches},
		{"ChannelScopeIsolation", testDDMChannelScopeIsolation},
		{"GetAppleDDMAssetForDelivery", testGetAppleDDMAssetForDelivery},
		{"GetAppleDDMAssetsReferencedByDeclarations", testGetAppleDDMAssetsReferencedByDeclarations},
		{"AssetsUpdatedAtRoundTripAndToken", testDDMAssetsUpdatedAtRoundTripAndToken},
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
	err = ds.MDMAppleStoreDDMStatusReport(ctx, host.UUID, fleet.PayloadScopeSystem, updates)
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

// testDDMChannelScopeIsolation verifies the four DDM serving queries keep the
// device (System) and user (User) channels independent: each channel sees only
// its own declarations, tokens are computed per channel, and a status report on
// one channel doesn't touch the other channel's rows.
func testDDMChannelScopeIsolation(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-ddm-scope",
		UUID:            "test-host-uuid-ddm-scope",
		HardwareSerial:  "ABC123-DDM-SCOPE",
		PrimaryIP:       "192.168.1.60",
		PrimaryMac:      "00:00:00:00:00:60",
		OsqueryHostID:   new("test-host-uuid-ddm-scope"),
		NodeKey:         new("test-host-uuid-ddm-scope"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	setupMDMDeviceAndEnrollment(t, ds, ctx, host.UUID, host.HardwareSerial)

	devDecl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Name:       "DeviceDecl",
		Identifier: "com.example.device",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.device"}`),
		Scope:      fleet.PayloadScopeSystem,
	}, nil, nil)
	require.NoError(t, err)
	userDecl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Name:       "UserDecl",
		Identifier: "com.example.user",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.user"}`),
		Scope:      fleet.PayloadScopeUser,
	}, nil, nil)
	require.NoError(t, err)

	readBinaryToken := func(declUUID string) []byte {
		var tok []byte
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &tok, "SELECT token FROM mdm_apple_declarations WHERE declaration_uuid = ?", declUUID)
		})
		return tok
	}
	devToken := readBinaryToken(devDecl.DeclarationUUID)
	userToken := readBinaryToken(userDecl.DeclarationUUID)

	pending := fleet.MDMDeliveryPending
	require.NoError(t, ds.BulkUpsertMDMAppleHostDeclarations(ctx, []*fleet.MDMAppleHostDeclaration{
		{
			HostUUID: host.UUID, DeclarationUUID: devDecl.DeclarationUUID, Name: devDecl.Name,
			Identifier: devDecl.Identifier, Status: &pending, OperationType: fleet.MDMOperationTypeInstall,
			Token: string(devToken), Scope: fleet.PayloadScopeSystem,
		},
		{
			HostUUID: host.UUID, DeclarationUUID: userDecl.DeclarationUUID, Name: userDecl.Name,
			Identifier: userDecl.Identifier, Status: &pending, OperationType: fleet.MDMOperationTypeInstall,
			Token: string(userToken), Scope: fleet.PayloadScopeUser,
		},
	}))

	// Tokens are computed per channel and differ.
	sysTok, err := ds.MDMAppleDDMDeclarationsToken(ctx, host.UUID, fleet.PayloadScopeSystem)
	require.NoError(t, err)
	usrTok, err := ds.MDMAppleDDMDeclarationsToken(ctx, host.UUID, fleet.PayloadScopeUser)
	require.NoError(t, err)
	require.NotEmpty(t, sysTok.DeclarationsToken)
	require.NotEmpty(t, usrTok.DeclarationsToken)
	require.NotEqual(t, sysTok.DeclarationsToken, usrTok.DeclarationsToken)

	// Declaration items are scoped to their channel.
	sysItems, err := ds.MDMAppleDDMDeclarationItems(ctx, host.UUID, fleet.PayloadScopeSystem)
	require.NoError(t, err)
	require.Len(t, sysItems, 1)
	require.Equal(t, "com.example.device", sysItems[0].Identifier)

	usrItems, err := ds.MDMAppleDDMDeclarationItems(ctx, host.UUID, fleet.PayloadScopeUser)
	require.NoError(t, err)
	require.Len(t, usrItems, 1)
	require.Equal(t, "com.example.user", usrItems[0].Identifier)

	// The declaration response respects scope: the user declaration is not served
	// on the device channel and vice versa.
	_, err = ds.MDMAppleDDMDeclarationsResponse(ctx, "com.example.user", host.UUID, fleet.PayloadScopeSystem)
	require.True(t, fleet.IsNotFound(err))
	gotUser, err := ds.MDMAppleDDMDeclarationsResponse(ctx, "com.example.user", host.UUID, fleet.PayloadScopeUser)
	require.NoError(t, err)
	require.Equal(t, userDecl.DeclarationUUID, gotUser.DeclarationUUID)

	_, err = ds.MDMAppleDDMDeclarationsResponse(ctx, "com.example.device", host.UUID, fleet.PayloadScopeUser)
	require.True(t, fleet.IsNotFound(err))

	// A status report on the user channel only transitions user-scoped rows.
	verified := fleet.MDMDeliveryVerified
	err = ds.MDMAppleStoreDDMStatusReport(ctx, host.UUID, fleet.PayloadScopeUser, []*fleet.MDMAppleHostDeclaration{
		{Token: fmt.Sprintf("%X", userToken), Status: &verified, OperationType: fleet.MDMOperationTypeInstall},
	})
	require.NoError(t, err)

	type statusRow struct {
		DeclarationUUID string `db:"declaration_uuid"`
		Status          string `db:"status"`
	}
	var rows []statusRow
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &rows, `
			SELECT declaration_uuid, COALESCE(status,'') AS status
			FROM host_mdm_apple_declarations WHERE host_uuid = ? ORDER BY declaration_uuid`, host.UUID)
	})
	statusByUUID := make(map[string]string, len(rows))
	for _, r := range rows {
		statusByUUID[r.DeclarationUUID] = r.Status
	}
	require.Equal(t, "verified", statusByUUID[userDecl.DeclarationUUID], "user-scoped row should be verified by the user-channel report")
	require.Equal(t, "pending", statusByUUID[devDecl.DeclarationUUID], "device-scoped row must be untouched by a user-channel report")
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
	}, nil, nil)
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

func testListAppleDDMAssets(t *testing.T, ds *Datastore) {
	t.Run("no assets returns empty list", func(t *testing.T) {
		ctx := t.Context()
		assets, err := ds.ListAppleDDMAssets(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, assets)
	})

	t.Run("returns assets for requested team", func(t *testing.T) {
		ctx := t.Context()

		// insert helper
		_, err := ds.CreateAppleDDMAsset(ctx, "asset-1", "asset.identifier", []byte(`{"foo":"bar"}`), new(uint(1)))
		require.NoError(t, err)

		assets, err := ds.ListAppleDDMAssets(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, assets)

		assets, err = ds.ListAppleDDMAssets(ctx, new(uint(1)))
		require.NoError(t, err)
		require.Len(t, assets, 1)
	})
}

func testGetAppleDDMAsset(t *testing.T, ds *Datastore) {
	t.Run("returns not found for missing asset", func(t *testing.T) {
		ctx := t.Context()
		asset, err := ds.GetAppleDDMAsset(ctx, "fake-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.Nil(t, asset)
	})

	t.Run("returns asset for existing asset", func(t *testing.T) {
		ctx := t.Context()
		assetUUID, err := ds.CreateAppleDDMAsset(ctx, "asset-1", "asset.identifier", []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		asset, err := ds.GetAppleDDMAsset(ctx, assetUUID)
		require.NoError(t, err)
		require.NotNil(t, asset)
	})

	t.Run("return error for empty asset uuid", func(t *testing.T) {
		ctx := t.Context()
		asset, err := ds.GetAppleDDMAsset(ctx, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "asset UUID is required")
		require.Nil(t, asset)
	})
}

func testGetAppleDDMAssetForDownload(t *testing.T, ds *Datastore) {
	t.Run("returns not found for missing asset", func(t *testing.T) {
		ctx := t.Context()
		asset, err := ds.GetAppleDDMAssetForDownload(ctx, "fake-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.Nil(t, asset)
	})

	t.Run("returns asset values for existing asset", func(t *testing.T) {
		ctx := t.Context()
		assetName := "asset-1"
		assetUUID, err := ds.CreateAppleDDMAsset(ctx, assetName, "asset.identifier", []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		asset, err := ds.GetAppleDDMAssetForDownload(ctx, assetUUID)
		require.NoError(t, err)
		require.NotNil(t, asset)
		require.Equal(t, assetName, asset.Name)
		require.NotNil(t, asset.Data)
	})

	t.Run("return error for empty asset uuid", func(t *testing.T) {
		ctx := t.Context()
		asset, err := ds.GetAppleDDMAssetForDownload(ctx, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "asset UUID is required")
		require.Nil(t, asset)
	})
}

func testDeleteAppleDDMAsset(t *testing.T, ds *Datastore) {
	t.Run("returns not found for missing asset", func(t *testing.T) {
		ctx := t.Context()
		err := ds.DeleteAppleDDMAsset(ctx, "fake-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("returns error for empty asset UUID", func(t *testing.T) {
		ctx := t.Context()
		err := ds.DeleteAppleDDMAsset(ctx, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "asset UUID is required")
	})

	t.Run("deletes existing asset", func(t *testing.T) {
		ctx := t.Context()
		assetUUID, err := ds.CreateAppleDDMAsset(ctx, "asset-1", "asset.identifier", []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		err = ds.DeleteAppleDDMAsset(ctx, assetUUID)
		require.NoError(t, err)
	})

	t.Run("returns foreign key error for asset with declaration association", func(t *testing.T) {
		ctx := t.Context()
		assetUUID, err := ds.CreateAppleDDMAsset(ctx, "asset-1", "asset.identifier", []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		// Insert a declaration, and decl<->asset association.
		declUUID := uuid.NewString()
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			DeclarationUUID: declUUID,
			Identifier:      "declaration.identifier",
			Name:            "decl-name",
			RawJSON:         []byte(`{"foo":"bar"}`),
		}, nil, nil)
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err = q.ExecContext(ctx, `INSERT INTO mdm_apple_declaration_asset_references (declaration_uuid, asset_uuid) VALUES (?, ?)`, decl.DeclarationUUID, assetUUID)
			require.NoError(t, err)
			return nil
		})

		err = ds.DeleteAppleDDMAsset(ctx, assetUUID)
		require.Error(t, err)
		var foreignKeyErr *foreignKeyError
		require.ErrorAs(t, err, &foreignKeyErr)
	})
}

func testBatchSetAppleDDMAssets(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	assetData := func(typ, identifier, dataURL string) []byte {
		return []byte(fmt.Sprintf(`{"Type":%q,"Identifier":%q,"Payload":{"Reference":{"DataURL":%q}}}`, typ, identifier, dataURL))
	}
	uploadedAt := func(identifier string) time.Time {
		var ts time.Time
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &ts,
				`SELECT uploaded_at FROM mdm_apple_declaration_assets WHERE team_id = 0 AND identifier = ?`, identifier)
		})
		return ts
	}

	// Create two new assets.
	changes, err := ds.BatchSetAppleDDMAssets(ctx, nil, []*fleet.MDMAppleDDMAssetToSet{
		{Name: "a", Identifier: "id.a", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.a", "https://example.com/a")},
		{Name: "b", Identifier: "id.b", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.b", "https://example.com/b")},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"a", "b"}, changes.Created)
	require.Empty(t, changes.Edited)
	require.Empty(t, changes.Deleted)
	assets, err := ds.ListAppleDDMAssets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, assets, 2)
	firstUploaded := uploadedAt("id.a")

	// Re-applying the same content is a no-op: uploaded_at must not change and
	// no changes are reported.
	changes, err = ds.BatchSetAppleDDMAssets(ctx, nil, []*fleet.MDMAppleDDMAssetToSet{
		{Name: "a", Identifier: "id.a", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.a", "https://example.com/a")},
		{Name: "b", Identifier: "id.b", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.b", "https://example.com/b")},
	})
	require.NoError(t, err)
	require.Empty(t, changes.Created)
	require.Empty(t, changes.Edited)
	require.Empty(t, changes.Deleted)
	require.True(t, uploadedAt("id.a").Equal(firstUploaded))

	// Editing an asset's payload bumps uploaded_at and is reported as edited.
	changes, err = ds.BatchSetAppleDDMAssets(ctx, nil, []*fleet.MDMAppleDDMAssetToSet{
		{Name: "a", Identifier: "id.a", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.a", "https://example.com/a-edited")},
		{Name: "b", Identifier: "id.b", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.b", "https://example.com/b")},
	})
	require.NoError(t, err)
	require.Empty(t, changes.Created)
	require.ElementsMatch(t, []string{"a"}, changes.Edited)
	require.Empty(t, changes.Deleted)
	require.True(t, uploadedAt("id.a").After(firstUploaded))

	// Omitting an asset deletes it and is reported as deleted.
	changes, err = ds.BatchSetAppleDDMAssets(ctx, nil, []*fleet.MDMAppleDDMAssetToSet{
		{Name: "a", Identifier: "id.a", Type: "com.apple.asset.data", Data: assetData("com.apple.asset.data", "id.a", "https://example.com/a-edited")},
	})
	require.NoError(t, err)
	require.Empty(t, changes.Created)
	require.Empty(t, changes.Edited)
	require.ElementsMatch(t, []string{"b"}, changes.Deleted)
	assets, err = ds.ListAppleDDMAssets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, assets, 1)

	// Changing an existing asset's type (same identifier) is rejected.
	_, err = ds.BatchSetAppleDDMAssets(ctx, nil, []*fleet.MDMAppleDDMAssetToSet{
		{Name: "a", Identifier: "id.a", Type: "com.apple.asset.other", Data: assetData("com.apple.asset.other", "id.a", "https://example.com/a-edited")},
	})
	require.Error(t, err)
	var conflictErr *fleet.ConflictError
	require.ErrorAs(t, err, &conflictErr)

	// Deleting an asset that is still referenced by a declaration is rejected.
	decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl.identifier",
		Name:       "decl-name",
		RawJSON:    []byte(`{"foo":"bar"}`),
	}, nil, nil)
	require.NoError(t, err)
	referencedAsset, err := ds.GetAppleDDMAsset(ctx, assets[0].AssetUUID)
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO mdm_apple_declaration_asset_references (declaration_uuid, asset_uuid) VALUES (?, ?)`,
			decl.DeclarationUUID, referencedAsset.AssetUUID)
		return err
	})
	_, err = ds.BatchSetAppleDDMAssets(ctx, nil, nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &conflictErr)
	require.Contains(t, err.Error(), referencedAsset.Identifier)
}

func testCreateAppleDDMAsset(t *testing.T, ds *Datastore) {
	t.Run("creates asset with valid data", func(t *testing.T) {
		ctx := t.Context()
		assetUUID, err := ds.CreateAppleDDMAsset(ctx, "valid-asset", "valid-asset-identifier", []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)
		require.NotEmpty(t, assetUUID)
	})

	t.Run("fails to create asset with empty name", func(t *testing.T) {
		ctx := t.Context()
		_, err := ds.CreateAppleDDMAsset(ctx, "", "asset.identifier", []byte(`{"foo":"bar"}`), nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("fails to create asset with empty identifier", func(t *testing.T) {
		ctx := t.Context()
		_, err := ds.CreateAppleDDMAsset(ctx, "asset-1", "", []byte(`{"foo":"bar"}`), nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "asset identifier is required")
	})

	t.Run("fails to create asset with empty data", func(t *testing.T) {
		ctx := t.Context()
		_, err := ds.CreateAppleDDMAsset(ctx, "asset-1", "asset.identifier", nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "asset data is required")
	})

	t.Run("returns already exists error when creating asset with duplicate identifier", func(t *testing.T) {
		ctx := t.Context()
		assetIdentifier := "conflict.asset.identifier"
		_, err := ds.CreateAppleDDMAsset(ctx, "conflict-asset-1", assetIdentifier, []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		_, err = ds.CreateAppleDDMAsset(ctx, "asset-2", assetIdentifier, []byte(`{"foo":"baz"}`), nil)
		require.Error(t, err)
		var alreadyExistsErr *existsError
		require.ErrorAs(t, err, &alreadyExistsErr)
	})

	t.Run("returns already exists error when creating asset with duplicate name", func(t *testing.T) {
		ctx := t.Context()

		assetName := "conflict-asset-name"
		_, err := ds.CreateAppleDDMAsset(ctx, assetName, "conflict.asset.identifier-one", []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		_, err = ds.CreateAppleDDMAsset(ctx, assetName, "conflict.asset.identifier-2", []byte(`{"foo":"baz"}`), nil)
		require.Error(t, err)
		var alreadyExistsErr *existsError
		require.ErrorAs(t, err, &alreadyExistsErr)
	})

	t.Run("does not conflict across teams", func(t *testing.T) {
		ctx := t.Context()
		assetIdentifier := "no-conflict.asset.identifier"
		assetName := "no-conflict.asset-1"
		_, err := ds.CreateAppleDDMAsset(ctx, assetName, assetIdentifier, []byte(`{"foo":"bar"}`), nil)
		require.NoError(t, err)

		_, err = ds.CreateAppleDDMAsset(ctx, assetName, assetIdentifier, []byte(`{"foo":"baz"}`), new(uint(1)))
		require.NoError(t, err)
	})
}

func testGetAppleDDMAssetForDelivery(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	const identifier = "com.example.shared.asset"

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "ddm-delivery-team"})
	require.NoError(t, err)

	// Same identifier in two different teams (global vs a real team). The unique
	// key is (team_id, identifier), so both assets can coexist.
	globalAssetUUID, err := ds.CreateAppleDDMAsset(ctx, "global-asset", identifier, []byte(`{"global":true}`), nil)
	require.NoError(t, err)
	teamAssetUUID, err := ds.CreateAppleDDMAsset(ctx, "team-asset", identifier, []byte(`{"team":true}`), &team.ID)
	require.NoError(t, err)
	require.NotEqual(t, globalAssetUUID, teamAssetUUID)

	// A declaration in each team that references its team's asset. Delivery is
	// now scoped through the host's installed declarations, so the asset is only
	// reachable via a declaration that references it.
	globalDecl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Name:       "GlobalDecl",
		Identifier: "com.example.globalDecl",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.globalDecl"}`),
	}, nil, []string{globalAssetUUID})
	require.NoError(t, err)
	teamDecl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		TeamID:     &team.ID,
		Name:       "TeamDecl",
		Identifier: "com.example.teamDecl",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.teamDecl"}`),
	}, nil, []string{teamAssetUUID})
	require.NoError(t, err)

	// A global host that has the global declaration installed, and a team host
	// that has the team declaration installed.
	globalHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "ddm-delivery-global",
		UUID:            "ddm-delivery-global-uuid",
		HardwareSerial:  "DDM-DELIVERY-GLOBAL",
		PrimaryIP:       "192.168.1.80",
		PrimaryMac:      "00:00:00:00:00:80",
		OsqueryHostID:   new("ddm-delivery-global-uuid"),
		NodeKey:         new("ddm-delivery-global-uuid"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	teamHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "ddm-delivery-team",
		UUID:            "ddm-delivery-team-uuid",
		HardwareSerial:  "DDM-DELIVERY-TEAM",
		PrimaryIP:       "192.168.1.81",
		PrimaryMac:      "00:00:00:00:00:81",
		OsqueryHostID:   new("ddm-delivery-team-uuid"),
		NodeKey:         new("ddm-delivery-team-uuid"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID})))

	insertHostDeclaration(t, ds, ctx, globalHost.UUID, globalDecl.DeclarationUUID, "global-token", "verified", "install", globalDecl.Identifier)
	insertHostDeclaration(t, ds, ctx, teamHost.UUID, teamDecl.DeclarationUUID, "team-token", "verified", "install", teamDecl.Identifier)

	t.Run("returns the asset scoped to the host's team", func(t *testing.T) {
		got, err := ds.GetAppleDDMAssetForDelivery(ctx, identifier, globalHost.UUID)
		require.NoError(t, err)
		require.Equal(t, globalAssetUUID, got.AssetUUID)
		require.JSONEq(t, `{"global":true}`, string(got.Data))

		got, err = ds.GetAppleDDMAssetForDelivery(ctx, identifier, teamHost.UUID)
		require.NoError(t, err)
		require.Equal(t, teamAssetUUID, got.AssetUUID)
		require.JSONEq(t, `{"team":true}`, string(got.Data))
	})

	t.Run("does not return an asset the host does not reference", func(t *testing.T) {
		// A host with no declaration referencing the asset gets nothing, even
		// though an asset with this identifier exists in its team.
		bareHost, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        "ddm-delivery-bare",
			UUID:            "ddm-delivery-bare-uuid",
			HardwareSerial:  "DDM-DELIVERY-BARE",
			PrimaryIP:       "192.168.1.82",
			PrimaryMac:      "00:00:00:00:00:82",
			OsqueryHostID:   new("ddm-delivery-bare-uuid"),
			NodeKey:         new("ddm-delivery-bare-uuid"),
			DetailUpdatedAt: time.Now(),
			Platform:        "darwin",
		})
		require.NoError(t, err)

		_, err = ds.GetAppleDDMAssetForDelivery(ctx, identifier, bareHost.UUID)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("does not leak an asset from another team", func(t *testing.T) {
		// A team host that (incorrectly) references the global declaration must
		// not receive the global asset, because the asset's team_id does not
		// match the host's team_id.
		mismatchHost, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        "ddm-delivery-mismatch",
			UUID:            "ddm-delivery-mismatch-uuid",
			HardwareSerial:  "DDM-DELIVERY-MISMATCH",
			PrimaryIP:       "192.168.1.83",
			PrimaryMac:      "00:00:00:00:00:83",
			OsqueryHostID:   new("ddm-delivery-mismatch-uuid"),
			NodeKey:         new("ddm-delivery-mismatch-uuid"),
			DetailUpdatedAt: time.Now(),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{mismatchHost.ID})))
		insertHostDeclaration(t, ds, ctx, mismatchHost.UUID, globalDecl.DeclarationUUID, "mismatch-token", "verified", "install", globalDecl.Identifier)

		_, err = ds.GetAppleDDMAssetForDelivery(ctx, identifier, mismatchHost.UUID)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("unknown host is not found", func(t *testing.T) {
		_, err := ds.GetAppleDDMAssetForDelivery(ctx, identifier, "does-not-exist-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("missing identifier is an error", func(t *testing.T) {
		_, err := ds.GetAppleDDMAssetForDelivery(ctx, "", globalHost.UUID)
		require.Error(t, err)
	})
}

func testGetAppleDDMAssetsReferencedByDeclarations(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	asset1UUID, err := ds.CreateAppleDDMAsset(ctx, "asset-one", "com.example.asset.one", []byte(`{"a":1}`), nil)
	require.NoError(t, err)
	asset2UUID, err := ds.CreateAppleDDMAsset(ctx, "asset-two", "com.example.asset.two", []byte(`{"a":2}`), nil)
	require.NoError(t, err)
	// Unreferenced asset — must not be returned.
	_, err = ds.CreateAppleDDMAsset(ctx, "asset-three", "com.example.asset.three", []byte(`{"a":3}`), nil)
	require.NoError(t, err)

	// declA references both assets, declB references only asset1.
	declA, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Name:       "DeclA",
		Identifier: "com.example.declA",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.declA"}`),
	}, nil, []string{asset1UUID, asset2UUID})
	require.NoError(t, err)
	declB, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Name:       "DeclB",
		Identifier: "com.example.declB",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.declB"}`),
	}, nil, []string{asset1UUID})
	require.NoError(t, err)

	t.Run("empty input returns empty slice", func(t *testing.T) {
		got, err := ds.GetAppleDDMAssetsReferencedByDeclarations(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("returns assets referenced by the given declarations, deduped", func(t *testing.T) {
		got, err := ds.GetAppleDDMAssetsReferencedByDeclarations(ctx, []string{declA.DeclarationUUID, declB.DeclarationUUID})
		require.NoError(t, err)
		gotUUIDs := make([]string, 0, len(got))
		for _, a := range got {
			gotUUIDs = append(gotUUIDs, a.AssetUUID)
		}
		// asset1 is referenced by both declarations but must appear once (DISTINCT).
		require.ElementsMatch(t, []string{asset1UUID, asset2UUID}, gotUUIDs)
	})

	t.Run("single declaration returns only its references", func(t *testing.T) {
		got, err := ds.GetAppleDDMAssetsReferencedByDeclarations(ctx, []string{declB.DeclarationUUID})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, asset1UUID, got[0].AssetUUID)
	})
}

func testDDMAssetsUpdatedAtRoundTripAndToken(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-assets-token",
		UUID:            "test-host-uuid-assets-token",
		HardwareSerial:  "ASSETS-TOKEN-1",
		PrimaryIP:       "192.168.1.70",
		PrimaryMac:      "00:00:00:00:00:70",
		OsqueryHostID:   new("test-host-uuid-assets-token"),
		NodeKey:         new("test-host-uuid-assets-token"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	setupMDMDeviceAndEnrollment(t, ds, ctx, host.UUID, host.HardwareSerial)

	decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Name:       "AssetTokenDecl",
		Identifier: "com.example.assettoken",
		RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.assettoken"}`),
		Scope:      fleet.PayloadScopeSystem,
	}, nil, nil)
	require.NoError(t, err)

	var token []byte
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &token, "SELECT token FROM mdm_apple_declarations WHERE declaration_uuid = ?", decl.DeclarationUUID)
	})

	pending := fleet.MDMDeliveryPending

	// First: install the declaration WITHOUT any assets_updated_at.
	require.NoError(t, ds.BulkUpsertMDMAppleHostDeclarations(ctx, []*fleet.MDMAppleHostDeclaration{
		{
			HostUUID: host.UUID, DeclarationUUID: decl.DeclarationUUID, Name: decl.Name,
			Identifier: decl.Identifier, Status: &pending, OperationType: fleet.MDMOperationTypeInstall,
			Token: string(token), Scope: fleet.PayloadScopeSystem,
		},
	}))

	tokBefore, err := ds.MDMAppleDDMDeclarationsToken(ctx, host.UUID, fleet.PayloadScopeSystem)
	require.NoError(t, err)
	require.NotEmpty(t, tokBefore.DeclarationsToken)

	// Now simulate an asset-only update: same declaration/token, but with
	// assets_updated_at stamped (as the reconciler would do).
	assetsUpdatedAt := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, ds.BulkUpsertMDMAppleHostDeclarations(ctx, []*fleet.MDMAppleHostDeclaration{
		{
			HostUUID: host.UUID, DeclarationUUID: decl.DeclarationUUID, Name: decl.Name,
			Identifier: decl.Identifier, Status: &pending, OperationType: fleet.MDMOperationTypeInstall,
			Token: string(token), Scope: fleet.PayloadScopeSystem, AssetsUpdatedAt: &assetsUpdatedAt,
		},
	}))

	// assets_updated_at is persisted.
	var stored *time.Time
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &stored,
			"SELECT assets_updated_at FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_uuid = ?",
			host.UUID, decl.DeclarationUUID)
	})
	require.NotNil(t, stored)
	require.True(t, assetsUpdatedAt.Equal(*stored), "expected %s, got %s", assetsUpdatedAt, *stored)

	// The SQL-computed declarations token must change once assets_updated_at is set,
	// even though the declaration's static token is unchanged. This is what causes
	// the host to re-sync on an asset-only update.
	tokAfter, err := ds.MDMAppleDDMDeclarationsToken(ctx, host.UUID, fleet.PayloadScopeSystem)
	require.NoError(t, err)
	require.NotEqual(t, tokBefore.DeclarationsToken, tokAfter.DeclarationsToken)

	// And it must match the Go-side EffectiveDDMToken over the same inputs, proving
	// the token endpoint and the declaration-items endpoint agree.
	items, err := ds.MDMAppleDDMDeclarationItems(ctx, host.UUID, fleet.PayloadScopeSystem)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].AssetsUpdatedAt)
	require.True(t, assetsUpdatedAt.Equal(*items[0].AssetsUpdatedAt))
}
