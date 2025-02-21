package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAndroid(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewAndroidHost", testNewAndroidHost},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testNewAndroidHost(t *testing.T, ds *Datastore) {
	const deviceID = "device_id"
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			Hostname:        "hostname",
			ComputerName:    "computer_name",
			Platform:        "android",
			OSVersion:       "Android 14",
			Build:           "build",
			Memory:          1024,
			TeamID:          nil,
			HardwareSerial:  "hardware_serial",
		},
		Device: &android.Device{
			EnterpriseSpecificID: nil,
			PolicyID:             ptr.Uint(1),
			LastPolicySyncTime:   ptr.Time(time.Time{}),
		},
	}
	host.SetDeviceID(deviceID)

	result, err := ds.NewAndroidHost(testCtx(), host)
	require.NoError(t, err)
	assert.NotZero(t, result.Host.ID)
	assert.NotZero(t, result.Device.ID)

	resultLite, err := ds.AndroidHostLite(testCtx(), deviceID)
	require.NoError(t, err)
	assert.Equal(t, result.Host.ID, resultLite.Host.ID)
	assert.Equal(t, result.Device.ID, resultLite.Device.ID)

	// Inserting the same host again should be fine.
	// This may occur when 2 Fleet servers received the same host information via pubsub.
	resultCopy, err := ds.NewAndroidHost(testCtx(), host)
	require.NoError(t, err)
	assert.Equal(t, result.Host.ID, resultCopy.Host.ID)
	assert.Equal(t, result.Device.ID, resultCopy.Device.ID)
}

func testCtx() context.Context {
	return context.Background()
}
