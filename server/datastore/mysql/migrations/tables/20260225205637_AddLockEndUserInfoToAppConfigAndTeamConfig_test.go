package tables

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20260225205637(t *testing.T) {
	db := applyUpToPrev(t)

	// Setup AppConfig
	var previousAppConfigJSON []byte
	var appConfig fleet.AppConfig
	err := db.Get(&previousAppConfigJSON, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)
	err = json.Unmarshal(previousAppConfigJSON, &appConfig)
	require.NoError(t, err)

	appConfig.MDM.MacOSSetup.EnableEndUserAuthentication = true

	newAppConfigJSON, err := json.Marshal(appConfig)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE app_config_json SET json_value = ?`, newAppConfigJSON)
	require.NoError(t, err)

	// Setup Teams
	// Team 1: Configured
	teamConfig1 := fleet.TeamConfig{}
	teamConfig1.MDM.MacOSSetup.EnableEndUserAuthentication = true
	teamConfig1JSON, err := json.Marshal(teamConfig1)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teams (id, name, config) VALUES (1, 'team1', ?)`, teamConfig1JSON)
	require.NoError(t, err)

	// Team 2: Not Configured (missing deadline)
	teamConfig2 := fleet.TeamConfig{}
	teamConfig2.MDM.MacOSSetup.EnableEndUserAuthentication = false
	teamConfig2JSON, err := json.Marshal(teamConfig2)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teams (id, name, config) VALUES (2, 'team2', ?)`, teamConfig2JSON)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Verify AppConfig
	var rawAppConfig []byte
	err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&rawAppConfig)
	require.NoError(t, err)
	var finalAppConfig fleet.AppConfig
	err = json.Unmarshal(rawAppConfig, &finalAppConfig)
	require.NoError(t, err)
	require.True(t, finalAppConfig.MDM.MacOSSetup.LockEndUserInfo.Set)
	require.True(t, finalAppConfig.MDM.MacOSSetup.LockEndUserInfo.Value)

	// Verify Teams
	// Team 1: EUA enabled -> true
	var rawTeamConfig1 []byte
	err = db.QueryRow(`SELECT config FROM teams WHERE id = 1`).Scan(&rawTeamConfig1)
	require.NoError(t, err)
	var finalTeamConfig1 fleet.TeamConfig
	err = json.Unmarshal(rawTeamConfig1, &finalTeamConfig1)
	require.NoError(t, err)
	require.True(t, finalTeamConfig1.MDM.MacOSSetup.LockEndUserInfo.Set)
	require.True(t, finalTeamConfig1.MDM.MacOSSetup.LockEndUserInfo.Value)

	// Team 2: EUA disabled -> False
	var rawTeamConfig2 []byte
	err = db.QueryRow(`SELECT config FROM teams WHERE id = 2`).Scan(&rawTeamConfig2)
	require.NoError(t, err)
	var finalTeamConfig2 fleet.TeamConfig
	err = json.Unmarshal(rawTeamConfig2, &finalTeamConfig2)
	require.NoError(t, err)
	require.True(t, finalTeamConfig2.MDM.MacOSSetup.LockEndUserInfo.Set)
	require.False(t, finalTeamConfig2.MDM.MacOSSetup.LockEndUserInfo.Value)
}

func TestUp_20260225205637_AppConfigEUADisabled(t *testing.T) {
	db := applyUpToPrev(t)

	// Setup AppConfig
	var previousAppConfigJSON []byte
	var appConfig fleet.AppConfig
	err := db.Get(&previousAppConfigJSON, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)
	err = json.Unmarshal(previousAppConfigJSON, &appConfig)
	require.NoError(t, err)

	appConfig.MDM.MacOSSetup.EnableEndUserAuthentication = false

	newAppConfigJSON, err := json.Marshal(appConfig)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE app_config_json SET json_value = ?`, newAppConfigJSON)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Verify AppConfig
	var rawAppConfig []byte
	err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&rawAppConfig)
	require.NoError(t, err)
	var finalAppConfig fleet.AppConfig
	err = json.Unmarshal(rawAppConfig, &finalAppConfig)
	require.NoError(t, err)
	require.True(t, finalAppConfig.MDM.MacOSSetup.LockEndUserInfo.Set)
	require.False(t, finalAppConfig.MDM.MacOSSetup.LockEndUserInfo.Value)
}
