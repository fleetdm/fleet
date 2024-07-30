package update

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func mockConfigFn() func() error {
	return nil
}

func TestDiskEncryptionMiddlewareNotificationOff(t *testing.T) {
	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.RotateDiskEncryptionKey = true
	r := ApplyDiskEncryptionRunnerMiddleware(nil, 1*time.Second)
	err := r.Run(cfg)
	require.NoError(t, err)
}

func TestDiskEncryptionUpdatesDisabled(t *testing.T) {
	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.RotateDiskEncryptionKey = true
	r := ApplyDiskEncryptionRunnerMiddleware(nil, 1*time.Second)
	err := r.Run(cfg)
	require.NoError(t, err)
}
