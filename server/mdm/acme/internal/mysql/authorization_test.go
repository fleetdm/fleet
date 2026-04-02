package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/stretchr/testify/require"
)

func TestACMEAuthorization(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "acme_authorization")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"GetValidAuthorization", testGetValidAuthorization},
		{"GetAuthorizationWithInvalidID", testGetAuthorizationWithInvalidID},
		{"GetAuthorizationWithInvalidAccountID", testGetAuthorizationWithInvalidAccountID},
		{"GetAuthorizationWithInvalidInputs", testGetAuthorizationWithInvalidInputs},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

func testGetValidAuthorization(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	order, authorization, _ := createTestOrderForAccount(t, account, env)

	authResp, err := env.ds.GetAuthorizationByID(t.Context(), account.ID, authorization.ID)
	require.NoError(t, err)
	require.NotNil(t, authResp)
	require.Equal(t, types.AuthorizationStatusPending, authResp.Status)
	require.Equal(t, "permanent-identifier", authResp.Identifier.Type)
	require.Equal(t, "serial-123", authResp.Identifier.Value)
	require.Equal(t, order.ID, authResp.ACMEOrderID)
	require.Equal(t, authorization.ID, authResp.ID)
}

func testGetAuthorizationWithInvalidID(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	authResp, err := env.ds.GetAuthorizationByID(t.Context(), account.ID, 9999) // non-existent ID
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/authorizationDoesNotExist") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, authResp)
}

func testGetAuthorizationWithInvalidAccountID(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	_, authorization, _ := createTestOrderForAccount(t, account, env)

	authResp, err := env.ds.GetAuthorizationByID(t.Context(), 9999, authorization.ID) // non-existent account ID
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/authorizationDoesNotExist") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, authResp)
}

func testGetAuthorizationWithInvalidInputs(t *testing.T, env *testEnv) {
	authResp, err := env.ds.GetAuthorizationByID(t.Context(), 0, 0) // zero account ID
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "malformed") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, authResp)

	authResp, err = env.ds.GetAuthorizationByID(t.Context(), 1, 0) // zero authorization ID
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "malformed") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, authResp)
}

func createTestOrderForAccount(t *testing.T, account *types.Account, env *testEnv) (*types.Order, *types.Authorization, *types.Challenge) {
	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	result, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	return result, authorization, challenge
}
