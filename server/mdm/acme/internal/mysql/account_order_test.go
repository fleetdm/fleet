package mysql

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/stretchr/testify/require"
	"go.step.sm/crypto/jose"
)

func generateTestJWK(t *testing.T) jose.JSONWebKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return jose.JSONWebKey{Key: key.Public()}
}

func TestCreateAccount(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "acme_create_account")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"CreateNewAccount", testCreateNewAccount},
		{"ReturnExistingSameJWK", testReturnExistingSameJWK},
		{"OnlyReturnExistingFound", testOnlyReturnExistingFound},
		{"OnlyReturnExistingNotFound", testOnlyReturnExistingNotFound},
		{"AccountCreationLimit", testAccountCreationLimit},
		{"AccountRevoked", testAccountRevoked},
		{"InvalidEnrollmentID", testInvalidEnrollmentID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

func testCreateNewAccount(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk1 := generateTestJWK(t)
	account1 := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk1,
	}

	result1, didCreate, err := env.ds.CreateAccount(t.Context(), account1, false)
	require.NoError(t, err)
	require.NotZero(t, result1.ID)
	require.Equal(t, enrollment.ID, result1.EnrollmentID)
	require.True(t, didCreate)

	// verify enrollment's not_valid_after was set
	updatedEnrollment1, err := env.ds.GetACMEEnrollment(t.Context(), enrollment.PathIdentifier)
	require.NoError(t, err)
	require.NotNil(t, updatedEnrollment1.NotValidAfter)
	require.True(t, updatedEnrollment1.NotValidAfter.After(time.Now()))

	time.Sleep(time.Second) // ensure different timestamp

	// create another account
	jwk2 := generateTestJWK(t)
	account2 := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk2,
	}

	result2, didCreate, err := env.ds.CreateAccount(t.Context(), account2, false)
	require.NoError(t, err)
	require.NotZero(t, result2.ID)
	require.Equal(t, enrollment.ID, result2.EnrollmentID)
	require.NotEqual(t, result1.ID, result2.ID)
	require.True(t, didCreate)

	// verify enrollment's not_valid_after was not updated as it was already set
	updatedEnrollment2, err := env.ds.GetACMEEnrollment(t.Context(), enrollment.PathIdentifier)
	require.NoError(t, err)
	require.NotNil(t, updatedEnrollment1.NotValidAfter)
	require.True(t, updatedEnrollment1.NotValidAfter.After(time.Now()))
	require.True(t, updatedEnrollment1.NotValidAfter.Equal(*updatedEnrollment2.NotValidAfter))
}

func testReturnExistingSameJWK(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk := generateTestJWK(t)

	account1 := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}
	result1, didCreate, err := env.ds.CreateAccount(t.Context(), account1, false)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.True(t, didCreate)

	// create again with same JWK
	account2 := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}
	result2, didCreate, err := env.ds.CreateAccount(t.Context(), account2, false)
	require.NoError(t, err)
	require.NotNil(t, result2)
	require.Equal(t, result1.ID, result2.ID)
	require.False(t, didCreate)
}

func testOnlyReturnExistingFound(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk := generateTestJWK(t)

	// create the account first
	account := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}
	created, didCreate, err := env.ds.CreateAccount(t.Context(), account, false)
	require.NoError(t, err)
	require.True(t, didCreate)

	// now look it up with onlyReturnExisting=true
	lookup := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}
	found, didCreate, err := env.ds.CreateAccount(t.Context(), lookup, true)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, created.ID, found.ID)
	require.False(t, didCreate)
}

func testOnlyReturnExistingNotFound(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk := generateTestJWK(t)
	account := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}

	result, didCreate, err := env.ds.CreateAccount(t.Context(), account, true)
	require.Nil(t, result)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error:accountDoesNotExist")
	require.False(t, didCreate)
}

func testAccountCreationLimit(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	// create 3 accounts (the max)
	for range maxAccountsPerEnrollment {
		jwk := generateTestJWK(t)
		account := &types.Account{
			EnrollmentID: enrollment.ID,
			JSONWebKey:   jwk,
		}
		_, _, err := env.ds.CreateAccount(t.Context(), account, false)
		require.NoError(t, err)
	}

	// 4th should fail
	jwk := generateTestJWK(t)
	account := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}
	result, _, err := env.ds.CreateAccount(t.Context(), account, false)
	require.Nil(t, result)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/tooManyAccounts")
}

func testAccountRevoked(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk := generateTestJWK(t)
	account := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}

	created, didCreate, err := env.ds.CreateAccount(t.Context(), account, false)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.True(t, didCreate)

	// revoke the account directly in the DB
	_, err = env.DB.ExecContext(t.Context(), `UPDATE acme_accounts SET revoked = 1 WHERE id = ?`, created.ID)
	require.NoError(t, err)

	// try to create again with the same JWK — should get accountRevoked error
	account2 := &types.Account{
		EnrollmentID: enrollment.ID,
		JSONWebKey:   jwk,
	}
	_, _, err = env.ds.CreateAccount(t.Context(), account2, false)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/accountRevoked")
}

func testInvalidEnrollmentID(t *testing.T, env *testEnv) {
	jwk := generateTestJWK(t)
	account := &types.Account{
		EnrollmentID: 99999,
		JSONWebKey:   jwk,
	}

	result, _, err := env.ds.CreateAccount(t.Context(), account, false)
	require.Nil(t, result)
	require.Error(t, err)
}
