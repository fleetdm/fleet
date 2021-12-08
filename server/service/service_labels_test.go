package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestGetLabels(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	require.NoError(t, ds.MigrateTables(context.Background()))
	require.NoError(t, ds.MigrateData(context.Background()))

	svc := newTestService(ds, nil, nil)

	labels, err := svc.ListLabels(test.UserContext(test.UserAdmin), fleet.ListOptions{Page: 0, PerPage: 1000})
	require.NoError(t, err)
	require.Len(t, labels, 7)
}
