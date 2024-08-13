package gdmf

import (
	"testing"

	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/stretchr/testify/require"
)

func TestGetLatest(t *testing.T) {
	d := &apple_mdm.MachineInfo{
		SoftwareUpdateDeviceID: "X589AMLUAP",
		Product:                "MacBookPro16,1",
	}
	res, err := GetLatestOSVersion(*d)
	require.NoError(t, err)
	require.Equal(t, struct{}{}, res)
}
