package fleet_test

import (
	"testing"
	"time"

	chartapi "github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestMobileOnlineWindowMatchesChart guards against drift between
// fleet.MobileOnlineWindow (used by Host.mobileStatus and the mysql predicates)
// and chartapi.MobileOnlineWindowSeconds (used by the chart bounded context's
// FindOnlineHostIDs). The two live in different packages because server/chart
// can't import server/fleet; this test is the only automated sync check.
func TestMobileOnlineWindowMatchesChart(t *testing.T) {
	require.Equal(t,
		int(fleet.MobileOnlineWindow/time.Second),
		chartapi.MobileOnlineWindowSeconds,
		"fleet.MobileOnlineWindow and chartapi.MobileOnlineWindowSeconds drifted; edit both together",
	)
}
