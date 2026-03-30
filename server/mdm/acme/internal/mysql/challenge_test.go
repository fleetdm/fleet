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
		{"GetChallengesWithInvalidAuthorizationID", testGetChallengesWithInvalidAuthorizationID},
		{"GetChallengesWithZeroAuthorizationID", testGetChallengesWithZeroAuthorizationID},
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

func testGetChallengesWithInvalidAuthorizationID(t *testing.T, env *testEnv) {
	challenges, err := env.ds.GetChallengesByAuthorizationID(t.Context(), 999999) // non-existent ID
	require.Error(t, err)
	require.Nil(t, challenges)
}

func testGetChallengesWithZeroAuthorizationID(t *testing.T, env *testEnv) {
	challenges, err := env.ds.GetChallengesByAuthorizationID(t.Context(), 0)
	require.Error(t, err)
	require.Nil(t, challenges)
}
