package mysql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestAndroidAppConfigs(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestAddDeleteAndroidAppConfig", testAndroidAppConfigCrud},
		// {"TestAddAppWithConfig", testAddAppWithAndroidConfig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testAndroidAppConfigCrud(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	ds.upsertAndroidAppConfigurationTx(ctx, tx, &team.ID, "adamid", json.RawMessage{})

}
