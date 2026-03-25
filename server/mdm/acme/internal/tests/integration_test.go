package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

// newAccountURL returns the full URL for the new_account endpoint.
func (s *integrationTestSuite) newAccountURL(pathIdentifier string) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/new_account", s.server.URL, pathIdentifier)
}

func TestIntegration(t *testing.T) {
	s := setupIntegrationTest(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, s *integrationTestSuite)
	}{
		{"NewNonce", testNewNonce},
		{"GetDirectory", testGetDirectory},
		{"CreateAccount", testCreateAccount},
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

		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, s.newAccountURL(enrollValid.PathIdentifier), nil)
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
		privateKey := generateTestKey(t)
		nonce1 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody1 := buildJWS(t, privateKey, nonce1, s.newAccountURL(enrollValid.PathIdentifier), nil)
		acctResp1, _, resp1 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody1)

		require.Equal(t, http.StatusCreated, resp1.StatusCode)
		require.NotEmpty(t, resp1.Header.Get("Location"))
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp1.Header.Get("Location"))
		location1 := resp1.Header.Get("Location")

		// create again with same key - should return existing (use the valid returned nonce)
		nonce2 := resp1.Header.Get("Replay-Nonce")
		jwsBody2 := buildJWS(t, privateKey, nonce2, s.newAccountURL(enrollValid.PathIdentifier), nil)
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
			key := generateTestKey(t)
			jwsBody := buildJWS(t, key, nonce, s.newAccountURL(enrollForLimit.PathIdentifier), nil)
			_, _, resp := s.createAccount(t, enrollForLimit.PathIdentifier, jwsBody)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			nonce = resp.Header.Get("Replay-Nonce")
		}

		// 4th should fail
		key := generateTestKey(t)
		jwsBody := buildJWS(t, key, nonce, s.newAccountURL(enrollForLimit.PathIdentifier), nil)
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

		privateKey := generateTestKey(t)

		// create an account
		nonce := s.getNonce(t, enrollForRevoke.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, s.newAccountURL(enrollForRevoke.PathIdentifier), nil)
		_, _, resp1 := s.createAccount(t, enrollForRevoke.PathIdentifier, jwsBody)
		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		// revoke it directly in the DB
		var accountID uint
		err := s.DB.GetContext(t.Context(), &accountID, `SELECT id FROM acme_accounts WHERE acme_enrollment_id = ? ORDER BY id DESC LIMIT 1`, enrollForRevoke.ID)
		require.NoError(t, err)
		_, err = s.DB.ExecContext(t.Context(), `UPDATE acme_accounts SET revoked = 1 WHERE id = ?`, accountID)
		require.NoError(t, err)

		// try to create again with the same key — should get accountRevoked error
		nonce2 := resp1.Header.Get("Replay-Nonce")
		jwsBody2 := buildJWS(t, privateKey, nonce2, s.newAccountURL(enrollForRevoke.PathIdentifier), nil)
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
		privateKey := generateTestKey(t)
		nonce1 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody1 := buildJWS(t, privateKey, nonce1, s.newAccountURL(enrollValid.PathIdentifier), nil)
		acctResp1, _, resp1 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody1)
		require.Regexp(t, "/api/mdm/acme/"+enrollValid.PathIdentifier+`/accounts/\d+`, resp1.Header.Get("Location"))
		location1 := resp1.Header.Get("Location")

		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		// lookup with onlyReturnExisting (use the valid returned nonce)
		nonce2 := resp1.Header.Get("Replay-Nonce")
		jwsBody2 := buildJWS(t, privateKey, nonce2, s.newAccountURL(enrollValid.PathIdentifier), map[string]any{"onlyReturnExisting": true})
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

		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{"onlyReturnExisting": true}
		jwsBody := buildJWS(t, privateKey, nonce, s.newAccountURL(enrollValid.PathIdentifier), payload)
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
		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		badIdentifier := "no-such-identifier"
		jwsBody := buildJWS(t, privateKey, nonce, s.newAccountURL(badIdentifier), nil)
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
		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, s.newAccountURL(enrollRevoked.PathIdentifier), nil)
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

		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, s.newAccountURL(enrollExpired.PathIdentifier), nil)
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

		privateKey := generateTestKey(t)
		jwsBody := buildJWS(t, privateKey, "bad-nonce-value", s.newAccountURL(enrollValid.PathIdentifier), nil)
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

		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "", nil)
		_, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		// TODO: 500 is returned because errors in DecodeBody do not go through
		// the domain error encoder, it bypsases this in kithttp framework.
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

		privateKey := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody := buildJWS(t, privateKey, nonce, "http://example.com", nil)
		_, acmeErr, resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.NotNil(t, acmeErr)
		require.Contains(t, acmeErr.Type, "unauthorized")
		// nonce is generated because the error occurs in the endpoint handler (not decode)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Empty(t, resp.Header.Get("Location"))
	})
}
