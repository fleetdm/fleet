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

func TestAccountOrder(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "acme_account_order")
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

		{"CreateNewOrder", testCreateNewOrder},
		{"OrderCreationLimit", testOrderCreationLimit},
		{"InvalidAccountID", testInvalidAccountID},
		{"MultipleOrdersDifferentAccounts", testMultipleOrdersDifferentAccounts},

		{"GetExistingOrder", testGetExistingOrder},
		{"OrderNotFound", testGetOrderNotFound},
		{"WrongAccountID", testGetOrderWrongAccountID},

		{"ListOrderIDs", testListOrderIDs},
		{"ListOrderIDsExcludesInvalid", testListOrderIDsExcludesInvalid},
		{"ListOrderIDsEmpty", testListOrderIDsEmpty},
		{"ListOrderIDsInvalidAccount", testListOrderIDsInvalidAccount},

		{"GetCertificatePEM", testGetCertificatePEM},
		{"GetCertificatePEMNotFound", testGetCertificatePEMNotFound},
		{"GetCertificatePEMRevoked", testGetCertificatePEMRevoked},
		{"GetCertificatePEMWrongAccount", testGetCertificatePEMWrongAccount},
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
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk1,
	}

	result1, didCreate, err := env.ds.CreateAccount(t.Context(), account1, false)
	require.NoError(t, err)
	require.NotZero(t, result1.ID)
	require.Equal(t, enrollment.ID, result1.ACMEEnrollmentID)
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
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk2,
	}

	result2, didCreate, err := env.ds.CreateAccount(t.Context(), account2, false)
	require.NoError(t, err)
	require.NotZero(t, result2.ID)
	require.Equal(t, enrollment.ID, result2.ACMEEnrollmentID)
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
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
	}
	result1, didCreate, err := env.ds.CreateAccount(t.Context(), account1, false)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.True(t, didCreate)

	// create again with same JWK
	account2 := &types.Account{
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
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
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
	}
	created, didCreate, err := env.ds.CreateAccount(t.Context(), account, false)
	require.NoError(t, err)
	require.True(t, didCreate)

	// now look it up with onlyReturnExisting=true
	lookup := &types.Account{
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
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
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
	}

	result, didCreate, err := env.ds.CreateAccount(t.Context(), account, true)
	require.Nil(t, result)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error:accountDoesNotExist") // nolint:nilaway // cannot be nil due to previous require
	require.False(t, didCreate)
}

func testAccountCreationLimit(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	// create 3 accounts (the max)
	for range maxAccountsPerEnrollment {
		jwk := generateTestJWK(t)
		account := &types.Account{
			ACMEEnrollmentID: enrollment.ID,
			JSONWebKey:       jwk,
		}
		_, _, err := env.ds.CreateAccount(t.Context(), account, false)
		require.NoError(t, err)
	}

	// 4th should fail
	jwk := generateTestJWK(t)
	account := &types.Account{
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
	}
	result, _, err := env.ds.CreateAccount(t.Context(), account, false)
	require.Nil(t, result)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/tooManyAccounts") // nolint:nilaway // cannot be nil due to previous require
}

func testAccountRevoked(t *testing.T, env *testEnv) {
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk := generateTestJWK(t)
	account := &types.Account{
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
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
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
	}
	_, _, err = env.ds.CreateAccount(t.Context(), account2, false)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/accountRevoked") // nolint:nilaway // cannot be nil due to previous require
}

func testInvalidEnrollmentID(t *testing.T, env *testEnv) {
	jwk := generateTestJWK(t)
	account := &types.Account{
		ACMEEnrollmentID: 99999,
		JSONWebKey:       jwk,
	}

	result, _, err := env.ds.CreateAccount(t.Context(), account, false)
	require.Nil(t, result)
	require.Error(t, err)
}

// createTestAccountForOrder is a helper that creates an enrollment and account for order tests.
func createTestAccountForOrder(t *testing.T, env *testEnv) (*types.Account, *types.Enrollment) {
	t.Helper()
	enrollment := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollment)

	jwk := generateTestJWK(t)
	account := &types.Account{
		ACMEEnrollmentID: enrollment.ID,
		JSONWebKey:       jwk,
	}
	created, _, err := env.ds.CreateAccount(t.Context(), account, false)
	require.NoError(t, err)
	return created, enrollment
}

func buildTestOrder(accountID uint, identifierValue string) (*types.Order, *types.Authorization, *types.Challenge) {
	return &types.Order{
			ACMEAccountID: accountID,
			Status:        types.OrderStatusPending,
			Identifiers: []types.Identifier{
				{Type: types.IdentifierTypePermanentIdentifier, Value: identifierValue},
			},
		}, &types.Authorization{
			Identifier: types.Identifier{Type: types.IdentifierTypePermanentIdentifier, Value: identifierValue},
			Status:     types.AuthorizationStatusPending,
		}, &types.Challenge{
			ChallengeType: types.DeviceAttestationChallengeType,
			Token:         "test-token",
			Status:        types.ChallengeStatusPending,
		}
}

func testCreateNewOrder(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	result, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)
	require.NotZero(t, result.ID)
	require.Equal(t, account.ID, result.ACMEAccountID)
	require.Equal(t, types.OrderStatusPending, result.Status)

	// authorization and challenge IDs should be set by CreateOrder
	require.NotZero(t, authorization.ID)
	require.NotZero(t, challenge.ID)
}

func testOrderCreationLimit(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	// create maxOrdersPerAccount orders (the max)
	for range maxOrdersPerAccount {
		order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
		_, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
		require.NoError(t, err)
	}

	// the next order should fail
	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	result, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.Nil(t, result)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/tooManyOrders") // nolint:nilaway // cannot be nil due to previous require
}

func testInvalidAccountID(t *testing.T, env *testEnv) {
	order, authorization, challenge := buildTestOrder(99999, "serial-123")
	result, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.Nil(t, result)
	require.Error(t, err)
}

func testMultipleOrdersDifferentAccounts(t *testing.T, env *testEnv) {
	account1, _ := createTestAccountForOrder(t, env)
	account2, _ := createTestAccountForOrder(t, env)

	// create max orders for account1
	for range maxOrdersPerAccount {
		order, authorization, challenge := buildTestOrder(account1.ID, "serial-123")
		_, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
		require.NoError(t, err)
	}

	// account2 should still be able to create orders independently
	order, authorization, challenge := buildTestOrder(account2.ID, "serial-456")
	result, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)
	require.NotZero(t, result.ID)
	require.Equal(t, account2.ID, result.ACMEAccountID)
}

func testGetExistingOrder(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	gotOrder, gotAuths, err := env.ds.GetOrderByID(t.Context(), account.ID, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, gotOrder.ID)
	require.Equal(t, account.ID, gotOrder.ACMEAccountID)
	require.Equal(t, types.OrderStatusPending, gotOrder.Status)
	require.False(t, gotOrder.Finalized)
	require.Len(t, gotOrder.Identifiers, 1)
	require.Equal(t, types.IdentifierTypePermanentIdentifier, gotOrder.Identifiers[0].Type)
	require.Equal(t, "serial-123", gotOrder.Identifiers[0].Value)

	require.Len(t, gotAuths, 1)
	require.Equal(t, authorization.ID, gotAuths[0].ID)
	require.Equal(t, authorization.Identifier, gotAuths[0].Identifier)
}

func testGetOrderNotFound(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	_, _, err := env.ds.GetOrderByID(t.Context(), account.ID, 99999)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "orderDoesNotExist") // nolint:nilaway // cannot be nil due to previous require
}

func testGetOrderWrongAccountID(t *testing.T, env *testEnv) {
	account1, _ := createTestAccountForOrder(t, env)
	account2, _ := createTestAccountForOrder(t, env)

	order, authorization, challenge := buildTestOrder(account1.ID, "serial-123")
	created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	// try to get the order using account2's ID — should fail
	_, _, err = env.ds.GetOrderByID(t.Context(), account2.ID, created.ID)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "orderDoesNotExist") // nolint:nilaway // cannot be nil due to previous require
}

func testListOrderIDs(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	// create a couple of orders
	var expectedIDs []uint
	for range 2 {
		order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
		created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
		require.NoError(t, err)
		expectedIDs = append(expectedIDs, created.ID)
	}

	ids, err := env.ds.ListAccountOrderIDs(t.Context(), account.ID)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, ids)
}

func testListOrderIDsExcludesInvalid(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	// create two orders
	allIDs := make([]uint, 0, 2)
	for range cap(allIDs) {
		order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
		created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
		require.NoError(t, err)
		allIDs = append(allIDs, created.ID)
	}

	// mark the first order as invalid directly in the DB
	_, err := env.DB.ExecContext(t.Context(), `UPDATE acme_orders SET status = 'invalid' WHERE id = ?`, allIDs[0])
	require.NoError(t, err)

	ids, err := env.ds.ListAccountOrderIDs(t.Context(), account.ID)
	require.NoError(t, err)
	require.Equal(t, []uint{allIDs[1]}, ids)
}

func testListOrderIDsEmpty(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	ids, err := env.ds.ListAccountOrderIDs(t.Context(), account.ID)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func testListOrderIDsInvalidAccount(t *testing.T, env *testEnv) {
	ids, err := env.ds.ListAccountOrderIDs(t.Context(), 99999)
	require.NoError(t, err)
	require.Empty(t, ids)
}

// insertTestCertificate inserts an identity serial and certificate into the database for testing.
func insertTestCertificate(t *testing.T, env *testEnv, serial uint64, certPEM string, revoked bool) {
	t.Helper()
	ctx := t.Context()

	_, err := env.DB.ExecContext(ctx, `INSERT INTO identity_serials (serial) VALUES (?)`, serial)
	require.NoError(t, err)

	_, err = env.DB.ExecContext(ctx, `
		INSERT INTO identity_certificates (serial, not_valid_before, not_valid_after, certificate_pem, revoked)
		VALUES (?, NOW(), DATE_ADD(NOW(), INTERVAL 1 YEAR), ?, ?)
	`, serial, certPEM, revoked)
	require.NoError(t, err)
}

func testGetCertificatePEM(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	// insert a certificate and link it to the order
	var certSerial uint64 = 1001
	expectedPEM := "-----BEGIN CERTIFICATE-----\ntest-cert-pem\n-----END CERTIFICATE-----"
	insertTestCertificate(t, env, certSerial, expectedPEM, false)

	_, err = env.DB.ExecContext(t.Context(), `UPDATE acme_orders SET issued_certificate_serial = ? WHERE id = ?`, certSerial, created.ID)
	require.NoError(t, err)

	gotPEM, err := env.ds.GetCertificatePEMByOrderID(t.Context(), account.ID, created.ID)
	require.NoError(t, err)
	require.Equal(t, expectedPEM, gotPEM)
}

func testGetCertificatePEMNotFound(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	// create an order without linking a certificate
	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	_, err = env.ds.GetCertificatePEMByOrderID(t.Context(), account.ID, created.ID)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "certificateDoesNotExist") // nolint:nilaway // cannot be nil due to previous require
}

func testGetCertificatePEMRevoked(t *testing.T, env *testEnv) {
	account, _ := createTestAccountForOrder(t, env)

	order, authorization, challenge := buildTestOrder(account.ID, "serial-123")
	created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	// insert a revoked certificate and link it to the order
	var certSerial uint64 = 2001
	insertTestCertificate(t, env, certSerial, "-----BEGIN CERTIFICATE-----\nrevoked\n-----END CERTIFICATE-----", true)

	_, err = env.DB.ExecContext(t.Context(), `UPDATE acme_orders SET issued_certificate_serial = ? WHERE id = ?`, certSerial, created.ID)
	require.NoError(t, err)

	_, err = env.ds.GetCertificatePEMByOrderID(t.Context(), account.ID, created.ID)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "certificateDoesNotExist") // nolint:nilaway // cannot be nil due to previous require
}

func testGetCertificatePEMWrongAccount(t *testing.T, env *testEnv) {
	account1, _ := createTestAccountForOrder(t, env)
	account2, _ := createTestAccountForOrder(t, env)

	order, authorization, challenge := buildTestOrder(account1.ID, "serial-123")
	created, err := env.ds.CreateOrder(t.Context(), order, authorization, challenge)
	require.NoError(t, err)

	// insert a certificate and link it to account1's order
	var certSerial uint64 = 3001
	insertTestCertificate(t, env, certSerial, "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----", false)

	_, err = env.DB.ExecContext(t.Context(), `UPDATE acme_orders SET issued_certificate_serial = ? WHERE id = ?`, certSerial, created.ID)
	require.NoError(t, err)

	// try to get the certificate using account2's ID — should fail
	_, err = env.ds.GetCertificatePEMByOrderID(t.Context(), account2.ID, created.ID)
	require.Error(t, err)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "certificateDoesNotExist") // nolint:nilaway // cannot be nil due to previous require
}
