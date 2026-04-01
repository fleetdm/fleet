package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/stretchr/testify/require"
)

func TestACMEChallenge(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "acme_challenge")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"GetValidChallengesForAuthorization", testGetValidChallengesForAuthorization},
		{"NoChallengesForAuthorization", testGetChallengesWithNoChallengesForAuthorization},
		{"GetChallengesWithInvalidAuthorizationID", testGetChallengesWithInvalidAuthorizationID},
		{"GetChallengesWithZeroAuthorizationID", testGetChallengesWithZeroAuthorizationID},

		{"GetChallengeByIDWithValidID", testGetChallengeByIDWithValidID},
		{"GetChallengeByIDWithInvalidID", testGetChallengeByIDWithInvalidID},
		{"GetChallengeByIDWithInvalidAccountID", testGetChallengeByIDWithInvalidAccountID},

		{"UpdateChallengeHappyPath", testUpdateChallengeHappyPath},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

func testGetValidChallengesForAuthorization(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	_, authorization, _ := createTestOrderForAccount(t, account, env)

	challenges, err := env.ds.GetChallengesByAuthorizationID(t.Context(), authorization.ID)
	require.NoError(t, err)
	require.Len(t, challenges, 1)
	require.Equal(t, "pending", challenges[0].Status)
	require.Equal(t, types.DeviceAttestationChallengeType, challenges[0].ChallengeType)
	require.Equal(t, authorization.ID, challenges[0].ACMEAuthorizationID)
}

func testGetChallengesWithNoChallengesForAuthorization(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	_, authorization, _ := createTestOrderForAccount(t, account, env)

	// Delete the challenge to simulate no challenges for the authorization
	_, err := env.TestDB.DB.ExecContext(t.Context(), "DELETE FROM acme_challenges WHERE acme_authorization_id = ?", authorization.ID)
	require.NoError(t, err)

	challenges, err := env.ds.GetChallengesByAuthorizationID(t.Context(), authorization.ID)
	var acmeError *types.ACMEError
	require.ErrorAs(t, err, &acmeError)
	require.Contains(t, acmeError.Type, "error/challengeDoesNotExist") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, challenges)
}

func testGetChallengesWithInvalidAuthorizationID(t *testing.T, env *testEnv) {
	challenges, err := env.ds.GetChallengesByAuthorizationID(t.Context(), 999999) // non-existent ID
	var acmeError *types.ACMEError
	require.ErrorAs(t, err, &acmeError)
	require.Contains(t, acmeError.Type, "error/challengeDoesNotExist") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, challenges)
}

func testGetChallengesWithZeroAuthorizationID(t *testing.T, env *testEnv) {
	challenges, err := env.ds.GetChallengesByAuthorizationID(t.Context(), 0)
	var acmeError *types.ACMEError
	require.ErrorAs(t, err, &acmeError)
	require.Contains(t, acmeError.Type, "malformed") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, challenges)
}

func testGetChallengeByIDWithValidID(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	_, authorization, challenge := createTestOrderForAccount(t, account, env)

	challenge, err := env.ds.GetChallengeByID(t.Context(), account.ID, challenge.ID)
	require.NoError(t, err)
	require.NotNil(t, challenge)
	require.Equal(t, "pending", challenge.Status)
	require.Equal(t, types.DeviceAttestationChallengeType, challenge.ChallengeType)
	require.Equal(t, authorization.ID, challenge.ACMEAuthorizationID)
}

func testGetChallengeByIDWithInvalidID(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	challenge, err := env.ds.GetChallengeByID(t.Context(), account.ID, 9999) // non-existent ID
	var acmeError *types.ACMEError
	require.ErrorAs(t, err, &acmeError)
	require.Contains(t, acmeError.Type, "error/challengeDoesNotExist") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, challenge)
}

func testGetChallengeByIDWithInvalidAccountID(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	_, _, challenge := createTestOrderForAccount(t, account, env)

	challenge, err := env.ds.GetChallengeByID(t.Context(), 9999, challenge.ID) // non-existent account ID
	var acmeError *types.ACMEError
	require.ErrorAs(t, err, &acmeError)
	require.Contains(t, acmeError.Type, "error/challengeDoesNotExist") //nolint:nilaway // cannot be null due to previous require
	require.Nil(t, challenge)
}

func testUpdateChallengeHappyPath(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)
	order, _, challenge := createTestOrderForAccount(t, account, env)

	challenge.Status = "valid"
	updatedChallenge, err := env.ds.UpdateChallenge(t.Context(), challenge)
	require.NoError(t, err)
	require.NotNil(t, updatedChallenge)
	require.Equal(t, "valid", updatedChallenge.Status)
	require.Equal(t, challenge.ID, updatedChallenge.ID)

	order, auhtz, err := env.ds.GetOrderByID(t.Context(), account.ID, order.ID)
	require.NoError(t, err)
	require.NotNil(t, auhtz)
	for _, auth := range auhtz {
		require.Equal(t, "valid", auth.Status)
	}

	require.Equal(t, "ready", order.Status)
}
