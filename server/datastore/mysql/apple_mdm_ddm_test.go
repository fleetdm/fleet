package mysql

import (
	"context"
	"slices"
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

// Helper function to check if a declaration is deleted
func checkDeclarationDeleted(t *testing.T, ds *Datastore, ctx context.Context, hostUUID, declarationUUID, operation string) {
	var count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		db := q.(*sqlx.DB)
		return db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM host_mdm_apple_declarations
			WHERE host_uuid = ? AND declaration_uuid = ? AND operation_type = ?`,
			hostUUID, declarationUUID, operation).Scan(&count)
	})
	assert.Zero(t, count)
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
			})
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

		// Insert 3 install declarations with NULL status
		for i := 0; i < 3; i++ {
			insertHostDeclaration(
				t, ds, ctx,
				host.UUID,
				declarations[i].DeclarationUUID,
				"ABC", // token update will trigger resend
				"verified",
				"install",
				declarations[i].Identifier,
			)
		}

		// Insert 3 remove declarations with NULL status
		for i := 0; i < 3; i++ {
			insertHostDeclaration(
				t, ds, ctx,
				host.UUID,
				removeDeclarations[i].DeclarationUUID,
				removeDeclarations[i].DeclarationUUID, // token
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
			})
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
			var token string
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				err := sqlx.GetContext(ctx, q, &token,
					"SELECT token as token FROM mdm_apple_declarations WHERE declaration_uuid = ?", declarationUUID)
				return err
			})
			return token
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
				// Last remove operation uses its declaration UUID as token
				removeTokens[i] = removeDeclarations[i].DeclarationUUID
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
				if slices.Contains(removeTokens, decl.DeclarationUUID) {
					checkDeclarationStatus(t, ds, ctx, host.UUID, decl.DeclarationUUID, "pending", "remove")
				} else {
					checkDeclarationDeleted(t, ds, ctx, host.UUID, decl.DeclarationUUID, "remove")
				}
			}

			// Check install declarations
			for _, decl := range installDeclarations {
				if slices.Contains(removeTokens, decl.Token) {
					checkDeclarationStatus(t, ds, ctx, host.UUID, decl.DeclarationUUID, "verified", "install")
				} else {
					checkDeclarationStatus(t, ds, ctx, host.UUID, decl.DeclarationUUID, "pending", "install")
				}
			}
		}
	})
}
