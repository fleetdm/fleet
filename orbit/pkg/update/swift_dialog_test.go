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
	var f OrbitConfigFetcher = &dummyConfigFetcher{cfg: cfg}
	f = ApplySwiftDialogDownloaderMiddleware(f, nil)

	// we used to get a panic if updates were disabled (see #11980)
	gotCfg, err := f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
}
