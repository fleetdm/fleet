package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// Custom host vital values are arbitrary admin/external strings, so one may
// contain a literal $FLEET_VAR_<name>. variables.Replace is a blind global
// string replace, so vitals must be expanded after the Fleet-var pass — else a
// $FLEET_VAR_ token embedded in a vital value would be rewritten by that pass.
func TestReplaceDeclarationFleetVariablesExpandsVitalsLast(t *testing.T) {
	ctx := t.Context()
	ds := mysqltest.CreateMySQLDS(t)
	svc := MDMAppleDDMService{
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	host, err := ds.NewHost(ctx, &fleet.Host{
		UUID:            "vital-order-uuid",
		Hostname:        "vital-order-host",
		HardwareSerial:  "SERIAL123",
		OsqueryHostID:   new("vital-order"),
		NodeKey:         new("vital-order"),
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	vital, err := ds.CreateCustomHostVital(ctx, "asset_tag")
	require.NoError(t, err)
	// The vital's value deliberately embeds a literal $FLEET_VAR_ token.
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, host.ID, vital.ID, "tag-$FLEET_VAR_HOST_HARDWARE_SERIAL"))

	contents := fmt.Sprintf(`{"vital":"$FLEET_HOST_VITAL_%d","serial":"$FLEET_VAR_HOST_HARDWARE_SERIAL"}`, vital.ID)
	out, err := svc.replaceDeclarationFleetVariables(ctx, contents, host.UUID)
	require.NoError(t, err)

	// The genuine $FLEET_VAR_HOST_HARDWARE_SERIAL reference expands to the serial,
	// but the identical token inside the vital's value survives intact because
	// vitals are expanded last (variables.Replace never sees it).
	require.JSONEq(t, `{"vital":"tag-$FLEET_VAR_HOST_HARDWARE_SERIAL","serial":"SERIAL123"}`, out)
}

func TestDeclarativeManagement_DeclarationItems(t *testing.T) {
	ctx := t.Context()
	ds := mysqltest.CreateMySQLDS(t)
	ddmService := MDMAppleDDMService{
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	// Helper function to create a host
	createHost := func(t *testing.T, hostUUID, hardwareSerial string) {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			UUID:            hostUUID,
			Hostname:        "test-host-" + hostUUID,
			HardwareSerial:  hardwareSerial,
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "00:00:00:00:00:00",
			OsqueryHostID:   ptr.String(hostUUID),
			NodeKey:         ptr.String(hostUUID),
			DetailUpdatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	// Helper function to create a declaration
	createDeclaration := func(t *testing.T, uuid, name, identifier string) *fleet.MDMAppleDeclaration {
		declaration := &fleet.MDMAppleDeclaration{
			DeclarationUUID: uuid,
			Name:            name,
			Identifier:      identifier,
			TeamID:          nil,
			RawJSON:         []byte(fmt.Sprintf(`{"Type":"com.apple.test.declaration","Identifier":"%s"}`, identifier)),
		}
		declaration, err := ds.NewMDMAppleDeclaration(context.Background(), declaration, nil)
		require.NoError(t, err)
		return declaration
	}

	// Helper function to set up device and enrollment records
	setupDeviceAndEnrollment := func(t *testing.T, hostUUID, hardwareSerial string) {
		// Insert the device record first (required for foreign key constraints)
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate) VALUES (?, ?, ?)`,
				hostUUID, hardwareSerial, "test")
			return err
		})

		// Insert a record into nano_enrollments table (required for foreign key constraints)
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				hostUUID, hostUUID, "Device", "topic", "push_magic", "token_hex", 1, time.Now())
			return err
		})
	}

	// Helper function to insert a host declaration
	insertHostDeclaration := func(t *testing.T, hostUUID, declarationUUID, status, operationType, identifier string) string {
		var token string
		var statusPtr *string
		if status != "" {
			statusPtr = ptr.String(status)
		}
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			// First, get the right token of the declaration
			err := sqlx.GetContext(ctx, q, &token,
				"SELECT HEX(token) as token FROM mdm_apple_declarations WHERE declaration_uuid = ?", declarationUUID)
			require.NoError(t, err)
			_, err = q.ExecContext(ctx, `
				INSERT INTO host_mdm_apple_declarations 
				(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier) 
				VALUES (?, ?, ?, ?, UNHEX(?), ?)`,
				hostUUID, declarationUUID, statusPtr, operationType, token, identifier)
			return err
		})
		return token
	}

	// Helper function to call DeclarativeManagement and verify response
	callDeclarativeManagementAndVerify := func(t *testing.T, hostUUID string,
		expectedConfigurations, expectedActivations int,
	) fleet.MDMAppleDDMDeclarationItemsResponse {
		req := mdm.Request{
			Context: ctx,
			EnrollID: &mdm.EnrollID{
				ID: hostUUID,
			},
		}

		dm := mdm.DeclarativeManagement{}
		dm.UDID = hostUUID
		dm.Endpoint = "declaration-items"

		response, err := ddmService.DeclarativeManagement(&req, &dm)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Parse the response
		var declarationItemsResponse fleet.MDMAppleDDMDeclarationItemsResponse
		err = json.Unmarshal(response, &declarationItemsResponse)
		require.NoError(t, err)

		// Verify the declarations in the response
		require.Len(t, declarationItemsResponse.Declarations.Configurations, expectedConfigurations)
		require.Len(t, declarationItemsResponse.Declarations.Activations, expectedActivations)

		return declarationItemsResponse
	}

	// Helper function to check if a declaration has status "pending"
	checkDeclarationStatus := func(t *testing.T, hostUUID, declarationUUID, expectedStatus string) {
		var status string
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			db := q.(*sqlx.DB)
			return db.QueryRowContext(ctx, `
				SELECT status FROM host_mdm_apple_declarations 
				WHERE host_uuid = ? AND declaration_uuid = ?`,
				hostUUID, declarationUUID).Scan(&status)
		})
		require.Equal(t, expectedStatus, status)
	}

	// Helper function to set the uploaded_at timestamp for a host declaration
	setDeclarationUploadedAt := func(t *testing.T, declarationUUID string, timestamp time.Time) {
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				UPDATE mdm_apple_declarations 
				SET uploaded_at = ?
				WHERE declaration_uuid = ?`,
				timestamp, declarationUUID)
			return err
		})
	}

	t.Run("SingleDeclaration", func(t *testing.T) {
		hostUUID := "test-host-uuid-1"
		hardwareSerial := "ABC123-1"

		// Create a test host
		createHost(t, hostUUID, hardwareSerial)

		// Create a test declaration
		declaration := createDeclaration(t, "test-declaration-uuid-1", "Test Declaration 1", "com.example.test.declaration.1")

		// Set up device and enrollment records
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		// Insert a host declaration
		token := insertHostDeclaration(t, hostUUID, declaration.DeclarationUUID, "pending", "install", declaration.Identifier)

		// Get the expected declarations token from the DB.
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)

		// Verify the declarations in the response
		require.Equal(t, declaration.Identifier, response.Declarations.Configurations[0].Identifier)
		require.Equal(t, token, response.Declarations.Configurations[0].ServerToken)

		// Verify the activations in the response
		require.Equal(t, declaration.DeclarationUUID+".activation", response.Declarations.Activations[0].Identifier)
		require.Equal(t, token, response.Declarations.Activations[0].ServerToken)
	})

	t.Run("ActivationUpdatedAtFoldsIntoToken", func(t *testing.T) {
		hostUUID := "test-host-uuid-act"
		hardwareSerial := "ABC123-ACT"

		createHost(t, hostUUID, hardwareSerial)
		declaration := createDeclaration(t, "test-declaration-uuid-act", "Test Declaration Act", "com.example.test.declaration.act")
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)
		insertHostDeclaration(t, hostUUID, declaration.DeclarationUUID, "pending", "install", declaration.Identifier)

		tokenBefore, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)
		respBefore := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)
		require.Equal(t, tokenBefore.DeclarationsToken, respBefore.DeclarationsToken)

		// Stamping activation_updated_at must move the token, and the SQL and Go
		// computations must still agree. They are written independently, so a
		// mismatch would re-sync every host on every check-in.
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE host_mdm_apple_declarations SET activation_updated_at = NOW(6) WHERE host_uuid = ?`, hostUUID)
			return err
		})

		tokenAfter, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)
		respAfter := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)

		require.Equal(t, tokenAfter.DeclarationsToken, respAfter.DeclarationsToken, "SQL and Go tokens must agree")
		require.NotEqual(t, tokenBefore.DeclarationsToken, tokenAfter.DeclarationsToken, "activation change must move the token")
	})

	t.Run("NoDeclarations", func(t *testing.T) {
		hostUUID := "test-host-uuid-2"
		hardwareSerial := "ABC123-2"

		// Create a test host
		createHost(t, hostUUID, hardwareSerial)

		// Set up device and enrollment records
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 0, 0)

		// Get the expected declarations token from the DB.
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)
	})

	t.Run("MultipleDeclarations", func(t *testing.T) {
		hostUUID := "test-host-uuid-3"
		hardwareSerial := "ABC123-3"

		// Create a test host
		createHost(t, hostUUID, hardwareSerial)

		// Create test declarations
		declaration1 := createDeclaration(t, "test-declaration-uuid-3-1", "Test Declaration 3-1", "com.example.test.declaration.3.1")
		declaration2 := createDeclaration(t, "test-declaration-uuid-3-2", "Test Declaration 3-2", "com.example.test.declaration.3.2")
		declaration3 := createDeclaration(t, "test-declaration-uuid-3-3", "Test Declaration 3-3", "com.example.test.declaration.3.3")

		// Set up device and enrollment records
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		// Insert host declarations
		insertHostDeclaration(t, hostUUID, declaration1.DeclarationUUID, "pending", "install", declaration1.Identifier)
		insertHostDeclaration(t, hostUUID, declaration2.DeclarationUUID, "pending", "install", declaration2.Identifier)
		insertHostDeclaration(t, hostUUID, declaration3.DeclarationUUID, "pending", "remove", declaration3.Identifier)

		// Get the expected declarations token from the DB.
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 2, 2)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)

		// Verify the declarations in the response (only install operations)
		identifiers := []string{
			response.Declarations.Configurations[0].Identifier,
			response.Declarations.Configurations[1].Identifier,
		}
		require.Contains(t, identifiers, declaration1.Identifier)
		require.Contains(t, identifiers, declaration2.Identifier)
		require.NotContains(t, identifiers, declaration3.Identifier)

		// Verify the activations in the response
		activationIdentifiers := []string{
			response.Declarations.Activations[0].Identifier,
			response.Declarations.Activations[1].Identifier,
		}
		require.Contains(t, activationIdentifiers, declaration1.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration2.DeclarationUUID+".activation")
		require.NotContains(t, activationIdentifiers, declaration3.DeclarationUUID+".activation")
	})

	t.Run("RemoveDeclarationsWithNullStatus", func(t *testing.T) {
		hostUUID := "test-host-uuid-4"
		hardwareSerial := "ABC123-4"

		// Create a test host
		createHost(t, hostUUID, hardwareSerial)

		// Create test declarations
		declaration1 := createDeclaration(t, "test-declaration-uuid-4-1", "Test Declaration 4-1", "com.example.test.declaration.4.1")
		declaration2 := createDeclaration(t, "test-declaration-uuid-4-2", "Test Declaration 4-2", "com.example.test.declaration.4.2")
		declaration3 := createDeclaration(t, "test-declaration-uuid-4-3", "Test Declaration 4-3", "com.example.test.declaration.4.3")

		// Set up device and enrollment records
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		// Insert host declarations
		token1 := insertHostDeclaration(t, hostUUID, declaration1.DeclarationUUID, "pending", "install", declaration1.Identifier)
		// Use empty string for NULL status
		insertHostDeclaration(t, hostUUID, declaration2.DeclarationUUID, "", "remove", declaration2.Identifier)
		insertHostDeclaration(t, hostUUID, declaration3.DeclarationUUID, "", "remove", declaration3.Identifier)

		// Get the expected declarations token from the DB.
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)

		// Verify the declarations in the response (only install operations)
		require.Equal(t, declaration1.Identifier, response.Declarations.Configurations[0].Identifier)
		require.Equal(t, token1, response.Declarations.Configurations[0].ServerToken)

		// Verify the activations in the response
		require.Equal(t, declaration1.DeclarationUUID+".activation", response.Declarations.Activations[0].Identifier)
		require.Equal(t, token1, response.Declarations.Activations[0].ServerToken)

		// Check that the remove declarations with NULL status were updated to "pending"
		checkDeclarationStatus(t, hostUUID, declaration2.DeclarationUUID, "pending")
		checkDeclarationStatus(t, hostUUID, declaration3.DeclarationUUID, "pending")
	})

	t.Run("DeclarationsWithSameUploadedAt", func(t *testing.T) {
		hostUUID := "test-host-uuid-5"
		hardwareSerial := "ABC123-5"

		// Create a test host
		createHost(t, hostUUID, hardwareSerial)

		// Create test declarations - 5 with same timestamp, 3 with different timestamps
		declaration1 := createDeclaration(t, "test-declaration-uuid-5-1", "Test Declaration 5-1", "com.example.test.declaration.5.1")
		declaration2 := createDeclaration(t, "test-declaration-uuid-5-2", "Test Declaration 5-2", "com.example.test.declaration.5.2")
		declaration3 := createDeclaration(t, "test-declaration-uuid-5-3", "Test Declaration 5-3", "com.example.test.declaration.5.3")
		declaration4 := createDeclaration(t, "test-declaration-uuid-5-4", "Test Declaration 5-4", "com.example.test.declaration.5.4")
		declaration5 := createDeclaration(t, "test-declaration-uuid-5-5", "Test Declaration 5-5", "com.example.test.declaration.5.5")
		declaration6 := createDeclaration(t, "test-declaration-uuid-5-6", "Test Declaration 5-6", "com.example.test.declaration.5.6")
		declaration7 := createDeclaration(t, "test-declaration-uuid-5-7", "Test Declaration 5-7", "com.example.test.declaration.5.7")
		declaration8 := createDeclaration(t, "test-declaration-uuid-5-8", "Test Declaration 5-8", "com.example.test.declaration.5.8")

		// Set up device and enrollment records
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		// Insert host declarations
		token1 := insertHostDeclaration(t, hostUUID, declaration1.DeclarationUUID, "pending", "install", declaration1.Identifier)
		token2 := insertHostDeclaration(t, hostUUID, declaration2.DeclarationUUID, "pending", "install", declaration2.Identifier)
		token3 := insertHostDeclaration(t, hostUUID, declaration3.DeclarationUUID, "pending", "install", declaration3.Identifier)
		token4 := insertHostDeclaration(t, hostUUID, declaration4.DeclarationUUID, "pending", "install", declaration4.Identifier)
		token5 := insertHostDeclaration(t, hostUUID, declaration5.DeclarationUUID, "pending", "install", declaration5.Identifier)
		token6 := insertHostDeclaration(t, hostUUID, declaration6.DeclarationUUID, "pending", "install", declaration6.Identifier)
		token7 := insertHostDeclaration(t, hostUUID, declaration7.DeclarationUUID, "pending", "install", declaration7.Identifier)
		token8 := insertHostDeclaration(t, hostUUID, declaration8.DeclarationUUID, "pending", "install", declaration8.Identifier)

		// Set the same uploaded_at timestamp for first 5 declarations
		sameTimestamp := time.Now()
		setDeclarationUploadedAt(t, declaration1.DeclarationUUID, sameTimestamp)
		setDeclarationUploadedAt(t, declaration2.DeclarationUUID, sameTimestamp)
		setDeclarationUploadedAt(t, declaration3.DeclarationUUID, sameTimestamp)
		setDeclarationUploadedAt(t, declaration4.DeclarationUUID, sameTimestamp)
		setDeclarationUploadedAt(t, declaration5.DeclarationUUID, sameTimestamp)

		// Set different uploaded_at timestamps for the other 3 declarations
		setDeclarationUploadedAt(t, declaration6.DeclarationUUID, sameTimestamp.Add(1*time.Hour))
		setDeclarationUploadedAt(t, declaration7.DeclarationUUID, sameTimestamp.Add(2*time.Hour))
		setDeclarationUploadedAt(t, declaration8.DeclarationUUID, sameTimestamp.Add(3*time.Hour))

		// Get the expected declarations token from the DB.
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 8, 8)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)

		// Verify the declarations in the response
		configIdentifiers := make([]string, 8)
		configTokens := make([]string, 8)
		for i, config := range response.Declarations.Configurations {
			configIdentifiers[i] = config.Identifier
			configTokens[i] = config.ServerToken
		}

		// Check that all declarations are included
		require.Contains(t, configIdentifiers, declaration1.Identifier)
		require.Contains(t, configIdentifiers, declaration2.Identifier)
		require.Contains(t, configIdentifiers, declaration3.Identifier)
		require.Contains(t, configIdentifiers, declaration4.Identifier)
		require.Contains(t, configIdentifiers, declaration5.Identifier)
		require.Contains(t, configIdentifiers, declaration6.Identifier)
		require.Contains(t, configIdentifiers, declaration7.Identifier)
		require.Contains(t, configIdentifiers, declaration8.Identifier)

		// Check that all tokens are included
		require.Contains(t, configTokens, token1)
		require.Contains(t, configTokens, token2)
		require.Contains(t, configTokens, token3)
		require.Contains(t, configTokens, token4)
		require.Contains(t, configTokens, token5)
		require.Contains(t, configTokens, token6)
		require.Contains(t, configTokens, token7)
		require.Contains(t, configTokens, token8)

		// Verify the activations in the response
		activationIdentifiers := make([]string, 8)
		activationTokens := make([]string, 8)
		for i, activation := range response.Declarations.Activations {
			activationIdentifiers[i] = activation.Identifier
			activationTokens[i] = activation.ServerToken
		}

		// Check that all activation identifiers are included
		require.Contains(t, activationIdentifiers, declaration1.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration2.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration3.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration4.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration5.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration6.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration7.DeclarationUUID+".activation")
		require.Contains(t, activationIdentifiers, declaration8.DeclarationUUID+".activation")

		// Check that all activation tokens are included
		require.Contains(t, activationTokens, token1)
		require.Contains(t, activationTokens, token2)
		require.Contains(t, activationTokens, token3)
		require.Contains(t, activationTokens, token4)
		require.Contains(t, activationTokens, token5)
		require.Contains(t, activationTokens, token6)
		require.Contains(t, activationTokens, token7)
		require.Contains(t, activationTokens, token8)
	})

	t.Run("UserChannelScopeIsolation", func(t *testing.T) {
		hostUUID := "test-host-uuid-user-scope"
		hardwareSerial := "ABC123-USER-SCOPE"
		userEnrollmentID := hostUUID + ":user-1"

		createHost(t, hostUUID, hardwareSerial)
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)
		// Add a user-channel enrollment for the same device.
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_users (id, device_id, user_short_name, user_long_name) VALUES (?, ?, ?, ?)`,
				"user-1", hostUUID, "u", "user")
			if err != nil {
				return err
			}
			_, err = q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, user_id, type, topic, push_magic, token_hex, enabled, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				userEnrollmentID, hostUUID, "user-1", "User", "topic", "push_magic", "token_hex_user", 1, time.Now())
			return err
		})

		// A device-scoped and a user-scoped declaration.
		deviceDecl := createDeclaration(t, "user-scope-device-decl", "DeviceDecl", "com.example.userscope.device")
		userDeclRaw := &fleet.MDMAppleDeclaration{
			DeclarationUUID: "user-scope-user-decl",
			Name:            "UserDecl",
			Identifier:      "com.example.userscope.user",
			RawJSON:         []byte(`{"Type":"com.apple.test.declaration","Identifier":"com.example.userscope.user"}`),
			Scope:           fleet.PayloadScopeUser,
		}
		userDecl, err := ds.NewMDMAppleDeclaration(ctx, userDeclRaw, nil)
		require.NoError(t, err)

		// Apple supports management declarations on the user channel too, and
		// nothing in Fleet scopes by declaration type, so it must ride along.
		userMgmtDecl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			DeclarationUUID: "user-scope-user-mgmt",
			Name:            "UserMgmtDecl",
			Identifier:      "com.example.userscope.mgmt",
			RawJSON:         []byte(`{"Type":"com.apple.management.organization-info","Identifier":"com.example.userscope.mgmt","Payload":{"Name":"Fleet"}}`),
			Scope:           fleet.PayloadScopeUser,
		}, nil)
		require.NoError(t, err)

		insertScopedHostDeclaration := func(declUUID, identifier string, scope fleet.PayloadScope) {
			mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				var token string
				if err := sqlx.GetContext(ctx, q, &token, "SELECT HEX(token) FROM mdm_apple_declarations WHERE declaration_uuid = ?", declUUID); err != nil {
					return err
				}
				_, err := q.ExecContext(ctx, `
					INSERT INTO host_mdm_apple_declarations
					(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier, scope)
					VALUES (?, ?, 'pending', 'install', UNHEX(?), ?, ?)`,
					hostUUID, declUUID, token, identifier, scope)
				return err
			})
		}
		insertScopedHostDeclaration(deviceDecl.DeclarationUUID, deviceDecl.Identifier, fleet.PayloadScopeSystem)
		insertScopedHostDeclaration(userDecl.DeclarationUUID, userDecl.Identifier, fleet.PayloadScopeUser)
		insertScopedHostDeclaration(userMgmtDecl.DeclarationUUID, userMgmtDecl.Identifier, fleet.PayloadScopeUser)

		callChannel := func(enrollID *mdm.EnrollID) fleet.MDMAppleDDMDeclarationItemsResponse {
			req := mdm.Request{Context: ctx, EnrollID: enrollID}
			dm := mdm.DeclarativeManagement{}
			dm.UDID = hostUUID
			dm.Endpoint = "declaration-items"
			response, err := ddmService.DeclarativeManagement(&req, &dm)
			require.NoError(t, err)
			require.NotNil(t, response)
			var parsed fleet.MDMAppleDDMDeclarationItemsResponse
			require.NoError(t, json.Unmarshal(response, &parsed))
			return parsed
		}

		// Device channel: only the device declaration, and the token matches the
		// SQL-computed System token (parity).
		deviceResp := callChannel(&mdm.EnrollID{ID: hostUUID})
		require.Len(t, deviceResp.Declarations.Configurations, 1)
		require.Equal(t, deviceDecl.Identifier, deviceResp.Declarations.Configurations[0].Identifier)
		require.Empty(t, deviceResp.Declarations.Management, "the user-scoped management declaration must not leak to the device channel")
		sysToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)
		require.Equal(t, sysToken.DeclarationsToken, deviceResp.DeclarationsToken)

		// User channel (EnrollID with ParentID set): only the user declaration, and
		// the token matches the SQL-computed User token (parity).
		userResp := callChannel(&mdm.EnrollID{ID: userEnrollmentID, ParentID: hostUUID})
		require.Len(t, userResp.Declarations.Configurations, 1)
		require.Equal(t, userDecl.Identifier, userResp.Declarations.Configurations[0].Identifier)
		require.Len(t, userResp.Declarations.Management, 1)
		require.Equal(t, userMgmtDecl.Identifier, userResp.Declarations.Management[0].Identifier)
		userToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID, fleet.PayloadScopeUser)
		require.NoError(t, err)
		require.Equal(t, userToken.DeclarationsToken, userResp.DeclarationsToken)

		// The two channels produce different tokens.
		require.NotEqual(t, deviceResp.DeclarationsToken, userResp.DeclarationsToken)
	})

	t.Run("DeliveryStripsPayloadScope", func(t *testing.T) {
		hostUUID := "test-host-uuid-strip"
		hardwareSerial := "ABC123-STRIP"
		createHost(t, hostUUID, hardwareSerial)
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		// The stored declaration retains the Fleet-only top-level PayloadScope key
		// (it's only stripped at delivery).
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			DeclarationUUID: "strip-decl",
			Name:            "StripDecl",
			Identifier:      "com.example.strip",
			RawJSON:         []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.strip","PayloadScope":"System","Payload":{"Enabled":true}}`),
			Scope:           fleet.PayloadScopeSystem,
		}, nil)
		require.NoError(t, err)
		require.Contains(t, string(decl.RawJSON), "PayloadScope", "stored raw_json keeps PayloadScope")

		insertHostDeclaration(t, hostUUID, decl.DeclarationUUID, "pending", "install", decl.Identifier)

		// Fetch the full configuration declaration served to the device.
		req := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: hostUUID}}
		dm := mdm.DeclarativeManagement{}
		dm.UDID = hostUUID
		dm.Endpoint = "declaration/configuration/" + decl.Identifier
		response, err := ddmService.DeclarativeManagement(&req, &dm)
		require.NoError(t, err)

		var served map[string]any
		require.NoError(t, json.Unmarshal(response, &served))
		require.NotContains(t, served, "PayloadScope", "PayloadScope must be stripped from the declaration served to the device")
		require.Equal(t, "com.example.strip", served["Identifier"])
		require.Contains(t, served, "ServerToken")
	})

	t.Run("PredicateStatusMapping", func(t *testing.T) {
		// Payloads below are the real reports a macOS 26 host sent to a Fleet
		// server, not hand-written. Apple splits a predicate outcome across two
		// arrays: the activation carries Info.Predicate, while the configuration
		// it gates reports Error.ActivationFailed. Reading only the
		// configuration makes a host the predicate simply excluded look failed.
		predicateFalse := func(configIdent, activationIdent, token string) fleet.MDMAppleDDMStatusReport {
			var r fleet.MDMAppleDDMStatusReport
			r.StatusItems.Management.Declarations.Activations = []fleet.MDMAppleDDMStatusDeclaration{{
				Identifier:  activationIdent,
				Active:      false,
				Valid:       fleet.MDMAppleDeclarationValid,
				ServerToken: token,
				Reasons: []fleet.MDMAppleDDMStatusErrorReason{{
					Code:        fleet.MDMAppleDDMReasonPredicate,
					Description: "Activations (" + activationIdent + ") predicate (FALSEPREDICATE) evaluated to false.",
					Details: map[string]any{
						"Identifier":  activationIdent,
						"ServerToken": token,
						"Predicate":   "FALSEPREDICATE",
					},
				}},
			}}
			r.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{{
				Identifier:  configIdent,
				Active:      false,
				Valid:       fleet.MDMAppleDeclarationUnknown,
				ServerToken: token,
				Reasons: []fleet.MDMAppleDDMStatusErrorReason{{
					Code:        fleet.MDMAppleDDMReasonActivationFailed,
					Description: "Activation " + activationIdent + " has errors.",
					Details: map[string]any{
						"Identifier":  activationIdent,
						"ServerToken": token,
					},
				}},
			}}
			return r
		}

		// Same shape, but the activation reports no Info.Predicate -- the
		// activation genuinely failed rather than being scoped out.
		activationBroken := func(configIdent, activationIdent, token string) fleet.MDMAppleDDMStatusReport {
			r := predicateFalse(configIdent, activationIdent, token)
			r.StatusItems.Management.Declarations.Activations[0].Reasons = nil
			return r
		}

		applied := func(configIdent, token string) fleet.MDMAppleDDMStatusReport {
			var r fleet.MDMAppleDDMStatusReport
			r.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{{
				Identifier:  configIdent,
				Active:      true,
				Valid:       fleet.MDMAppleDeclarationValid,
				ServerToken: token,
			}}
			return r
		}

		// Apple: "A management declaration has an active state which is always
		// false and not part of the activation process", so these report
		// Active:false even when fully applied. Grading them like configurations
		// left every one of them stuck on verifying.
		management := func(ident, token string, valid fleet.MDMAppleDeclarationValidity) fleet.MDMAppleDDMStatusReport {
			var r fleet.MDMAppleDDMStatusReport
			r.StatusItems.Management.Declarations.Management = []fleet.MDMAppleDDMStatusDeclaration{{
				Identifier:  ident,
				Active:      false,
				Valid:       valid,
				ServerToken: token,
			}}
			return r
		}

		cases := []struct {
			name        string
			report      func(configIdent, activationIdent, token string) fleet.MDMAppleDDMStatusReport
			declRawJSON string
			wantStatus  fleet.MDMDeliveryStatus
			wantDetail  string
		}{
			{
				name: "valid management declaration is verified despite being inactive",
				report: func(ident, _ string, token string) fleet.MDMAppleDDMStatusReport {
					return management(ident, token, fleet.MDMAppleDeclarationValid)
				},
				declRawJSON: `{"Type":"com.apple.management.organization-info","Identifier":"%s","Payload":{"Name":"Fleet"}}`,
				wantStatus:  fleet.MDMDeliveryVerified,
			},
			{
				name: "invalid management declaration is failed",
				report: func(ident, _ string, token string) fleet.MDMAppleDDMStatusReport {
					return management(ident, token, fleet.MDMAppleDeclarationInvalid)
				},
				declRawJSON: `{"Type":"com.apple.management.organization-info","Identifier":"%s","Payload":{"Name":"Fleet"}}`,
				wantStatus:  fleet.MDMDeliveryFailed,
			},
			{
				name: "unchecked management declaration stays verifying",
				report: func(ident, _ string, token string) fleet.MDMAppleDDMStatusReport {
					return management(ident, token, fleet.MDMAppleDeclarationUnknown)
				},
				declRawJSON: `{"Type":"com.apple.management.organization-info","Identifier":"%s","Payload":{"Name":"Fleet"}}`,
				wantStatus:  fleet.MDMDeliveryVerifying,
			},
			{
				// Unknown means "not checked yet", but reasons mean something already
				// went wrong -- without this it would wait for a verdict forever.
				name: "unknown management declaration reporting errors is failed",
				report: func(ident, _ string, token string) fleet.MDMAppleDDMStatusReport {
					r := management(ident, token, fleet.MDMAppleDeclarationUnknown)
					r.StatusItems.Management.Declarations.Management[0].Reasons = []fleet.MDMAppleDDMStatusErrorReason{{
						Code:        "Error.InvalidPayload",
						Description: "ManagementPayload (" + ident + ") has an invalid payload.",
					}}
					return r
				},
				declRawJSON: `{"Type":"com.apple.management.organization-info","Identifier":"%s","Payload":{"Name":"Fleet"}}`,
				wantStatus:  fleet.MDMDeliveryFailed,
			},
			{
				name:       "predicate excluded the host is verified, not failed",
				report:     predicateFalse,
				wantStatus: fleet.MDMDeliveryVerified,
				wantDetail: "Fleet verified, but predicate (FALSEPREDICATE) evaluated to false and settings were not applied to this host.",
			},
			{
				name:       "activation failure without a predicate reason is failed",
				report:     activationBroken,
				wantStatus: fleet.MDMDeliveryFailed,
			},
			{
				name: "applied configuration is verified",
				report: func(configIdent, _ string, token string) fleet.MDMAppleDDMStatusReport {
					return applied(configIdent, token)
				},
				wantStatus: fleet.MDMDeliveryVerified,
			},
		}

		for i, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				suffix := fmt.Sprintf("%d", i)
				hostUUID := "test-host-uuid-pred-" + suffix
				hardwareSerial := "PRED-" + suffix
				createHost(t, hostUUID, hardwareSerial)
				setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

				configIdent := "com.example.pred." + suffix
				activationIdent := configIdent + ".custom"
				rawJSON := `{"Type":"com.apple.configuration.test","Identifier":"` + configIdent + `","Payload":{"Enabled":true}}`
				if c.declRawJSON != "" {
					rawJSON = fmt.Sprintf(c.declRawJSON, configIdent)
				}
				decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
					Name:       "PredDecl-" + suffix,
					Identifier: configIdent,
					RawJSON:    []byte(rawJSON),
					Scope:      fleet.PayloadScopeSystem,
				}, nil)
				require.NoError(t, err)
				token := insertHostDeclaration(t, hostUUID, decl.DeclarationUUID, "pending", "install", decl.Identifier)

				raw, err := json.Marshal(c.report(configIdent, activationIdent, token))
				require.NoError(t, err)

				req := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: hostUUID}}
				dm := mdm.DeclarativeManagement{Data: raw}
				dm.UDID = hostUUID
				dm.Endpoint = "status"
				_, err = ddmService.DeclarativeManagement(&req, &dm)
				require.NoError(t, err)

				var got struct {
					Status *string `db:"status"`
					Detail string  `db:"detail"`
				}
				mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(ctx, q, &got,
						`SELECT status, COALESCE(detail, '') AS detail FROM host_mdm_apple_declarations WHERE host_uuid = ?`, hostUUID)
				})
				require.NotNil(t, got.Status)
				require.Equal(t, string(c.wantStatus), *got.Status)
				if c.wantDetail != "" {
					require.Equal(t, c.wantDetail, got.Detail)
				}
			})
		}
	})

	t.Run("CustomActivationIsScopedToHostsThatHaveTheDeclaration", func(t *testing.T) {
		inScopeUUID, inScopeSerial := "test-host-uuid-scoped-in", "SCOPE-IN"
		outOfScopeUUID, outOfScopeSerial := "test-host-uuid-scoped-out", "SCOPE-OUT"

		createHost(t, inScopeUUID, inScopeSerial)
		setupDeviceAndEnrollment(t, inScopeUUID, inScopeSerial)
		createHost(t, outOfScopeUUID, outOfScopeSerial)
		setupDeviceAndEnrollment(t, outOfScopeUUID, outOfScopeSerial)

		teamID := uint(42)
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Name:       "ScopedDecl",
			Identifier: "com.example.scoped",
			TeamID:     &teamID,
			RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.scoped","Payload":{"Enabled":true}}`),
			Scope:      fleet.PayloadScopeSystem,
			Activation: &fleet.MDMAppleCustomActivation{
				Identifier:              "com.example.scoped.act",
				RawJSON:                 []byte(`{"Type":"com.apple.activation.simple","Identifier":"com.example.scoped.act","Payload":{"StandardConfigurations":["com.example.scoped"]}}`),
				ConfigurationIdentifier: "com.example.scoped",
			},
		}, nil)
		require.NoError(t, err)

		// Only the in-scope host gets a host_mdm_apple_declarations row, which is
		// what team and label scoping ultimately produce.
		insertHostDeclaration(t, inScopeUUID, decl.DeclarationUUID, "pending", "install", decl.Identifier)

		manifest := callDeclarativeManagementAndVerify(t, inScopeUUID, 1, 1)
		require.Equal(t, "com.example.scoped.act", manifest.Declarations.Activations[0].Identifier)

		req := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: inScopeUUID}}
		dm := mdm.DeclarativeManagement{}
		dm.UDID = inScopeUUID
		dm.Endpoint = "declaration/activation/com.example.scoped.act"
		_, err = ddmService.DeclarativeManagement(&req, &dm)
		require.NoError(t, err)

		// The out-of-scope host sees nothing, and cannot fetch the activation by
		// name even though it exists in the database.
		outManifest := callDeclarativeManagementAndVerify(t, outOfScopeUUID, 0, 0)
		require.Empty(t, outManifest.Declarations.Activations)

		outReq := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: outOfScopeUUID}}
		outDM := mdm.DeclarativeManagement{}
		outDM.UDID = outOfScopeUUID
		outDM.Endpoint = "declaration/activation/com.example.scoped.act"
		_, err = ddmService.DeclarativeManagement(&outReq, &outDM)
		require.Error(t, err, "a host outside the declaration's scope must not resolve its activation")
	})

	t.Run("CustomActivationCarriesItsOwnToken", func(t *testing.T) {
		hostUUID, hardwareSerial := "test-host-uuid-acttoken", "ACT-TOKEN"
		createHost(t, hostUUID, hardwareSerial)
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		teamID := uint(43)
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Name:       "ActTokenDecl",
			Identifier: "com.example.acttoken",
			TeamID:     &teamID,
			RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.acttoken","Payload":{"Enabled":true}}`),
			Scope:      fleet.PayloadScopeSystem,
			Activation: &fleet.MDMAppleCustomActivation{
				Identifier:              "com.example.acttoken.act",
				RawJSON:                 []byte(`{"Type":"com.apple.activation.simple","Identifier":"com.example.acttoken.act","Payload":{"StandardConfigurations":["com.example.acttoken"]}}`),
				ConfigurationIdentifier: "com.example.acttoken",
			},
		}, nil)
		require.NoError(t, err)
		insertHostDeclaration(t, hostUUID, decl.DeclarationUUID, "pending", "install", decl.Identifier)

		manifest := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)
		advertised := manifest.Declarations.Activations[0].ServerToken

		// The activation's token is its own, not the declaration's: otherwise
		// editing the declaration would needlessly re-sync the activation.
		require.NotEqual(t, manifest.Declarations.Configurations[0].ServerToken, advertised,
			"a custom activation must not ride on the declaration's token")

		// What the device fetches has to carry exactly what was advertised, or it
		// re-fetches forever.
		req := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: hostUUID}}
		dm := mdm.DeclarativeManagement{}
		dm.UDID = hostUUID
		dm.Endpoint = "declaration/activation/com.example.acttoken.act"
		served, err := ddmService.DeclarativeManagement(&req, &dm)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(served, &body))
		require.Equal(t, advertised, body["ServerToken"],
			"the served activation token must match the manifest")
	})

	t.Run("ActivationTokenFoldsInVariablesUpdatedAt", func(t *testing.T) {
		hostUUID, hardwareSerial := "test-host-uuid-acttok2", "ACT-TOK2"
		createHost(t, hostUUID, hardwareSerial)
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		teamID := uint(45)
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Name:       "ActTokDecl",
			Identifier: "com.example.acttok",
			TeamID:     &teamID,
			RawJSON:    []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.acttok","Payload":{"Enabled":true}}`),
			Scope:      fleet.PayloadScopeSystem,
			Activation: &fleet.MDMAppleCustomActivation{
				Identifier:              "com.example.acttok.act",
				RawJSON:                 []byte(`{"Type":"com.apple.activation.simple","Identifier":"com.example.acttok.act","Payload":{"StandardConfigurations":["com.example.acttok"]}}`),
				ConfigurationIdentifier: "com.example.acttok",
			},
		}, nil)
		require.NoError(t, err)
		insertHostDeclaration(t, hostUUID, decl.DeclarationUUID, "pending", "install", decl.Identifier)

		before := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1).Declarations.Activations[0].ServerToken

		// A variable's value changing bumps variables_updated_at. The activation
		// is expanded per host, so its token has to move too -- otherwise the host
		// re-syncs, re-fetches the configuration, and keeps the stale activation.
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE host_mdm_apple_declarations SET variables_updated_at = ? WHERE host_uuid = ? AND declaration_uuid = ?`,
				time.Now().UTC(), hostUUID, decl.DeclarationUUID)
			return err
		})

		after := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1).Declarations.Activations[0].ServerToken
		require.NotEqual(t, before, after, "the activation token must change when variables_updated_at does")

		// Delivery has to agree with the new advertised token.
		req := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: hostUUID}}
		dm := mdm.DeclarativeManagement{}
		dm.UDID = hostUUID
		dm.Endpoint = "declaration/activation/com.example.acttok.act"
		served, err := ddmService.DeclarativeManagement(&req, &dm)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(served, &body))
		require.Equal(t, after, body["ServerToken"])
	})

	t.Run("ActivationVariablesCheckedWhenDeclarationHasNone", func(t *testing.T) {
		hostUUID, hardwareSerial := "test-host-uuid-actvar", "ACT-VAR"
		createHost(t, hostUUID, hardwareSerial)
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		teamID := uint(44)
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Name:       "ActVarDecl",
			Identifier: "com.example.actvar",
			TeamID:     &teamID,
			// No variables in the declaration itself.
			RawJSON: []byte(`{"Type":"com.apple.configuration.test","Identifier":"com.example.actvar","Payload":{"Enabled":true}}`),
			Scope:   fleet.PayloadScopeSystem,
			Activation: &fleet.MDMAppleCustomActivation{
				Identifier: "com.example.actvar.act",
				// ...but the activation references a vital that doesn't exist.
				RawJSON:                 []byte(`{"Type":"com.apple.activation.simple","Identifier":"com.example.actvar.act","Payload":{"StandardConfigurations":["com.example.actvar"],"Predicate":"$FLEET_HOST_VITAL_999999 == 'x'"}}`),
				ConfigurationIdentifier: "com.example.actvar",
			},
		}, nil)
		require.NoError(t, err)
		insertHostDeclaration(t, hostUUID, decl.DeclarationUUID, "pending", "install", decl.Identifier)

		// The activation's variables have to be checked on the activation itself.
		// Gating on the declaration's variables_updated_at skipped this entirely,
		// leaving the host with an activation it could never resolve.
		manifest := callDeclarativeManagementAndVerify(t, hostUUID, 0, 0)
		require.Empty(t, manifest.Declarations.Configurations)
		require.Empty(t, manifest.Declarations.Activations)

		var status string
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &status,
				`SELECT status FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_uuid = ?`,
				hostUUID, decl.DeclarationUUID)
		})
		require.Equal(t, string(fleet.MDMDeliveryFailed), status)
	})

	t.Run("ManagementDeclarationRoutingAndEndpointGuard", func(t *testing.T) {
		hostUUID := "test-host-uuid-mgmt"
		hardwareSerial := "ABC123-MGMT"

		createHost(t, hostUUID, hardwareSerial)
		setupDeviceAndEnrollment(t, hostUUID, hardwareSerial)

		mgmt, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Name:       "MgmtDecl",
			Identifier: "com.example.mgmt",
			RawJSON:    []byte(`{"Type":"com.apple.management.organization-info","Identifier":"com.example.mgmt","Payload":{"Echo":"foo"}}`),
			Scope:      fleet.PayloadScopeSystem,
		}, nil)
		require.NoError(t, err)
		insertHostDeclaration(t, hostUUID, mgmt.DeclarationUUID, "pending", "install", mgmt.Identifier)

		// Management declarations are never activated, so no activation is
		// synthesized and they don't appear under Configurations.
		manifest := callDeclarativeManagementAndVerify(t, hostUUID, 0, 0)
		require.Len(t, manifest.Declarations.Management, 1)
		require.Equal(t, mgmt.Identifier, manifest.Declarations.Management[0].Identifier)

		req := mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: hostUUID}}
		dm := mdm.DeclarativeManagement{}
		dm.UDID = hostUUID
		dm.Endpoint = "declaration/management/" + mgmt.Identifier
		response, err := ddmService.DeclarativeManagement(&req, &dm)
		require.NoError(t, err)
		var served map[string]any
		require.NoError(t, json.Unmarshal(response, &served))
		require.Equal(t, "com.apple.management.organization-info", served["Type"])

		// The two endpoints must not serve each other's rows.
		dm.Endpoint = "declaration/configuration/" + mgmt.Identifier
		_, err = ddmService.DeclarativeManagement(&req, &dm)
		require.Error(t, err)
	})
}

// osUpdatesDeclContents is a minimal DDM software-update declaration body that
// references both host-target OS Fleet variables. It is used by the OS-updates
// DDM sync tests below.
const osUpdatesDeclContents = `{
	"Type": "com.apple.configuration.softwareupdate.enforcement.specific",
	"Identifier": "com.fleetdm.fleet.mdm.os-updates.macos",
	"Payload": {
		"TargetOSVersion": "$FLEET_VAR_HOST_TARGET_OS_VERSION",
		"TargetLocalDateTime": "${FLEET_VAR_HOST_TARGET_OS_DEADLINE}T12:00:00"
	}
}`

// TestReplaceDeclarationFleetVariablesOSUpdateTargets covers the variable
// resolution branch for the OS-update target variables:
//   - the tracking row exists but the target hasn't been computed yet -> the
//     declaration is deferred with a notReadyYetError (so the caller marks it
//     pending, not failed),
//   - the target is set -> the version and RFC3339 deadline are substituted,
//   - there is no tracking row at all -> a hard error (marked failed).
func TestReplaceDeclarationFleetVariablesOSUpdateTargets(t *testing.T) {
	ctx := t.Context()
	ds := mysqltest.CreateMySQLDS(t)
	svc := MDMAppleDDMService{
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	newHost := func(hostUUID string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			UUID:            hostUUID,
			Hostname:        hostUUID,
			OsqueryHostID:   new(hostUUID),
			NodeKey:         new(hostUUID),
			Platform:        "darwin",
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)
		return h
	}

	t.Run("target not yet computed defers with notReadyYetError", func(t *testing.T) {
		h := newHost("os-var-notready")
		// A tracking row exists (device id captured) but no target has been set.
		require.NoError(t, ds.InsertAppleSoftwareUpdateDeviceID(ctx, h.UUID, "Mac14,2"))

		_, err := svc.replaceDeclarationFleetVariables(ctx, osUpdatesDeclContents, h.UUID)
		var notReady notReadyYetError
		require.ErrorAs(t, err, &notReady)
		require.Contains(t, notReady.Message, "not yet available")
		require.Contains(t, notReady.Message, "resend this profile once available")
	})

	t.Run("target set substitutes version and DateOnly noon deadline", func(t *testing.T) {
		h := newHost("os-var-ready")
		require.NoError(t, ds.InsertAppleSoftwareUpdateDeviceID(ctx, h.UUID, "Mac14,2"))

		deadline, _ := time.Parse(time.DateOnly, time.Now().UTC().Add(48*time.Hour).Format(time.DateOnly))
		resolvedAt := time.Now().UTC().Truncate(time.Microsecond)
		require.NoError(t, ds.SetAppleOSUpdateTargetsAndResend(ctx, []*fleet.ComputedAppleSoftwareUpdateHost{{
			AppleSoftwareUpdateHost: fleet.AppleSoftwareUpdateHost{
				HostUUID: h.UUID, TargetOSVersion: "15.1", TargetDeadline: &deadline, ResolvedAt: &resolvedAt,
			},
		}}))

		out, err := svc.replaceDeclarationFleetVariables(ctx, osUpdatesDeclContents, h.UUID)
		require.NoError(t, err)
		require.NotContains(t, out, "$FLEET_VAR")

		deadline = deadline.Add(12 * time.Hour) // noon local time

		var parsed struct {
			Payload struct {
				TargetOSVersion     string
				TargetLocalDateTime string
			}
		}
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		require.Equal(t, "15.1", parsed.Payload.TargetOSVersion)
		gotDeadline, err := time.Parse("2006-01-02T15:04:05", parsed.Payload.TargetLocalDateTime)
		require.NoError(t, err)
		require.True(t, deadline.Equal(gotDeadline), "want %s got %s", deadline, gotDeadline)
	})

	t.Run("missing tracking row is a hard error, not a defer", func(t *testing.T) {
		h := newHost("os-var-missing") // no InsertAppleSoftwareUpdateDeviceID -> no tracking row

		_, err := svc.replaceDeclarationFleetVariables(ctx, osUpdatesDeclContents, h.UUID)
		require.Error(t, err)
		var notReady notReadyYetError
		require.NotErrorAs(t, err, &notReady, "a missing tracking row must not defer")
		require.Contains(t, err.Error(), "not found")
	})
}

// TestDeclarativeManagementOSUpdatesPendingThenResolved exercises the full
// DDM-sync piece end to end at the handler layer: an OS-update declaration whose
// target variables can't be resolved yet is marked pending (with a user-facing
// detail) and excluded from the manifest; then, once the cron's datastore write
// sets the target and bumps the declaration for resend, the same fetches resolve
// and the declaration is served with concrete values.
func TestDeclarativeManagementOSUpdatesPendingThenResolved(t *testing.T) {
	ctx := t.Context()
	ds := mysqltest.CreateMySQLDS(t)
	svc := MDMAppleDDMService{
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	const (
		hostUUID       = "os-updates-ddm-host"
		deviceID       = "Mac14,2"
		declIdentifier = "com.fleetdm.fleet.mdm.os-updates.macos"
	)

	_, err := ds.NewHost(ctx, &fleet.Host{
		UUID:            hostUUID,
		Hostname:        hostUUID,
		OsqueryHostID:   new(hostUUID),
		NodeKey:         new(hostUUID),
		Platform:        "darwin",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)

	// Fleet-managed OS-updates DDM declaration referencing the two target vars.
	decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		DeclarationUUID: "os-updates-decl-uuid",
		Name:            fleetmdm.FleetMacOSUpdatesProfileName,
		Identifier:      declIdentifier,
		RawJSON:         []byte(osUpdatesDeclContents),
		Scope:           fleet.PayloadScopeSystem,
	}, []fleet.FleetVarName{fleet.FleetVarHostTargetOSVersion, fleet.FleetVarHostTargetOSDeadline})
	require.NoError(t, err)

	// Assign the declaration to the host. variables_updated_at is set (as the
	// reconciler would) so the sync path attempts variable resolution; status
	// starts NULL (freshly assigned, not yet delivered).
	initialVarsUpdated := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var token string
		if err := sqlx.GetContext(ctx, q, &token,
			"SELECT HEX(token) FROM mdm_apple_declarations WHERE declaration_uuid = ?", decl.DeclarationUUID); err != nil {
			return err
		}
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_apple_declarations
				(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier, declaration_name, scope, variables_updated_at)
			VALUES (?, ?, NULL, 'install', UNHEX(?), ?, ?, 'System', ?)`,
			hostUUID, decl.DeclarationUUID, token, declIdentifier, fleetmdm.FleetMacOSUpdatesProfileName, initialVarsUpdated)
		return err
	})

	// Tracking row exists (device id captured) but the target isn't computed yet.
	require.NoError(t, ds.InsertAppleSoftwareUpdateDeviceID(ctx, hostUUID, deviceID))

	readHostDecl := func() (status *string, detail string) {
		var row struct {
			Status *string `db:"status"`
			Detail string  `db:"detail"`
		}
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row,
				`SELECT status, COALESCE(detail, '') AS detail FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_uuid = ?`,
				hostUUID, decl.DeclarationUUID)
		})
		return row.Status, row.Detail
	}
	readVarsUpdatedAt := func() *time.Time {
		var vals []time.Time
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &vals,
				`SELECT variables_updated_at FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_uuid = ? AND variables_updated_at IS NOT NULL`,
				hostUUID, decl.DeclarationUUID)
		})
		if len(vals) == 0 {
			return nil
		}
		return &vals[0]
	}
	configItems := func() []fleet.MDMAppleDDMManifest {
		body, err := svc.handleDeclarationItems(ctx, hostUUID, fleet.PayloadScopeSystem)
		require.NoError(t, err)
		var resp fleet.MDMAppleDDMDeclarationItemsResponse
		require.NoError(t, json.Unmarshal(body, &resp))
		return resp.Declarations.Configurations
	}

	configParts := []string{"declaration", "configuration", declIdentifier}

	// === Phase 1: target not ready -> pending with detail, excluded from manifest ===

	body, err := svc.handleConfigurationDeclaration(ctx, configParts, hostUUID, fleet.PayloadScopeSystem, false)
	require.NoError(t, err)
	require.Nil(t, body, "an unresolvable declaration is served as an empty 200")

	status, detail := readHostDecl()
	require.NotNil(t, status)
	require.Equal(t, string(fleet.MDMDeliveryPending), *status)
	require.Contains(t, detail, "not yet available")

	for _, c := range configItems() {
		require.NotEqual(t, declIdentifier, c.Identifier, "unresolvable declaration must be excluded from the manifest")
	}
	// handleDeclarationItems also keeps it pending with the detail.
	status, detail = readHostDecl()
	require.NotNil(t, status)
	require.Equal(t, string(fleet.MDMDeliveryPending), *status)
	require.Contains(t, detail, "not yet available")

	// === Phase 2: cron computes the target and bumps the declaration for resend ===

	deadline, _ := time.Parse(time.DateOnly, time.Now().UTC().Add(48*time.Hour).Format(time.DateOnly))
	deadline = deadline.Add(12 * time.Hour) // noon local time
	resolvedAt := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, ds.SetAppleOSUpdateTargetsAndResend(ctx, []*fleet.ComputedAppleSoftwareUpdateHost{{
		AppleSoftwareUpdateHost: fleet.AppleSoftwareUpdateHost{
			HostUUID: hostUUID, TargetOSVersion: "15.1", TargetDeadline: &deadline, ResolvedAt: &resolvedAt,
		},
		Resend: true,
	}}))

	// The resend signal: status cleared to NULL and variables_updated_at bumped.
	status, _ = readHostDecl()
	require.Nil(t, status, "resend resets status to NULL")
	bumped := readVarsUpdatedAt()
	require.NotNil(t, bumped)
	require.True(t, bumped.After(initialVarsUpdated), "variables_updated_at should be bumped forward for resend")

	// === Phase 3: the declaration now resolves and is served with concrete values ===

	body, err = svc.handleConfigurationDeclaration(ctx, configParts, hostUUID, fleet.PayloadScopeSystem, false)
	require.NoError(t, err)
	require.NotNil(t, body)
	require.NotContains(t, string(body), "$FLEET_VAR")

	var served struct {
		Identifier string
		Payload    struct {
			TargetOSVersion     string
			TargetLocalDateTime string
		}
		ServerToken string
	}
	require.NoError(t, json.Unmarshal(body, &served))
	require.Equal(t, declIdentifier, served.Identifier)
	require.Equal(t, "15.1", served.Payload.TargetOSVersion)
	require.NotEmpty(t, served.ServerToken)
	gotDeadline, err := time.Parse("2006-01-02T15:04:05", served.Payload.TargetLocalDateTime)
	require.NoError(t, err)
	require.True(t, deadline.Equal(gotDeadline), "want %s got %s", deadline, gotDeadline)

	// And it is now included in the declaration-items manifest.
	found := false
	for _, c := range configItems() {
		if c.Identifier == declIdentifier {
			found = true
		}
	}
	require.True(t, found, "resolved OS-updates declaration should be present in declaration-items")
}
