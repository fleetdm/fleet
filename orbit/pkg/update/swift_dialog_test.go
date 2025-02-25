package update

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestSwiftDialogUpdatesDisabled(t *testing.T) {
	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.NeedsMDMMigration = true
	cfg.Notifications.RenewEnrollmentProfile = true
	r := ApplySwiftDialogDownloaderMiddleware(nil)

	// we used to get a panic if updates were disabled (see #11980)
	err := r.Run(cfg)
	require.NoError(t, err)
}
