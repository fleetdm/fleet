package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestInHouseApps(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestInHouseAppsCrud", testInHouseAppsCrud},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testInHouseAppsCrud(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	_, err := ds.InsertInHouseApp(ctx, &fleet.InHouseAppPayload{
		Name:      "foo",
		StorageID: "testingtesting123",
		Platform:  "ios",
	})

	require.NoError(t, err)
}
