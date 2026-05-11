package tests

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testhelpers"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	s := setupIntegrationTest(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, s *integrationTestSuite)
	}{
		{"NewNonce", testNewNonce},
		{"GetDirectory", testGetDirectory},
		{"CreateAccount", testCreateAccount},
		{"CreateOrder", testCreateOrder},
		{"GetOrder", testGetOrder},
		{"ListAccountOrders", testListAccountOrders},
		{"GetCertificate", testGetCertificate},
		{"GetAuthorization", testGetAuthorization},
		{"FinalizeOrder", testFinalizeOrder},
		{"DoChallengeDeviceAttestation", testDoChallengeDeviceAttestation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer s.truncateTables(t)
			c.fn(t, s)
		})
	}
}

func testNewNonce(t *testing.T, s *integrationTestSuite) {
	// create a valid enrollment
	enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollValid)

	// create a revoked enrollment
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	// create an expired enrollment
	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	cases := []struct {
		desc       string
		method     string
		identifier string
		wantStatus int
		wantNonce  bool
	}{
		{
			"GET with unknown identifier",
			http.MethodGet,
			"no-such-identifier",
			http.StatusNotFound,
			false,
		},
		{
			"HEAD with unknown identifier",
			http.MethodHead,
			"no-such-identifier",
			http.StatusNotFound,
			false,
		},
		{
			"POST with unknown identifier",
			http.MethodPost,
			"no-such-identifier",
			http.StatusNotFound,
			false,
		},
		{
			"GET with valid identifier",
			http.MethodGet,
			enrollValid.PathIdentifier,
			http.StatusNoContent,
			true,
		},
		{
			"HEAD with valid identifier",
			http.MethodHead,
			enrollValid.PathIdentifier,
			http.StatusOK,
			true,
		},
		{
			"POST with valid identifier",
			http.MethodPost,
			enrollValid.PathIdentifier,
			http.StatusNoContent,
			true,
		},
		{
			"GET with revoked identifier",
			http.MethodGet,
			enrollRevoked.PathIdentifier,
			http.StatusNotFound,
			false,
		},
		{
			"HEAD with revoked identifier",
			http.MethodHead,
			enrollRevoked.PathIdentifier,
			http.StatusNotFound,
			false,
		},
		{
			"POST with revoked identifier",
			http.MethodPost,
			enrollRevoked.PathIdentifier,
			http.StatusNotFound,
			false,
		},
		{
			"GET with expired identifier",
			http.MethodGet,
			enrollExpired.PathIdentifier,
			http.StatusNotFound,
			false,
		},
		{
			"HEAD with expired identifier",
			http.MethodHead,
			enrollExpired.PathIdentifier,
			http.StatusNotFound,
			false,
		},
		{
			"POST with expired identifier",
			http.MethodPost,
			enrollExpired.PathIdentifier,
			http.StatusNotFound,
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			result, resp := s.newNonce(t, c.method, c.identifier)
			require.Equal(t, c.wantStatus, resp.StatusCode)
			require.Equal(t, c.method, result.HTTPMethod)
			nonce := resp.Header.Get("Replay-Nonce")
			if c.wantNonce {
				t.Logf("Received nonce: %s", nonce)
				require.NotEmpty(t, nonce)
				require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
			} else {
				require.Empty(t, nonce)
				require.Empty(t, resp.Header.Get("Cache-Control"))
			}
		})
	}
}

func testGetDirectory(t *testing.T, s *integrationTestSuite) {
	// create a valid enrollment
	enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollValid)

	// create a revoked enrollment
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	// create an expired enrollment
	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	cases := []struct {
		desc       string
		method     string
		identifier string
		wantStatus int
		wantDir    *types.Directory
	}{
		{
			"GET with unknown identifier",
			http.MethodGet,
			"no-such-identifier",
			http.StatusNotFound,
			nil,
		},
		{
			"POST with unknown identifier",
			http.MethodPost,
			"no-such-identifier",
			http.StatusNotFound,
			nil,
		},
		{
			"GET with valid identifier",
			http.MethodGet,
			enrollValid.PathIdentifier,
			http.StatusOK,
			&types.Directory{
				NewNonce:   s.server.URL + "/api/mdm/acme/" + enrollValid.PathIdentifier + "/new_nonce",
				NewAccount: s.server.URL + "/api/mdm/acme/" + enrollValid.PathIdentifier + "/new_account",
				NewOrder:   s.server.URL + "/api/mdm/acme/" + enrollValid.PathIdentifier + "/new_order",
			},
		},
		{
			"POST with valid identifier",
			http.MethodPost,
			enrollValid.PathIdentifier,
			http.StatusOK,
			&types.Directory{
				NewNonce:   s.server.URL + "/api/mdm/acme/" + enrollValid.PathIdentifier + "/new_nonce",
				NewAccount: s.server.URL + "/api/mdm/acme/" + enrollValid.PathIdentifier + "/new_account",
				NewOrder:   s.server.URL + "/api/mdm/acme/" + enrollValid.PathIdentifier + "/new_order",
			},
		},
		{
			"GET with revoked identifier",
			http.MethodGet,
			enrollRevoked.PathIdentifier,
			http.StatusNotFound,
			nil,
		},
		{
			"POST with revoked identifier",
			http.MethodPost,
			enrollRevoked.PathIdentifier,
			http.StatusNotFound,
			nil,
		},
		{
			"GET with expired identifier",
			http.MethodGet,
			enrollExpired.PathIdentifier,
			http.StatusNotFound,
			nil,
		},
		{
			"POST with expired identifier",
			http.MethodPost,
			enrollExpired.PathIdentifier,
			http.StatusNotFound,
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			result, resp := s.getDirectory(t, c.method, c.identifier)
			require.Equal(t, c.wantStatus, resp.StatusCode)
			if c.wantDir != nil {
				require.Equal(t, c.wantDir, result.Directory)
			} else {
				require.Nil(t, result)
			}
		})
	}
}

func testCreateAccount(t *testing.T, s *integrationTestSuite) {
	// create enrollments for testing
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	t.Run("create new account", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(enrollValid.PathIdentifier), nil)
		acctResp, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		require.NotEmpty(t, resp.Header.Get("Location"))
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp.Header.Get("Location"))

		require.Equal(t, "valid", acctResp.Status)
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+/orders$`, acctResp.Orders)
	})

	t.Run("return existing account with same JWK", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		// create account
		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce1 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody1 := buildJWS(t, privateKey, nonce1, "", s.newAccountURL(enrollValid.PathIdentifier), nil)
		acctResp1, _, resp1 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody1)

		require.Equal(t, http.StatusCreated, resp1.StatusCode)
		require.NotEmpty(t, resp1.Header.Get("Location"))
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp1.Header.Get("Location"))
		location1 := resp1.Header.Get("Location")

		// create again with same key - should return existing (use the valid returned nonce)
		nonce2 := resp1.Header.Get("Replay-Nonce")
		jwsBody2 := buildJWS(t, privateKey, nonce2, "", s.newAccountURL(enrollValid.PathIdentifier), nil)
		acctResp2, _, resp2 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody2)

		require.Equal(t, http.StatusOK, resp2.StatusCode)

		require.Equal(t, "valid", acctResp2.Status)
		// same account, same orders URL
		require.Equal(t, acctResp1.Orders, acctResp2.Orders)
		require.NotEmpty(t, resp2.Header.Get("Location"))
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp2.Header.Get("Location"))
		require.Equal(t, location1, resp2.Header.Get("Location"))
	})

	t.Run("too many accounts", func(t *testing.T) {
		// use a dedicated enrollment so prior sub-tests don't affect the count
		enrollForLimit := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollForLimit)

		// create 3 accounts (the maximum allowed per enrollment)
		nonce := s.getNonce(t, enrollForLimit.PathIdentifier)
		for range 3 {
			key, err := testhelpers.GenerateTestKey()
			require.NoError(t, err)
			jwsBody := buildJWS(t, key, nonce, "", s.newAccountURL(enrollForLimit.PathIdentifier), nil)
			_, _, resp := s.createAccount(t, enrollForLimit.PathIdentifier, jwsBody)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			nonce = resp.Header.Get("Replay-Nonce")
		}

		// 4th should fail
		key, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		jwsBody := buildJWS(t, key, nonce, "", s.newAccountURL(enrollForLimit.PathIdentifier), nil)
		_, acmeErr, resp := s.createAccount(t, enrollForLimit.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Contains(t, acmeErr.Type, "error/tooManyAccounts")
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("revoked account", func(t *testing.T) {
		// use a dedicated enrollment to avoid interference with other sub-tests
		enrollForRevoke := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollForRevoke)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)

		// create an account
		nonce := s.getNonce(t, enrollForRevoke.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(enrollForRevoke.PathIdentifier), nil)
		_, _, resp1 := s.createAccount(t, enrollForRevoke.PathIdentifier, jwsBody)
		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		// revoke it directly in the DB
		var accountID uint
		err = s.DB.GetContext(t.Context(), &accountID, `SELECT id FROM acme_accounts WHERE acme_enrollment_id = ? ORDER BY id DESC LIMIT 1`, enrollForRevoke.ID)
		require.NoError(t, err)
		_, err = s.DB.ExecContext(t.Context(), `UPDATE acme_accounts SET revoked = 1 WHERE id = ?`, accountID)
		require.NoError(t, err)

		// try to create again with the same key — should get accountRevoked error
		nonce2 := resp1.Header.Get("Replay-Nonce")
		jwsBody2 := buildJWS(t, privateKey, nonce2, "", s.newAccountURL(enrollForRevoke.PathIdentifier), nil)
		_, acmeErr, resp2 := s.createAccount(t, enrollForRevoke.PathIdentifier, jwsBody2)

		require.Equal(t, http.StatusBadRequest, resp2.StatusCode)
		require.NotEmpty(t, resp2.Header.Get("Replay-Nonce"))
		require.Contains(t, acmeErr.Type, "error/accountRevoked")
		require.Empty(t, resp2.Header.Get("Location"))
	})

	t.Run("onlyReturnExisting account exists", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		// create account first
		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce1 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody1 := buildJWS(t, privateKey, nonce1, "", s.newAccountURL(enrollValid.PathIdentifier), nil)
		acctResp1, _, resp1 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody1)
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp1.Header.Get("Location"))
		location1 := resp1.Header.Get("Location")

		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		// lookup with onlyReturnExisting (use the valid returned nonce)
		nonce2 := resp1.Header.Get("Replay-Nonce")
		jwsBody2 := buildJWS(t, privateKey, nonce2, "", s.newAccountURL(enrollValid.PathIdentifier), map[string]any{"onlyReturnExisting": true})
		acctResp2, _, resp2 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody2)

		require.Equal(t, http.StatusOK, resp2.StatusCode)

		require.Equal(t, "valid", acctResp2.Status)
		require.Equal(t, acctResp1.Orders, acctResp2.Orders)
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp2.Header.Get("Location"))
		require.Equal(t, location1, resp2.Header.Get("Location"))
	})

	t.Run("onlyReturnExisting account does not exist", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{"onlyReturnExisting": true}
		jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(enrollValid.PathIdentifier), payload)
		_, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Contains(t, acmeErr.Type, "error:accountDoesNotExist")
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("unknown identifier", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		// we need a valid enrollment to get a nonce, then use a bad identifier for the account request
		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		badIdentifier := "no-such-identifier"
		jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(badIdentifier), nil)
		acctResp, acmeErr, resp := s.createAccount(t, badIdentifier, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, acctResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "error/enrollmentNotFound")
		// no nonce generated when the identifier is unknown
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("revoked enrollment", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		// get nonce from valid enrollment, then try to create account on revoked
		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(enrollRevoked.PathIdentifier), nil)
		acctResp, acmeErr, resp := s.createAccount(t, enrollRevoked.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, acctResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "error/enrollmentNotFound")
		// no nonce generated when the identifier is invalid
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("expired enrollment", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(enrollExpired.PathIdentifier), nil)
		acctResp, acmeErr, resp := s.createAccount(t, enrollExpired.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, acctResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "error/enrollmentNotFound")
		// no nonce generated when the identifier is invalid
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("invalid nonce", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, "bad-nonce-value", "", s.newAccountURL(enrollValid.PathIdentifier), nil)
		_, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "error:badNonce")
		// it does generate a new valid nonce
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("empty url in JWS header", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", "", nil)
		_, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "malformed")
		// no nonce generated — decode error bypasses the endpoint handler
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("invalid url in JWS header", func(t *testing.T) {
		enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enrollValid)

		privateKey, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", "http://example.com", nil)
		_, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unauthorized")
		// nonce is generated because the error occurs in the endpoint handler (not decode)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})
}

func testCreateOrder(t *testing.T, s *integrationTestSuite) {
	// create enrollments shared across sub-tests for error cases
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	t.Run("create new order", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		orderResp, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		require.NotEmpty(t, resp.Header.Get("Location"))
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/orders/\d+`, resp.Header.Get("Location"))

		require.Equal(t, types.OrderStatusPending, orderResp.Status)
		require.Len(t, orderResp.Identifiers, 1)
		require.Equal(t, "permanent-identifier", orderResp.Identifiers[0].Type)
		require.Equal(t, enroll.HostIdentifier, orderResp.Identifiers[0].Value)
		require.Len(t, orderResp.Authorizations, 1)
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/authorizations/\d+`, orderResp.Authorizations[0])
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/orders/\d+/finalize`, orderResp.Finalize)
	})

	t.Run("unsupported identifier type", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "dns", "value": "example.com"},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unsupportedIdentifier")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("wrong identifier value", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": "wrong-host-identifier"},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "rejectedIdentifier")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("multiple identifiers", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
				{"type": "permanent-identifier", "value": "another"},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unsupportedIdentifier")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("no identifiers", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unsupportedIdentifier")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("notBefore set", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
			"notBefore": time.Now().UTC().Format(time.RFC3339),
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "malformed")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("notAfter set", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
			"notAfter": time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "malformed")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("too many orders", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		// create 3 orders (the maximum allowed per account)
		for range 3 {
			payload := map[string]any{
				"identifiers": []map[string]string{
					{"type": "permanent-identifier", "value": enroll.HostIdentifier},
				},
			}
			jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
			_, _, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			nonce = resp.Header.Get("Replay-Nonce")
		}

		// 4th should fail
		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "tooManyOrders")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("unknown identifier", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		badIdentifier := "no-such-identifier"
		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(badIdentifier), payload)
		orderResp, acmeErr, resp := s.createOrder(t, badIdentifier, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, orderResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("revoked enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enrollRevoked.PathIdentifier), payload)
		orderResp, acmeErr, resp := s.createOrder(t, enrollRevoked.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, orderResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("expired enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enrollExpired.PathIdentifier), payload)
		orderResp, acmeErr, resp := s.createOrder(t, enrollExpired.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, orderResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})

	t.Run("invalid nonce", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, _ := s.createAccountForOrder(t, enroll)

		payload := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll.HostIdentifier},
			},
		}
		jwsBody := buildJWS(t, privateKey, "bad-nonce-value", accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
		_, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badNonce")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})
}

func testGetOrder(t *testing.T, s *integrationTestSuite) {
	// create enrollments shared across sub-tests for error cases
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	t.Run("get existing order", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		// GET the order via POST-as-GET
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(enroll.PathIdentifier, orderResp.ID), nil)
		gotOrder, acmeErr, resp := s.getOrder(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

		require.Equal(t, types.OrderStatusPending, gotOrder.Status)
		require.Len(t, gotOrder.Identifiers, 1)
		require.Equal(t, "permanent-identifier", gotOrder.Identifiers[0].Type)
		require.Equal(t, enroll.HostIdentifier, gotOrder.Identifiers[0].Value)
		require.Len(t, gotOrder.Authorizations, 1)
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/authorizations/\d+`, gotOrder.Authorizations[0])
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/orders/\d+/finalize`, gotOrder.Finalize)
		require.Empty(t, gotOrder.Certificate)
	})

	t.Run("non-empty payload rejected for POST-as-GET", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		// POST-as-GET requests must have an empty payload; sending a non-empty
		// one should be rejected with a malformed error.
		nonEmptyPayload := map[string]any{"foo": "bar"}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(enroll.PathIdentifier, orderResp.ID), nonEmptyPayload)
		gotOrder, acmeErr, resp := s.getOrder(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Nil(t, gotOrder)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "malformed")
		require.Contains(t, acmeErr.Detail, "payload must be empty")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("get finalized valid order includes certificate URL", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		// force the order to finalized + valid state directly in the database
		_, err := s.DB.ExecContext(t.Context(),
			`UPDATE acme_orders SET finalized = 1, status = 'valid' WHERE id = ?`, orderResp.ID)
		require.NoError(t, err)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(enroll.PathIdentifier, orderResp.ID), nil)
		gotOrder, acmeErr, resp := s.getOrder(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.Equal(t, types.OrderStatusValid, gotOrder.Status)
		require.Regexp(t, fmt.Sprintf("/api/mdm/acme/%s/orders/%d/certificate", enroll.PathIdentifier, orderResp.ID), gotOrder.Certificate)
	})

	t.Run("order does not exist", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		nonExistentOrderID := uint(99999)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(enroll.PathIdentifier, nonExistentOrderID), nil)
		orderResp, acmeErr, resp := s.getOrder(t, enroll.PathIdentifier, nonExistentOrderID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, orderResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderDoesNotExist")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("order belongs to different account", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		// create account A and an order under it
		_, _, orderResp, _ := s.createOrderForGet(t, enroll)

		// create account B (different key)
		privateKeyB, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonceB := s.getNonce(t, enroll.PathIdentifier)
		jwsBodyB := buildJWS(t, privateKeyB, nonceB, "", s.newAccountURL(enroll.PathIdentifier), nil)
		_, _, respB := s.createAccount(t, enroll.PathIdentifier, jwsBodyB)
		require.Equal(t, http.StatusCreated, respB.StatusCode)
		accountURLB := respB.Header.Get("Location")
		nonceB = respB.Header.Get("Replay-Nonce")

		// try to GET account A's order using account B's credentials
		jwsBody := buildJWS(t, privateKeyB, nonceB, accountURLB, s.getOrderURL(enroll.PathIdentifier, orderResp.ID), nil)
		gotOrder, acmeErr, resp := s.getOrder(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, gotOrder)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderDoesNotExist")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("unknown identifier", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		badIdentifier := "no-such-identifier"
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(badIdentifier, orderResp.ID), nil)
		gotOrder, acmeErr, resp := s.getOrder(t, badIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, gotOrder)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("revoked enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(enrollRevoked.PathIdentifier, orderResp.ID), nil)
		gotOrder, acmeErr, resp := s.getOrder(t, enrollRevoked.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, gotOrder)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("expired enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getOrderURL(enrollExpired.PathIdentifier, orderResp.ID), nil)
		gotOrder, acmeErr, resp := s.getOrder(t, enrollExpired.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, gotOrder)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("invalid nonce", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, _ := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, "bad-nonce-value", accountURL, s.getOrderURL(enroll.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getOrder(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badNonce")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})
}

func testListAccountOrders(t *testing.T, s *integrationTestSuite) {
	// create enrollments shared across sub-tests for error cases
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	t.Run("list orders with one order", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, _, nonce := s.createOrderForGet(t, enroll)
		accountID := parseAccountID(t, accountURL)

		// POST-as-GET to list orders
		listURL := s.listOrdersURL(enroll.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enroll.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

		require.Len(t, listResp.Orders, 1)
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/orders/\d+`, listResp.Orders[0])
	})

	t.Run("list orders with multiple orders", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		// create 3 orders
		for range 3 {
			payload := map[string]any{
				"identifiers": []map[string]string{
					{"type": "permanent-identifier", "value": enroll.HostIdentifier},
				},
			}
			jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
			_, _, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			nonce = resp.Header.Get("Replay-Nonce")
		}

		// list orders
		listURL := s.listOrdersURL(enroll.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enroll.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.Len(t, listResp.Orders, 3)
		for _, orderURL := range listResp.Orders {
			require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/orders/\d+`, orderURL)
		}
	})

	t.Run("list orders with no orders", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		listURL := s.listOrdersURL(enroll.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enroll.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, listResp.Orders)
	})

	t.Run("list orders excludes invalid orders", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		// create 2 orders
		orderIDs := make([]uint, 0, 2)
		for range cap(orderIDs) {
			payload := map[string]any{
				"identifiers": []map[string]string{
					{"type": "permanent-identifier", "value": enroll.HostIdentifier},
				},
			}
			jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
			orderResp, _, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			orderIDs = append(orderIDs, orderResp.ID)
			nonce = resp.Header.Get("Replay-Nonce")
		}

		// mark the first order as invalid
		_, err := s.DB.ExecContext(t.Context(),
			`UPDATE acme_orders SET status = 'invalid' WHERE id = ?`, orderIDs[0])
		require.NoError(t, err)

		// list orders — should only see the second order
		listURL := s.listOrdersURL(enroll.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enroll.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.Len(t, listResp.Orders, 1)
		require.Contains(t, listResp.Orders[0], fmt.Sprintf("/orders/%d", orderIDs[1]))
	})

	t.Run("order belongs to different account", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		// create account A with an order
		s.createOrderForGet(t, enroll)

		// create account B (different key)
		privateKeyB, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonceB := s.getNonce(t, enroll.PathIdentifier)
		jwsBodyB := buildJWS(t, privateKeyB, nonceB, "", s.newAccountURL(enroll.PathIdentifier), nil)
		_, _, respB := s.createAccount(t, enroll.PathIdentifier, jwsBodyB)
		require.Equal(t, http.StatusCreated, respB.StatusCode)
		accountURLB := respB.Header.Get("Location")
		accountIDB := parseAccountID(t, accountURLB)
		nonceB = respB.Header.Get("Replay-Nonce")

		// list orders for account B — should be empty since the order belongs to account A
		listURL := s.listOrdersURL(enroll.PathIdentifier, accountIDB)
		jwsBody := buildJWS(t, privateKeyB, nonceB, accountURLB, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enroll.PathIdentifier, accountIDB, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.Empty(t, listResp.Orders)
	})

	t.Run("unknown identifier", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		badIdentifier := "no-such-identifier"
		listURL := s.listOrdersURL(badIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, badIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, listResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("revoked enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		listURL := s.listOrdersURL(enrollRevoked.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enrollRevoked.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, listResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("expired enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		listURL := s.listOrdersURL(enrollExpired.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, listURL, nil)
		listResp, acmeErr, resp := s.listOrders(t, enrollExpired.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Nil(t, listResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("invalid nonce", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, _ := s.createAccountForOrder(t, enroll)
		accountID := parseAccountID(t, accountURL)

		listURL := s.listOrdersURL(enroll.PathIdentifier, accountID)
		jwsBody := buildJWS(t, privateKey, "bad-nonce-value", accountURL, listURL, nil)
		_, acmeErr, resp := s.listOrders(t, enroll.PathIdentifier, accountID, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badNonce")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})
}

func testGetCertificate(t *testing.T, s *integrationTestSuite) {
	// create enrollments shared across sub-tests for error cases
	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	t.Run("happy path", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		testCertPEM := "-----BEGIN CERTIFICATE-----\ntest-cert-pem\n-----END CERTIFICATE-----"
		s.finalizeOrderWithCert(t, orderResp.ID, 5001, testCertPEM)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enroll.PathIdentifier, orderResp.ID), nil)
		certBody, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Equal(t, "application/pem-certificate-chain", resp.Header.Get("Content-Type"))

		expectedRootPEM := "-----BEGIN CERTIFICATE-----\nroot\n-----END CERTIFICATE-----\n"
		require.Equal(t, testCertPEM+"\n"+expectedRootPEM, certBody)
	})

	t.Run("non-empty payload rejected for POST-as-GET", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		testCertPEM := "-----BEGIN CERTIFICATE-----\ntest-cert-pem-2\n-----END CERTIFICATE-----"
		s.finalizeOrderWithCert(t, orderResp.ID, 5002, testCertPEM)

		nonEmptyPayload := map[string]any{"foo": "bar"}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enroll.PathIdentifier, orderResp.ID), nonEmptyPayload)
		_, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "malformed")
		require.Contains(t, acmeErr.Detail, "payload must be empty")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("order not finalized (pending)", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enroll.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderNotFinalized")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("order in invalid state", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		_, err := s.DB.ExecContext(t.Context(),
			`UPDATE acme_orders SET status = 'invalid' WHERE id = ?`, orderResp.ID)
		require.NoError(t, err)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enroll.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderDoesNotExist")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("order does not exist", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		nonExistentOrderID := uint(99999)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enroll.PathIdentifier, nonExistentOrderID), nil)
		_, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, nonExistentOrderID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderDoesNotExist")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("order belongs to different account", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)

		// create account A and an order under it
		_, _, orderResp, _ := s.createOrderForGet(t, enroll)

		// create account B (different key)
		privateKeyB, err := testhelpers.GenerateTestKey()
		require.NoError(t, err)
		nonceB := s.getNonce(t, enroll.PathIdentifier)
		jwsBodyB := buildJWS(t, privateKeyB, nonceB, "", s.newAccountURL(enroll.PathIdentifier), nil)
		_, _, respB := s.createAccount(t, enroll.PathIdentifier, jwsBodyB)
		require.Equal(t, http.StatusCreated, respB.StatusCode)
		accountURLB := respB.Header.Get("Location")
		nonceB = respB.Header.Get("Replay-Nonce")

		// try to GET account A's order's certificate using account B's credentials
		jwsBody := buildJWS(t, privateKeyB, nonceB, accountURLB, s.getCertificateURL(enroll.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderDoesNotExist")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("unknown identifier", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		badIdentifier := "no-such-identifier"
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(badIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, badIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("revoked enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enrollRevoked.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, enrollRevoked.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("expired enrollment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.getCertificateURL(enrollExpired.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, enrollExpired.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "enrollmentNotFound")
		require.Empty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("invalid nonce", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, orderResp, _ := s.createOrderForGet(t, enroll)

		jwsBody := buildJWS(t, privateKey, "bad-nonce-value", accountURL, s.getCertificateURL(enroll.PathIdentifier, orderResp.ID), nil)
		_, acmeErr, resp := s.getCertificate(t, enroll.PathIdentifier, orderResp.ID, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badNonce")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})
}

func testGetAuthorization(t *testing.T, s *integrationTestSuite) {
	enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enroll)
	payload := map[string]any{
		"identifiers": []map[string]string{
			{"type": "permanent-identifier", "value": enroll.HostIdentifier},
		},
	}
	privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)
	jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
	orderResp, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)
	require.Nil(t, acmeErr)
	require.NotNil(t, orderResp)
	require.Len(t, orderResp.Authorizations, 1) // We only expect one authorization for any given order
	nextNonce := resp.Header.Get("Replay-Nonce")

	t.Run("successful authorization retrieval", func(t *testing.T) {
		authURL := orderResp.Authorizations[0]
		jws := buildJWS(t, privateKey, nextNonce, accountURL, authURL, nil)
		authResp, acmeErr, _ := s.getAuthorization(t, authURL, jws)
		require.Nil(t, acmeErr)
		require.NotNil(t, authResp)
		require.Equal(t, types.AuthorizationStatusPending, authResp.Status)
		require.Equal(t, "permanent-identifier", authResp.Identifier.Type)
		require.Equal(t, enroll.HostIdentifier, authResp.Identifier.Value)
		require.Len(t, authResp.Challenges, 1)
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/challenges/\d+`, authResp.Challenges[0].URL)
		require.Equal(t, types.DeviceAttestationChallengeType, authResp.Challenges[0].ChallengeType)
	})

	t.Run("valid auth but for a different enrollment", func(t *testing.T) {
		// create a second enrollment and order to get a different auth URL
		enroll2 := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll2)
		privateKey2, accountURL2, nonce2 := s.createAccountForOrder(t, enroll2)
		payload2 := map[string]any{
			"identifiers": []map[string]string{
				{"type": "permanent-identifier", "value": enroll2.HostIdentifier},
			},
		}
		jwsBody2 := buildJWS(t, privateKey2, nonce2, accountURL2, s.newOrderURL(enroll2.PathIdentifier), payload2)
		orderResp2, acmeErr2, resp2 := s.createOrder(t, enroll2.PathIdentifier, jwsBody2)
		require.Nil(t, acmeErr2)
		require.NotNil(t, orderResp2)
		require.Len(t, orderResp2.Authorizations, 1)
		authURL2 := orderResp2.Authorizations[0]
		nextNonce2 := resp2.Header.Get("Replay-Nonce")

		// try to use the auth URL from enroll2's order with enroll1's account/key
		jws := buildJWS(t, privateKey, nextNonce2, accountURL, authURL2, nil)
		authResp, acmeErr, resp := s.getAuthorization(t, authURL2, jws)
		nextNonce = resp.Header.Get("Replay-Nonce")

		require.Nil(t, authResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unauthorized")
	})

	t.Run("non existing auth", func(t *testing.T) {
		authURL := s.server.URL + "/api/mdm/acme/" + enroll.PathIdentifier + "/authorizations/99999"
		jws := buildJWS(t, privateKey, nextNonce, accountURL, authURL, nil)
		authResp, acmeErr, _ := s.getAuthorization(t, authURL, jws)

		require.Nil(t, authResp)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "error/authorizationDoesNotExist")
	})
}

func testFinalizeOrder(t *testing.T, s *integrationTestSuite) {
	t.Run("successful finalize", func(t *testing.T) {
		enroll, privateKey, accountURL, orderResp, nonce := s.createOrderForFinalize(t)
		s.makeOrderReady(t, orderResp.ID)

		csrPEM, _, err := testhelpers.GenerateCSRDER(enroll.HostIdentifier)
		require.NoError(t, err)
		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, orderResp.ID)
		payload := map[string]any{"csr": csrPEM}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		result, acmeErr, resp := s.finalizeOrder(t, finalizeURL, jwsBody)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Nil(t, acmeErr)
		require.NotNil(t, result)
		require.Equal(t, types.OrderStatusValid, result.Status)
		require.NotEmpty(t, result.Certificate)
		require.Regexp(t, "/api/mdm/acme/"+enroll.PathIdentifier+`/orders/\d+/certificate`, result.Certificate)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.NotEmpty(t, resp.Header.Get("Location"))
	})

	t.Run("order not ready - pending status", func(t *testing.T) {
		enroll, privateKey, accountURL, orderResp, nonce := s.createOrderForFinalize(t)
		// order is still in "pending" status (not made ready)

		csrPEM, _, err := testhelpers.GenerateCSRDER(enroll.HostIdentifier)
		require.NoError(t, err)
		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, orderResp.ID)
		payload := map[string]any{"csr": csrPEM}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		_, acmeErr, resp := s.finalizeOrder(t, finalizeURL, jwsBody)

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderNotReady")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("already finalized order", func(t *testing.T) {
		enroll, privateKey, accountURL, orderResp, nonce := s.createOrderForFinalize(t)
		s.makeOrderReady(t, orderResp.ID)

		// finalize the order first
		csrPEM, _, err := testhelpers.GenerateCSRDER(enroll.HostIdentifier)
		require.NoError(t, err)
		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, orderResp.ID)
		payload := map[string]any{"csr": csrPEM}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		_, _, resp := s.finalizeOrder(t, finalizeURL, jwsBody)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		nonce = resp.Header.Get("Replay-Nonce")

		// try to finalize again
		jwsBody2 := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		_, acmeErr, resp2 := s.finalizeOrder(t, finalizeURL, jwsBody2)

		require.Equal(t, http.StatusForbidden, resp2.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderNotReady")
		require.NotEmpty(t, resp2.Header.Get("Replay-Nonce"))
	})

	t.Run("CSR common name mismatch", func(t *testing.T) {
		enroll, privateKey, accountURL, orderResp, nonce := s.createOrderForFinalize(t)
		s.makeOrderReady(t, orderResp.ID)

		csrPEM, _, err := testhelpers.GenerateCSRDER("wrong-common-name")
		require.NoError(t, err)
		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, orderResp.ID)
		payload := map[string]any{"csr": csrPEM}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		_, acmeErr, resp := s.finalizeOrder(t, finalizeURL, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badCSR")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("invalid CSR PEM", func(t *testing.T) {
		enroll, privateKey, accountURL, orderResp, nonce := s.createOrderForFinalize(t)
		s.makeOrderReady(t, orderResp.ID)

		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, orderResp.ID)
		payload := map[string]any{"csr": "not-valid-pem"}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		_, acmeErr, resp := s.finalizeOrder(t, finalizeURL, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badCSR")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("non-existing order", func(t *testing.T) {
		enroll, privateKey, accountURL, _, nonce := s.createOrderForFinalize(t)

		csrPEM, _, err := testhelpers.GenerateCSRDER(enroll.HostIdentifier)
		require.NoError(t, err)
		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, 99999)
		payload := map[string]any{"csr": csrPEM}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, finalizeURL, payload)
		_, acmeErr, resp := s.finalizeOrder(t, finalizeURL, jwsBody)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "orderDoesNotExist")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})

	t.Run("invalid nonce", func(t *testing.T) {
		enroll, privateKey, accountURL, orderResp, _ := s.createOrderForFinalize(t)
		s.makeOrderReady(t, orderResp.ID)

		csrPEM, _, err := testhelpers.GenerateCSRDER(enroll.HostIdentifier)
		require.NoError(t, err)
		finalizeURL := s.finalizeOrderURL(enroll.PathIdentifier, orderResp.ID)
		payload := map[string]any{"csr": csrPEM}
		jwsBody := buildJWS(t, privateKey, "bad-nonce", accountURL, finalizeURL, payload)
		_, acmeErr, resp := s.finalizeOrder(t, finalizeURL, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "badNonce")
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
	})
}

func testDoChallengeDeviceAttestation(t *testing.T, s *integrationTestSuite) {
	t.Run("successful device attestation", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour)), HostIdentifier: "valid-serial"}
		s.InsertACMEEnrollment(t, enroll)

		privateKey, accountURL, challengeURL, challengeToken, nonce := s.createOrderForChallenge(t, enroll)
		leafCert, err := testhelpers.BuildAttestationLeafCert(s.attestCA, s.attestCAKey, enroll.HostIdentifier, challengeToken)
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leafCert, s.attestCA)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		challengeResp, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.Nil(t, acmeErr)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotNil(t, challengeResp)
		require.Equal(t, types.ChallengeStatusValid, challengeResp.Status)
	})

	t.Run("challenge not pending", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour)), HostIdentifier: "valid-serial"}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, challengeToken, nonce := s.createOrderForChallenge(t, enroll)

		// first do a successful challenge to move it out of pending state
		leafCert, err := testhelpers.BuildAttestationLeafCert(s.attestCA, s.attestCAKey, enroll.HostIdentifier, challengeToken)
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leafCert, s.attestCA)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		challengeResp, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)
		require.Nil(t, acmeErr)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotNil(t, challengeResp)
		require.Equal(t, types.ChallengeStatusValid, challengeResp.Status)
		nonce = resp.Header.Get("Replay-Nonce")

		// try to do the challenge again with the same JWS body
		jwsBody = buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		challengeResp2, acmeErr2, resp2 := s.doChallenge(t, challengeURL, jwsBody)
		require.NotNil(t, acmeErr2)
		require.Equal(t, http.StatusBadRequest, resp2.StatusCode)
		require.Nil(t, challengeResp2)
		require.Contains(t, acmeErr2.Type, "invalidChallengeStatus")
		require.Contains(t, acmeErr2.Detail, "not pending and can not be validated")
	})

	t.Run("device attestation error", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, _, nonce := s.createOrderForChallenge(t, enroll)
		payload := map[string]any{
			"error": "device attestation failed",
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unauthorized")
	})

	t.Run("bad base64 payload", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, _, nonce := s.createOrderForChallenge(t, enroll)
		// JWS with invalid base64 payload
		body := map[string]any{
			"attObj": "not a valid base64 string",
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, body)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Detail, "illegal base64 data")
		require.Contains(t, acmeErr.Type, "malformed")
	})

	t.Run("bad CBOR payload", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, _, nonce := s.createOrderForChallenge(t, enroll)
		// JWS with base64 that decodes but is not valid CBOR
		body := map[string]any{
			"attObj": base64.RawURLEncoding.EncodeToString([]byte("not valid CBOR data")),
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, body)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Detail, "not correctly CBOR formatted")
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
	})

	t.Run("unsupported format", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, _, nonce := s.createOrderForChallenge(t, enroll)
		// JWS with base64 that decodes to CBOR but has an unsupported format
		cborData, err := cbor.Marshal(map[string]any{"fmt": "unsupported-format"})
		require.NoError(t, err)
		body := map[string]any{
			"attObj": base64.RawURLEncoding.EncodeToString(cborData),
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, body)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Detail, "Unsupported device attestation format")
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
	})

	t.Run("invalid cert chain", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, challengeToken, nonce := s.createOrderForChallenge(t, enroll)

		// build a cert chain that is not valid (leaf signed by unknown CA)
		cert, key, err := testhelpers.GenerateTestAttestationCA()
		require.NoError(t, err)
		leafCert, err := testhelpers.BuildAttestationLeafCert(cert, key, enroll.HostIdentifier, challengeToken)
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leafCert, cert)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.NotNil(t, acmeErr)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
		require.Contains(t, acmeErr.Detail, "Failed to verify Apple Root CA is part of certificate chain")
	})

	t.Run("freshness nonce mismatch", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, _, nonce := s.createOrderForChallenge(t, enroll)

		leaf, err := testhelpers.BuildAttestationLeafCert(s.attestCA, s.attestCAKey, enroll.HostIdentifier, "dummy-challenge-token")
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leaf, s.attestCA)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.NotNil(t, acmeErr)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
		require.Contains(t, acmeErr.Detail, "Apple freshness nonce does not match challenge token")
	})

	t.Run("device serial mismatch", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, challengeToken, nonce := s.createOrderForChallenge(t, enroll)

		leaf, err := testhelpers.BuildAttestationLeafCert(s.attestCA, s.attestCAKey, "some-other-serial", challengeToken)
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leaf, s.attestCA)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.NotNil(t, acmeErr)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
		require.Contains(t, acmeErr.Detail, "Serial number in certificate does not match enrollment's host identifier")
	})

	t.Run("missing DEP assignment", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour)), HostIdentifier: "serial-without-dep"}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, challengeURL, challengeToken, nonce := s.createOrderForChallenge(t, enroll)

		leaf, err := testhelpers.BuildAttestationLeafCert(s.attestCA, s.attestCAKey, enroll.HostIdentifier, challengeToken)
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leaf, s.attestCA)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.NotNil(t, acmeErr)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
		require.Contains(t, acmeErr.Detail, "No DEP assignments found for serial number in certificate")
	})

	t.Run("non existing challenge", func(t *testing.T) {
		enroll := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
		s.InsertACMEEnrollment(t, enroll)
		privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

		challengeURL := s.server.URL + "/api/mdm/acme/" + enroll.PathIdentifier + "/challenges/99999"
		payload := map[string]any{
			"attObj": "fake",
		}
		jwsBody := buildJWS(t, privateKey, nonce, accountURL, challengeURL, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL, jwsBody)

		require.NotNil(t, acmeErr)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "challengeDoesNotExist")
		require.Contains(t, acmeErr.Detail, "Challenge with ID 99999 not found")
	})

	t.Run("challenge for different enrollment fails", func(t *testing.T) {
		enroll1 := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour)), HostIdentifier: "serial1"}
		enroll2 := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour)), HostIdentifier: "serial2"}
		s.InsertACMEEnrollment(t, enroll1)
		s.InsertACMEEnrollment(t, enroll2)

		privateKey1, accountURL1, challengeURL1, challengeToken1, nonce := s.createOrderForChallenge(t, enroll1)

		// try to do enroll1's challenge with a payload built from enroll2's device cert
		leaf, err := testhelpers.BuildAttestationLeafCert(s.attestCA, s.attestCAKey, enroll2.HostIdentifier, challengeToken1)
		require.NoError(t, err)
		payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leaf, s.attestCA)
		require.NoError(t, err)
		jwsBody := buildJWS(t, privateKey1, nonce, accountURL1, challengeURL1, payload)
		_, acmeErr, resp := s.doChallenge(t, challengeURL1, jwsBody)

		require.NotNil(t, acmeErr)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, acmeErr.Type, "badAttestationStatement")
		require.Contains(t, acmeErr.Detail, "Serial number in certificate does not match enrollment's host identifier")
	})
}
