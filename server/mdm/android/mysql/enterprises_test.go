package mysql_test

import (
	"context"
	"testing"

	ctu "github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
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
		{"DeleteEnterprises", testDeleteEnterprises},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer ctu.TruncateTables(t, ds.Writer(testCtx()), ds.Logger(), nil)

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

	result, err = ds.GetEnterprise(testCtx())
	require.NoError(t, err)
	assert.Equal(t, enterprise.ID, result.ID)
	assert.Equal(t, enterprise.EnterpriseID, result.EnterpriseID)
}

func testDeleteEnterprises(t *testing.T, ds *mysql.Datastore) {
	err := ds.DeleteEnterprises(testCtx())
	require.NoError(t, err)
	err = ds.DeleteOtherEnterprises(testCtx(), 9999)
	require.NoError(t, err)

	enterprise := createEnterprise(t, ds)
	result, err := ds.GetEnterpriseByID(testCtx(), enterprise.ID)
	require.NoError(t, err)
	assert.Equal(t, enterprise, result)

	// Create enteprise without enterprise_id
	id, err := ds.CreateEnterprise(testCtx())
	require.NoError(t, err)
	assert.NotZero(t, id)

	tempEnterprise := &android.Enterprise{
		ID:           id, // start with an invalid ID
		SignupName:   "signupUrls/C97372c91c6a85139",
		EnterpriseID: "",
	}
	err = ds.UpdateEnterprise(testCtx(), tempEnterprise)
	require.NoError(t, err)

	err = ds.DeleteOtherEnterprises(testCtx(), enterprise.ID)
	require.NoError(t, err)
	result, err = ds.GetEnterpriseByID(testCtx(), enterprise.ID)
	require.NoError(t, err)
	assert.Equal(t, enterprise, result)
	_, err = ds.GetEnterpriseByID(testCtx(), tempEnterprise.ID)
	assert.True(t, fleet.IsNotFound(err))

	err = ds.DeleteEnterprises(testCtx())
	require.NoError(t, err)
	_, err = ds.GetEnterpriseByID(testCtx(), enterprise.ID)
	assert.True(t, fleet.IsNotFound(err))

}

func createEnterprise(t *testing.T, ds *mysql.Datastore) *android.Enterprise {
	enterprise := &android.Enterprise{
		ID:           9999, // start with an invalid ID
		SignupName:   "signupUrls/C97372c91c6a85139",
		EnterpriseID: "LC04bp524j",
	}
	id, err := ds.CreateEnterprise(testCtx())
	require.NoError(t, err)
	assert.NotZero(t, id)

	enterprise.ID = id
	err = ds.UpdateEnterprise(testCtx(), enterprise)
	require.NoError(t, err)
	return enterprise
}

func testCtx() context.Context {
	return context.Background()
}
