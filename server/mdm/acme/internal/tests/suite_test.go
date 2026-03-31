package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.step.sm/crypto/jose"
)

// integrationTestSuite holds all dependencies for integration tests.
type integrationTestSuite struct {
	*testutils.TestDB
	ds     *mysql.Datastore
	server *httptest.Server
}

// setupIntegrationTest creates a new test suite with a real database and HTTP server.
func setupIntegrationTest(t *testing.T) *integrationTestSuite {
	t.Helper()

	tdb := testutils.SetupTestDB(t, "acme_integration")
	pool := redistest.SetupRedis(t, "acme_integration", false, false, false)
	ds := mysql.NewDatastore(tdb.Conns(), tdb.Logger)

	// Create mocks
	providers := newMockDataProviders(&fleet.AppConfig{
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://example.com", // will update with actual test server URL after it is started
		},
	},
		// not needed for current tests, can implement if needed for future tests
		acme.CSRSignerFunc(func(ctx context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error) {
			return &x509.Certificate{}, nil
		}),
	)

	// Create service
	svc := service.NewService(ds, pool, providers, tdb.Logger)

	// Create router with routes
	router := mux.NewRouter()
	routesFn := service.GetRoutes(svc)
	routesFn(router, nil)

	// Create test server
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	ac, err := providers.AppConfig(t.Context())
	require.NoError(t, err)
	ac.ServerSettings.ServerURL = server.URL

	return &integrationTestSuite{
		TestDB: tdb,
		ds:     ds,
		server: server,
	}
}

// truncateTables clears all test data between tests.
func (s *integrationTestSuite) truncateTables(t *testing.T) {
	t.Helper()
	s.TruncateTables(t)
}

func drainAndCloseBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// newNonce makes an HTTP request to new nonce endpoint and returns the parsed response and the raw response.
func (s *integrationTestSuite) newNonce(t *testing.T, httpMethod, pathIdentifier string) (*api_http.GetNewNonceResponse, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/new_nonce", pathIdentifier) //nolint:gosec // test server URL is safe
	req, err := http.NewRequest(httpMethod, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer drainAndCloseBody(resp)

	result := &api_http.GetNewNonceResponse{
		HTTPMethod: resp.Request.Method,
	}
	return result, resp
}

// getDirectory makes an HTTP request to get directory endpoint and returns the parsed response and the raw response.
func (s *integrationTestSuite) getDirectory(t *testing.T, httpMethod, pathIdentifier string) (*api_http.GetDirectoryResponse, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/directory", pathIdentifier) //nolint:gosec // test server URL is safe
	req, err := http.NewRequest(httpMethod, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer drainAndCloseBody(resp)

	if resp.StatusCode >= 300 {
		return nil, resp
	}

	var result api_http.GetDirectoryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return &result, resp
}

// staticNonce implements jose.NonceSource with a fixed nonce value.
type staticNonce struct {
	nonce string
}

func (s staticNonce) Nonce() (string, error) {
	return s.nonce, nil
}

// generateTestKey generates an ECDSA P-256 key pair and returns the private key and public JWK.
func generateTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

// getNonce obtains a fresh nonce from the new_nonce endpoint for the given enrollment.
func (s *integrationTestSuite) getNonce(t *testing.T, pathIdentifier string) string {
	t.Helper()
	_, resp := s.newNonce(t, http.MethodGet, pathIdentifier)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	nonce := resp.Header.Get("Replay-Nonce")
	require.NotEmpty(t, nonce)
	return nonce
}

// buildJWS constructs a JWS in flattened JSON serialization. When accountURL is
// empty, the JWK is embedded in the header (for new-account requests). When
// accountURL is set, it is used as the KeyID instead (for account-authenticated
// requests like new-order).
func buildJWS(t *testing.T, privateKey *ecdsa.PrivateKey, nonce, accountURL, endpointURL string, payload any) []byte {
	t.Helper()

	opts := &jose.SignerOptions{
		NonceSource: staticNonce{nonce: nonce},
		ExtraHeaders: map[jose.HeaderKey]any{
			"url": endpointURL,
		},
	}
	if accountURL == "" {
		opts.EmbedJWK = true
	} else {
		opts.ExtraHeaders["kid"] = accountURL
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.ES256, Key: privateKey},
		opts,
	)
	require.NoError(t, err)

	var payloadBytes []byte
	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		require.NoError(t, err)
	} else {
		payloadBytes = []byte("{}")
	}

	jws, err := signer.Sign(payloadBytes)
	require.NoError(t, err)

	return []byte(jws.FullSerialize())
}

// createAccount POSTs a JWS body to the new_account endpoint and returns the
// account response or acme error and the raw response.
func (s *integrationTestSuite) createAccount(t *testing.T, pathIdentifier string, jwsBody []byte) (*types.AccountResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/new_account", pathIdentifier) //nolint:gosec // test server URL is safe
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jwsBody))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer drainAndCloseBody(resp)

	if resp.StatusCode >= 300 {
		var acmeErr types.ACMEError
		if err := json.NewDecoder(resp.Body).Decode(&acmeErr); err == nil && acmeErr.Type != "" {
			return nil, &acmeErr, resp
		}
		return nil, nil, resp
	}

	var result types.AccountResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return &result, nil, resp
}

// newOrderURL returns the full URL for the new_order endpoint.
func (s *integrationTestSuite) newOrderURL(pathIdentifier string) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/new_order", s.server.URL, pathIdentifier)
}

// createAccountForOrder is a convenience helper that creates an account for an enrollment,
// returning the private key, account URL, and a fresh nonce for subsequent requests.
func (s *integrationTestSuite) createAccountForOrder(t *testing.T, enrollment *types.Enrollment) (*ecdsa.PrivateKey, string, string) {
	t.Helper()
	privateKey := generateTestKey(t)
	nonce := s.getNonce(t, enrollment.PathIdentifier)
	jwsBody := buildJWS(t, privateKey, nonce, "", s.newAccountURL(enrollment.PathIdentifier), nil)
	_, _, resp := s.createAccount(t, enrollment.PathIdentifier, jwsBody)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	accountURL := resp.Header.Get("Location")
	require.NotEmpty(t, accountURL)
	nextNonce := resp.Header.Get("Replay-Nonce")
	require.NotEmpty(t, nextNonce)
	return privateKey, accountURL, nextNonce
}

// createOrder POSTs a JWS body to the new_order endpoint and returns the
// order response or acme error and the raw response.
func (s *integrationTestSuite) createOrder(t *testing.T, pathIdentifier string, jwsBody []byte) (*types.OrderResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/new_order", pathIdentifier) //nolint:gosec // test server URL is safe
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jwsBody))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer drainAndCloseBody(resp)

	if resp.StatusCode >= 300 {
		var acmeErr types.ACMEError
		if err := json.NewDecoder(resp.Body).Decode(&acmeErr); err == nil && acmeErr.Type != "" {
			return nil, &acmeErr, resp
		}
		return nil, nil, resp
	}

	var result types.OrderResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return &result, nil, resp
}

func (s *integrationTestSuite) getAuthorization(t *testing.T, authUrl string, jwsBody []byte) (*types.AuthorizationResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, authUrl, bytes.NewReader(jwsBody))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer drainAndCloseBody(resp)

	if resp.StatusCode >= 300 {
		var acmeErr types.ACMEError
		if err := json.NewDecoder(resp.Body).Decode(&acmeErr); err == nil && acmeErr.Type != "" {
			return nil, &acmeErr, resp
		}
		return nil, nil, resp
	}

	var result types.AuthorizationResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return &result, nil, resp
}
