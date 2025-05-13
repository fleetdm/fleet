package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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

	// Create a test host
	hostUUID := "test-host-uuid"
	osqueryHostID := "1234"
	nodeKey := "1234"
	hardwareSerial := "ABC123"
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		UUID:            hostUUID,
		Hostname:        "test-host",
		HardwareSerial:  hardwareSerial,
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "00:00:00:00:00:00",
		OsqueryHostID:   &osqueryHostID,
		NodeKey:         &nodeKey,
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Create a test declaration
	declaration := &fleet.MDMAppleDeclaration{
		DeclarationUUID: "test-declaration-uuid",
		Name:            "Test Declaration",
		Identifier:      "com.example.test.declaration",
		TeamID:          nil,
		RawJSON:         []byte(`{"Type":"com.apple.test.declaration","Identifier":"com.example.test.declaration"}`),
	}
	declaration, err = ds.NewMDMAppleDeclaration(context.Background(), declaration)
	require.NoError(t, err)

	// Insert the device record first (required for foreign key constraints)
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, serial_number, authenticate) VALUES (?, ?, ?)`,
			hostUUID, hardwareSerial, "test")
		return err
	})

	// Insert a record into nano_enrollments table (required for foreign key constraints)
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex) VALUES (?, ?, ?, ?, ?, ?)`,
			hostUUID, hostUUID, "type", "topic", "push_magic", "token_hex")
		return err
	})

	// Insert the host declaration
	var token string
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		// First, get the right token of the declaration
		err = sqlx.GetContext(ctx, q, &token,
			"SELECT HEX(token) as token FROM mdm_apple_declarations WHERE declaration_uuid = ?", declaration.DeclarationUUID)
		require.NoError(t, err)

		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_apple_declarations 
			(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier) 
			VALUES (?, ?, ?, ?, UNHEX(?), ?)`,
			hostUUID, declaration.DeclarationUUID, "pending", "install", token, declaration.Identifier)
		require.NoError(t, err)
		return nil
	})

	// Call the DeclarativeManagement method
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

	// Get the expected declarations token from the DB.
	declarationsToken, err := ds.MDMAppleDDMDeclarationsToken(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, declarationsToken.DeclarationsToken, declarationItemsResponse.DeclarationsToken)

	// Verify the declarations in the response
	require.Len(t, declarationItemsResponse.Declarations.Configurations, 1)
	assert.Equal(t, declaration.Identifier, declarationItemsResponse.Declarations.Configurations[0].Identifier)
	assert.Equal(t, token, declarationItemsResponse.Declarations.Configurations[0].ServerToken)

	// Verify the activations in the response
	require.Len(t, declarationItemsResponse.Declarations.Activations, 1)
	assert.Equal(t, declaration.Identifier+".activation", declarationItemsResponse.Declarations.Activations[0].Identifier)
	assert.Equal(t, token, declarationItemsResponse.Declarations.Activations[0].ServerToken)
}
