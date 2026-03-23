package mysql

import "github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"

// testEnv holds test dependencies.
type testEnv struct {
	*testutils.TestDB
	ds *Datastore
}
