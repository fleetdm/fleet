package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

// readBody reads and closes the response body, returning the raw bytes.
func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return body
}

// newAccountURL returns the full URL for the new_account endpoint.
func (s *integrationTestSuite) newAccountURL(pathIdentifier string) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/new_account", s.server.URL, pathIdentifier)
}

// parseJSON decodes the response body as JSON into dst.
func parseJSON(t *testing.T, body []byte, dst any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(body, dst))
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
	enrollValid := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollValid)

	enrollRevoked := &types.Enrollment{Revoked: true, NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollRevoked)

	enrollExpired := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	s.InsertACMEEnrollment(t, enrollExpired)

	t.Run("create new account", func(t *testing.T) {
		privateKey, jwk := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{}
		jwsBody := buildJWS(t, privateKey, jwk, nonce, s.newAccountURL(enrollValid.PathIdentifier), payload)
		resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)
		body := readBody(t, resp)

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))
		require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

		var acctResp createAccountResponse
		parseJSON(t, body, &acctResp)
		require.Equal(t, "valid", acctResp.Status)
		require.NotEmpty(t, acctResp.Orders)
		require.Contains(t, acctResp.Orders, "/orders")
	})

	t.Run("return existing account with same JWK", func(t *testing.T) {
		privateKey, jwk := generateTestKey(t)

		// create account
		nonce1 := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{}
		jwsBody1 := buildJWS(t, privateKey, jwk, nonce1, s.newAccountURL(enrollValid.PathIdentifier), payload)
		resp1 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody1)
		body1 := readBody(t, resp1)
		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		var acctResp1 createAccountResponse
		parseJSON(t, body1, &acctResp1)

		// create again with same key - should return existing
		nonce2 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody2 := buildJWS(t, privateKey, jwk, nonce2, s.newAccountURL(enrollValid.PathIdentifier), payload)
		resp2 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody2)
		body2 := readBody(t, resp2)
		require.Equal(t, http.StatusOK, resp2.StatusCode)

		var acctResp2 createAccountResponse
		parseJSON(t, body2, &acctResp2)
		require.Equal(t, "valid", acctResp2.Status)
		// same account, same orders URL
		require.Equal(t, acctResp1.Orders, acctResp2.Orders)
	})

	t.Run("onlyReturnExisting account exists", func(t *testing.T) {
		privateKey, jwk := generateTestKey(t)

		// create account first
		nonce1 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody1 := buildJWS(t, privateKey, jwk, nonce1, s.newAccountURL(enrollValid.PathIdentifier), map[string]any{})
		resp1 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody1)
		body1 := readBody(t, resp1)
		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		var acctResp1 createAccountResponse
		parseJSON(t, body1, &acctResp1)

		// lookup with onlyReturnExisting
		nonce2 := s.getNonce(t, enrollValid.PathIdentifier)
		jwsBody2 := buildJWS(t, privateKey, jwk, nonce2, s.newAccountURL(enrollValid.PathIdentifier), map[string]any{"onlyReturnExisting": true})
		resp2 := s.createAccount(t, enrollValid.PathIdentifier, jwsBody2)
		body2 := readBody(t, resp2)
		require.Equal(t, http.StatusOK, resp2.StatusCode)

		var acctResp2 createAccountResponse
		parseJSON(t, body2, &acctResp2)
		require.Equal(t, "valid", acctResp2.Status)
		require.Equal(t, acctResp1.Orders, acctResp2.Orders)
	})

	t.Run("onlyReturnExisting account does not exist", func(t *testing.T) {
		privateKey, jwk := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{"onlyReturnExisting": true}
		jwsBody := buildJWS(t, privateKey, jwk, nonce, s.newAccountURL(enrollValid.PathIdentifier), payload)
		resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)
		body := readBody(t, resp)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NotEmpty(t, resp.Header.Get("Replay-Nonce"))

		var errResp acmeErrorResponse
		parseJSON(t, body, &errResp)
		require.Equal(t, "urn:ietf:params:acme:error:accountDoesNotExist", errResp.Type)
	})

	t.Run("unknown identifier", func(t *testing.T) {
		// we need a valid enrollment to get a nonce, then use a bad identifier for the account request
		privateKey, jwk := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		badIdentifier := "no-such-identifier"
		payload := map[string]any{}
		jwsBody := buildJWS(t, privateKey, jwk, nonce, s.newAccountURL(badIdentifier), payload)
		resp := s.createAccount(t, badIdentifier, jwsBody)
		_ = readBody(t, resp)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("revoked enrollment", func(t *testing.T) {
		// get nonce from valid enrollment, then try to create account on revoked
		privateKey, jwk := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{}
		jwsBody := buildJWS(t, privateKey, jwk, nonce, s.newAccountURL(enrollRevoked.PathIdentifier), payload)
		resp := s.createAccount(t, enrollRevoked.PathIdentifier, jwsBody)
		_ = readBody(t, resp)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("expired enrollment", func(t *testing.T) {
		privateKey, jwk := generateTestKey(t)
		nonce := s.getNonce(t, enrollValid.PathIdentifier)
		payload := map[string]any{}
		jwsBody := buildJWS(t, privateKey, jwk, nonce, s.newAccountURL(enrollExpired.PathIdentifier), payload)
		resp := s.createAccount(t, enrollExpired.PathIdentifier, jwsBody)
		_ = readBody(t, resp)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("invalid nonce", func(t *testing.T) {
		privateKey, jwk := generateTestKey(t)
		payload := map[string]any{}
		jwsBody := buildJWS(t, privateKey, jwk, "bad-nonce-value", s.newAccountURL(enrollValid.PathIdentifier), payload)
		resp := s.createAccount(t, enrollValid.PathIdentifier, jwsBody)
		body := readBody(t, resp)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errResp acmeErrorResponse
		parseJSON(t, body, &errResp)
		require.Equal(t, "urn:ietf:params:acme:error:badNonce", errResp.Type)
	})
}
