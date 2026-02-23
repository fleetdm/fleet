package tables

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20251209221730(t *testing.T) {
	db := applyUpToPrev(t)

	// Setup AppConfig
	var prevRaw []byte
	var appConfig1 fleet.AppConfig
	err := db.Get(&prevRaw, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)

	appConfig1.MDM.MacOSUpdates.Deadline = optjson.SetString("2025-01-01")
	appConfig1.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("14.0.0")

	b1, err := json.Marshal(appConfig1)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE app_config_json SET json_value = ?`, b1)
	require.NoError(t, err)

	// Setup Teams
	// Team 1: Configured
	teamConfig1 := fleet.TeamConfig{}
	teamConfig1.MDM.MacOSUpdates.Deadline = optjson.SetString("2025-01-01")
	teamConfig1.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("14.0.0")
	bT1, err := json.Marshal(teamConfig1)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teams (id, name, config) VALUES (1, 'team1', ?)`, bT1)
	require.NoError(t, err)

	// Team 2: Not Configured (missing deadline)
	teamConfig2 := fleet.TeamConfig{}
	teamConfig2.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("14.0.0")
	bT2, err := json.Marshal(teamConfig2)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teams (id, name, config) VALUES (2, 'team2', ?)`, bT2)
	require.NoError(t, err)

	// Team 3: Not Configured (missing min version)
	teamConfig3 := fleet.TeamConfig{}
	teamConfig3.MDM.MacOSUpdates.Deadline = optjson.SetString("2025-01-01")
	bT3, err := json.Marshal(teamConfig3)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teams (id, name, config) VALUES (3, 'team3', ?)`, bT3)
	require.NoError(t, err)

	// Team 4: Explicitly false but Configured (should become true)
	teamConfig4 := fleet.TeamConfig{}
	teamConfig4.MDM.MacOSUpdates.Deadline = optjson.SetString("2025-01-01")
	teamConfig4.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("14.0.0")
	teamConfig4.MDM.MacOSUpdates.UpdateNewHosts = optjson.Bool{Value: false}
	bT4, err := json.Marshal(teamConfig4)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO teams (id, name, config) VALUES (4, 'team4', ?)`, bT4)
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
	require.True(t, finalAppConfig.MDM.MacOSUpdates.UpdateNewHosts.Set)
	require.True(t, finalAppConfig.MDM.MacOSUpdates.UpdateNewHosts.Value)

	// Verify Teams
	// Team 1: Configured -> True
	var rawTeamConfig1 []byte
	err = db.QueryRow(`SELECT config FROM teams WHERE id = 1`).Scan(&rawTeamConfig1)
	require.NoError(t, err)
	var finalTeamConfig1 fleet.TeamConfig
	err = json.Unmarshal(rawTeamConfig1, &finalTeamConfig1)
	require.NoError(t, err)
	require.True(t, finalTeamConfig1.MDM.MacOSUpdates.UpdateNewHosts.Value)

	// Team 2: Missing deadline -> False
	var rawTeamConfig2 []byte
	err = db.QueryRow(`SELECT config FROM teams WHERE id = 2`).Scan(&rawTeamConfig2)
	require.NoError(t, err)
	var finalTeamConfig2 fleet.TeamConfig
	err = json.Unmarshal(rawTeamConfig2, &finalTeamConfig2)
	require.NoError(t, err)
	require.False(t, finalTeamConfig2.MDM.MacOSUpdates.UpdateNewHosts.Value)

	// Team 3: Missing minimum version -> False
	var rawTeamConfig3 []byte
	err = db.QueryRow(`SELECT config FROM teams WHERE id = 3`).Scan(&rawTeamConfig3)
	require.NoError(t, err)
	var finalTeamConfig3 fleet.TeamConfig
	err = json.Unmarshal(rawTeamConfig3, &finalTeamConfig3)
	require.NoError(t, err)
	require.False(t, finalTeamConfig3.MDM.MacOSUpdates.UpdateNewHosts.Value)

	// Team 4: Was false, but configured -> True
	var rawTeamConfig4 []byte
	err = db.QueryRow(`SELECT config FROM teams WHERE id = 4`).Scan(&rawTeamConfig4)
	require.NoError(t, err)
	var finalTeamConfig4 fleet.TeamConfig
	err = json.Unmarshal(rawTeamConfig4, &finalTeamConfig4)
	require.NoError(t, err)
	require.True(t, finalTeamConfig4.MDM.MacOSUpdates.UpdateNewHosts.Value)
}
