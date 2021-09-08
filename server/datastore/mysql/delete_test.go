package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteEntity(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	host, err := ds.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "1",
		UUID:            t.Name() + "1",
		OsqueryHostID:   t.Name(),
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	require.NoError(t, ds.deleteEntity("hosts", host.ID))

	host, err = ds.Host(host.ID)
	require.Error(t, err)
	assert.Nil(t, host)
}

func TestDeleteEntityByName(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	query1 := test.NewQuery(t, ds, t.Name()+"time", "select * from time", 0, true)

	require.NoError(t, ds.deleteEntityByName("queries", query1.Name))

	gotQ, err := ds.Query(context.Background(), query1.ID)
	require.Error(t, err)
	assert.Nil(t, gotQ)
}

func TestDeleteEntities(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	query1 := test.NewQuery(t, ds, t.Name()+"time1", "select * from time", 0, true)
	query2 := test.NewQuery(t, ds, t.Name()+"time2", "select * from time", 0, true)
	query3 := test.NewQuery(t, ds, t.Name()+"time3", "select * from time", 0, true)

	count, err := ds.deleteEntities("queries", []uint{query1.ID, query2.ID})
	require.NoError(t, err)
	assert.Equal(t, uint(2), count)

	gotQ, err := ds.Query(context.Background(), query1.ID)
	require.Error(t, err)
	assert.Nil(t, gotQ)

	gotQ, err = ds.Query(context.Background(), query2.ID)
	require.Error(t, err)
	assert.Nil(t, gotQ)

	gotQ, err = ds.Query(context.Background(), query3.ID)
	require.NoError(t, err)
	assert.Equal(t, query3.ID, gotQ.ID)
}
