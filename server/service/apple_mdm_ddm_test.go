package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestDeclarativeManagement_DeclarationItems(t *testing.T) {
	ctx := t.Context()
	ds := mysql.CreateMySQLDS(t)
	logger := log.NewLogfmtLogger(os.Stdout)
	ddmService := MDMAppleDDMService{
		ds:     ds,
		logger: logger,
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
		declaration, err := ds.NewMDMAppleDeclaration(context.Background(), declaration)
		require.NoError(t, err)
		return declaration
	}

	// Helper function to set up device and enrollment records
	setupDeviceAndEnrollment := func(t *testing.T, hostUUID, hardwareSerial string) {
		// Insert the device record first (required for foreign key constraints)
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate) VALUES (?, ?, ?)`,
				hostUUID, hardwareSerial, "test")
			return err
		})

		// Insert a record into nano_enrollments table (required for foreign key constraints)
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				hostUUID, hostUUID, "Device", "topic", "push_magic", "token_hex", 1)
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
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
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
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
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
		mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
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
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
		require.NoError(t, err)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)

		// Verify the declarations in the response
		require.Equal(t, declaration.Identifier, response.Declarations.Configurations[0].Identifier)
		require.Equal(t, token, response.Declarations.Configurations[0].ServerToken)

		// Verify the activations in the response
		require.Equal(t, declaration.Identifier+".activation", response.Declarations.Activations[0].Identifier)
		require.Equal(t, token, response.Declarations.Activations[0].ServerToken)
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
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
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
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
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
		require.Contains(t, activationIdentifiers, declaration1.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration2.Identifier+".activation")
		require.NotContains(t, activationIdentifiers, declaration3.Identifier+".activation")
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
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
		require.NoError(t, err)

		// Call DeclarativeManagement and verify response
		response := callDeclarativeManagementAndVerify(t, hostUUID, 1, 1)

		// Verify the token in the response matches the expected token
		require.Equal(t, expectedToken.DeclarationsToken, response.DeclarationsToken)

		// Verify the declarations in the response (only install operations)
		require.Equal(t, declaration1.Identifier, response.Declarations.Configurations[0].Identifier)
		require.Equal(t, token1, response.Declarations.Configurations[0].ServerToken)

		// Verify the activations in the response
		require.Equal(t, declaration1.Identifier+".activation", response.Declarations.Activations[0].Identifier)
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
		expectedToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
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
		require.Contains(t, activationIdentifiers, declaration1.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration2.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration3.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration4.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration5.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration6.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration7.Identifier+".activation")
		require.Contains(t, activationIdentifiers, declaration8.Identifier+".activation")

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
}
