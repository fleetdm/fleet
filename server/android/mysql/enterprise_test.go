package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHosts(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CreateGetEnterprise", testCreateGetEnterprise},
		{"UpdateEnterprise", testUpdateEnterprise},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testCreateGetEnterprise(t *testing.T, ds *Datastore) {
	_, err := ds.GetEnterpriseByID(testCtx(), 9999)
	assert.True(t, fleet.IsNotFound(err))

	id, err := ds.CreateEnterprise(testCtx())
	require.NoError(t, err)
	assert.NotZero(t, id)

	result, err := ds.GetEnterpriseByID(testCtx(), id)
	require.NoError(t, err)
	assert.Equal(t, &android.Enterprise{ID: id}, result)
}

func testUpdateEnterprise(t *testing.T, ds *Datastore) {
	enterprise := &android.Enterprise{
		ID:           9999, // start with an invalid ID
		SignupName:   "signupUrls/C97372c91c6a85139",
		EnterpriseID: "LC04bp524j",
	}
	err := ds.UpdateEnterprise(testCtx(), enterprise)
	assert.Error(t, err)

	id, err := ds.CreateEnterprise(testCtx())
	require.NoError(t, err)
	assert.NotZero(t, id)

	enterprise.ID = id
	err = ds.UpdateEnterprise(testCtx(), enterprise)
	require.NoError(t, err)

	result, err := ds.GetEnterpriseByID(testCtx(), enterprise.ID)
	require.NoError(t, err)
	assert.Equal(t, enterprise, result)

	enterprises, err := ds.ListEnterprises(testCtx())
	require.NoError(t, err)
	assert.Len(t, enterprises, 1)
	assert.Equal(t, enterprise, enterprises[0])
}
