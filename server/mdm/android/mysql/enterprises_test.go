package mysql_test

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mysql/testing_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnterprise(t *testing.T) {
	ds := testing_utils.CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *mysql.Datastore)
	}{
		{"CreateGetEnterprise", testCreateGetEnterprise},
		{"UpdateEnterprise", testUpdateEnterprise},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer testing_utils.TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testCreateGetEnterprise(t *testing.T, ds *mysql.Datastore) {
	_, err := ds.GetEnterpriseByID(testCtx(), 9999)
	assert.True(t, fleet.IsNotFound(err))

	id, err := ds.CreateEnterprise(testCtx())
	require.NoError(t, err)
	assert.NotZero(t, id)

	result, err := ds.GetEnterpriseByID(testCtx(), id)
	require.NoError(t, err)
	assert.Equal(t, &android.Enterprise{ID: id}, result)
}

func testUpdateEnterprise(t *testing.T, ds *mysql.Datastore) {
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

func testCtx() context.Context {
	return context.Background()
}
