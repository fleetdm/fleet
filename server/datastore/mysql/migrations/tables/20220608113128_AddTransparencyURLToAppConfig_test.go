package tables

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20220608113128(t *testing.T) {
	db := applyUpToPrev(t)

	var prevRaw []byte
	var prevConfig fleet.AppConfig
	err := db.Get(&prevRaw, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)

	err = json.Unmarshal(prevRaw, &prevConfig)
	require.NoError(t, err)
	require.Empty(t, prevConfig.FleetDesktop.TransparencyURL)

	applyNext(t, db)

	var newRaw []byte
	var newConfig fleet.AppConfig
	err = db.Get(&newRaw, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)

	err = json.Unmarshal(newRaw, &newConfig)
	require.NoError(t, err)
	require.Equal(t, "", newConfig.FleetDesktop.TransparencyURL)
}
