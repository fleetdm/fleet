package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/bootstrap"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testhelpers"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.step.sm/crypto/jose"
)

// integrationTestSuite holds all dependencies for integration tests.
type integrationTestSuite struct {
	*testutils.TestDB
	ds     *mysql.Datastore
	server *httptest.Server

	attestCA    *x509.Certificate
	attestCAKey *ecdsa.PrivateKey
}

// setupIntegrationTest creates a new test suite with a real database and HTTP server.
func setupIntegrationTest(t *testing.T) *integrationTestSuite {
	t.Helper()

	tdb := testutils.SetupTestDB(t, "acme_integration")
	pool := redistest.SetupRedis(t, "acme_integration", false, false, false)
	ds := mysql.NewDatastore(tdb.Conns(), tdb.Logger)
	cert, key, err := testhelpers.GenerateTestAttestationCA()
	require.NoError(t, err)
	rootPool := x509.NewCertPool()
	rootPool.AddCert(cert)

	// Create mocks
	providers := newMockDataProviders(
		"https://example.com", // will update with actual test server URL after it is started
		acme.CSRSignerFunc(func(ctx context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error) {
			res, err := tdb.DB.DB.Exec(`INSERT INTO identity_serials () VALUES ()`) // insert a row to get an auto-incremented ID for the cert serial number
			require.NoError(t, err)
			serialID, err := res.LastInsertId()
			require.NoError(t, err)
			_, err = tdb.DB.DB.Exec(`INSERT INTO identity_certificates (serial, not_valid_before, not_valid_after, certificate_pem) VALUES (?, NOW(), NOW(), ?)`, serialID, fmt.Appendf(nil, "-----BEGIN CERTIFICATE-----\nmock-cert-%d\n-----END CERTIFICATE-----", serialID))
			require.NoError(t, err)
			return &x509.Certificate{
				SerialNumber: big.NewInt(serialID),
				Raw:          []byte("mock-cert"),
			}, nil
		}),
		[]byte("-----BEGIN CERTIFICATE-----\nroot\n-----END CERTIFICATE-----"),
	)

	opts := bootstrap.WithTestAppleRootCAs(rootPool)
	// Create service
	svc := service.NewService(ds, pool, providers, tdb.Logger, opts)

	// Create router with routes
	router := mux.NewRouter()
	authMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint { return next } // no-op auth middleware for testing
	routesFn := service.GetRoutes(svc, authMiddleware)
	routesFn(router, nil)

	// Create test server
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	providers.serverURL = server.URL

	return &integrationTestSuite{
		TestDB:      tdb,
		ds:          ds,
		server:      server,
		attestCA:    cert,
		attestCAKey: key,
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

// doACMERequest is a generic helper that makes an HTTP request, decodes the
// response into T on success, or into an ACMEError on failure (status >= 300).
func doACMERequest[T any](t *testing.T, method, url string, body []byte) (*T, *types.ACMEError, *http.Response) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader) //nolint:gosec // test server URL is safe
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

	var result T
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return &result, nil, resp
}

// getDirectory makes an HTTP request to get directory endpoint and returns the parsed response and the raw response.
func (s *integrationTestSuite) getDirectory(t *testing.T, httpMethod, pathIdentifier string) (*api_http.GetDirectoryResponse, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/directory", pathIdentifier) //nolint:gosec // test server URL is safe
	result, _, resp := doACMERequest[api_http.GetDirectoryResponse](t, httpMethod, url, nil)
	return result, resp
}

// staticNonce implements jose.NonceSource with a fixed nonce value.
type staticNonce struct {
	nonce string
}

func (s staticNonce) Nonce() (string, error) {
	return s.nonce, nil
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
		// as per the RFC:
		// > [when doing a POST-as-GET] the "payload" field of the
		// > JWS object MUST be present and set to the empty string
		// > [...]  a zero-length (and thus non-JSON) payload
		payloadBytes = []byte("")
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
	return doACMERequest[types.AccountResponse](t, http.MethodPost, url, jwsBody)
}

// newAccountURL returns the full URL for the new_account endpoint.
func (s *integrationTestSuite) newAccountURL(pathIdentifier string) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/new_account", s.server.URL, pathIdentifier)
}

// newOrderURL returns the full URL for the new_order endpoint.
func (s *integrationTestSuite) newOrderURL(pathIdentifier string) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/new_order", s.server.URL, pathIdentifier)
}

// createAccountForOrder is a convenience helper that creates an account for an enrollment,
// returning the private key, account URL, and a fresh nonce for subsequent requests.
func (s *integrationTestSuite) createAccountForOrder(t *testing.T, enrollment *types.Enrollment) (*ecdsa.PrivateKey, string, string) {
	t.Helper()
	privateKey, err := testhelpers.GenerateTestKey()
	require.NoError(t, err)
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

// createOrderForGet is a convenience helper that creates an account and order for an enrollment,
// returning the private key, account URL, order response, and a fresh nonce for subsequent requests.
func (s *integrationTestSuite) createOrderForGet(t *testing.T, enroll *types.Enrollment) (*ecdsa.PrivateKey, string, *types.OrderResponse, string) {
	t.Helper()
	privateKey, accountURL, nonce := s.createAccountForOrder(t, enroll)

	payload := map[string]any{
		"identifiers": []map[string]string{
			{"type": "permanent-identifier", "value": enroll.HostIdentifier},
		},
	}
	jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
	orderResp, _, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	nextNonce := resp.Header.Get("Replay-Nonce")
	require.NotEmpty(t, nextNonce)
	return privateKey, accountURL, orderResp, nextNonce
}

// createOrderForChallenge is a convenience helper that creates account+order+fetches
// authorization, returning: privateKey, accountURL, challengeURL, challengeToken, nonce.
func (s *integrationTestSuite) createOrderForChallenge(t *testing.T, enroll *types.Enrollment) (privateKey *ecdsa.PrivateKey, accountURL, challengeURL, challengeToken, nonce string) {
	t.Helper()
	privateKey, accountURL, orderResp, nonce := s.createOrderForGet(t, enroll)

	require.Len(t, orderResp.Authorizations, 1)
	authURL := orderResp.Authorizations[0]

	authResp, _, resp := s.getAuthorization(t, authURL, buildJWS(t, privateKey, nonce, accountURL, authURL, nil))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, authResp.Challenges, 1)
	challenge := authResp.Challenges[0]
	nonce = resp.Header.Get("Replay-Nonce")
	require.NotEmpty(t, nonce)
	return privateKey, accountURL, challenge.URL, challenge.Token, nonce
}

// getOrderURL returns the full URL for the get order endpoint.
func (s *integrationTestSuite) getOrderURL(pathIdentifier string, orderID uint) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/orders/%d", s.server.URL, pathIdentifier, orderID)
}

// getOrder POSTs a JWS body to the order endpoint and returns the
// order response or acme error and the raw response.
func (s *integrationTestSuite) getOrder(t *testing.T, pathIdentifier string, orderID uint, jwsBody []byte) (*types.OrderResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/orders/%d", pathIdentifier, orderID) //nolint:gosec // test server URL is safe
	return doACMERequest[types.OrderResponse](t, http.MethodPost, url, jwsBody)
}

// createOrder POSTs a JWS body to the new_order endpoint and returns the
// order response or acme error and the raw response.
func (s *integrationTestSuite) createOrder(t *testing.T, pathIdentifier string, jwsBody []byte) (*types.OrderResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/new_order", pathIdentifier) //nolint:gosec // test server URL is safe
	return doACMERequest[types.OrderResponse](t, http.MethodPost, url, jwsBody)
}

// listOrdersURL returns the full URL for the list orders endpoint.
func (s *integrationTestSuite) listOrdersURL(pathIdentifier string, accountID uint) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/accounts/%d/orders", s.server.URL, pathIdentifier, accountID)
}

// listOrders POSTs a JWS body to the list orders endpoint and returns the
// list orders response or acme error and the raw response.
func (s *integrationTestSuite) listOrders(t *testing.T, pathIdentifier string, accountID uint, jwsBody []byte) (*api_http.ListOrdersResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	url := s.listOrdersURL(pathIdentifier, accountID)
	return doACMERequest[api_http.ListOrdersResponse](t, http.MethodPost, url, jwsBody)
}

// getCertificateURL returns the full URL for the get certificate endpoint.
func (s *integrationTestSuite) getCertificateURL(pathIdentifier string, orderID uint) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/orders/%d/certificate", s.server.URL, pathIdentifier, orderID)
}

// getCertificate POSTs a JWS body to the certificate endpoint and returns the
// PEM certificate chain (on success) or an ACME error (on failure) and the raw response.
// Unlike other helpers, the success response is raw PEM, not JSON.
func (s *integrationTestSuite) getCertificate(t *testing.T, pathIdentifier string, orderID uint, jwsBody []byte) (string, *types.ACMEError, *http.Response) {
	t.Helper()
	url := s.getCertificateURL(pathIdentifier, orderID)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jwsBody)) //nolint:gosec // test server URL is safe
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)

	if resp.StatusCode >= 300 {
		var acmeErr types.ACMEError
		if err := json.Unmarshal(body, &acmeErr); err == nil && acmeErr.Type != "" {
			return "", &acmeErr, resp
		}
		return "", nil, resp
	}

	return string(body), nil, resp
}

// finalizeOrderWithCert forces an order to finalized+valid state and inserts a
// linked certificate in the database. This is useful for setting up the state
// needed by the get certificate endpoint and any other test that needs a
// finalized order with a valid certificate.
func (s *integrationTestSuite) finalizeOrderWithCert(t *testing.T, orderID uint, certSerial uint64, certPEM string) {
	t.Helper()
	ctx := t.Context()

	_, err := s.DB.ExecContext(ctx,
		`UPDATE acme_orders SET finalized = 1, status = 'valid' WHERE id = ?`, orderID)
	require.NoError(t, err)

	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO identity_serials (serial) VALUES (?)`, certSerial)
	require.NoError(t, err)

	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO identity_certificates (serial, not_valid_before, not_valid_after, certificate_pem, revoked)
		VALUES (?, NOW(), DATE_ADD(NOW(), INTERVAL 1 YEAR), ?, ?)
	`, certSerial, certPEM, false)
	require.NoError(t, err)

	_, err = s.DB.ExecContext(ctx,
		`UPDATE acme_orders SET issued_certificate_serial = ? WHERE id = ?`, certSerial, orderID)
	require.NoError(t, err)
}

// parseAccountID extracts the numeric account ID from an account URL like
// ".../accounts/123" or ".../accounts/123/orders".
func parseAccountID(t *testing.T, accountURL string) uint {
	t.Helper()
	// strip trailing "/orders" if present
	u := strings.TrimSuffix(accountURL, "/orders")
	parts := strings.Split(u, "/")
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseUint(idStr, 10, 64)
	require.NoError(t, err)
	return uint(id)
}

// finalizeOrderURL returns the full URL for the finalize endpoint of a given order.
func (s *integrationTestSuite) finalizeOrderURL(pathIdentifier string, orderID uint) string {
	return fmt.Sprintf("%s/api/mdm/acme/%s/orders/%d/finalize", s.server.URL, pathIdentifier, orderID)
}

// finalizeOrder POSTs a JWS body to the finalize endpoint and returns the
// order response or acme error and the raw response.
func (s *integrationTestSuite) finalizeOrder(t *testing.T, finalizeURL string, jwsBody []byte) (*types.OrderResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	return doACMERequest[types.OrderResponse](t, http.MethodPost, finalizeURL, jwsBody)
}

// createOrderForFinalize is a convenience helper that creates an enrollment, account, and order,
// returning everything needed to test the finalize endpoint.
func (s *integrationTestSuite) createOrderForFinalize(t *testing.T) (enroll *types.Enrollment, privateKey *ecdsa.PrivateKey, accountURL string, orderResp *types.OrderResponse, nonce string) {
	t.Helper()
	enroll = &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	s.InsertACMEEnrollment(t, enroll)
	privateKey, accountURL, nonce = s.createAccountForOrder(t, enroll)

	payload := map[string]any{
		"identifiers": []map[string]string{
			{"type": "permanent-identifier", "value": enroll.HostIdentifier},
		},
	}
	jwsBody := buildJWS(t, privateKey, nonce, accountURL, s.newOrderURL(enroll.PathIdentifier), payload)
	orderResp, acmeErr, resp := s.createOrder(t, enroll.PathIdentifier, jwsBody)
	require.Nil(t, acmeErr)
	require.NotNil(t, orderResp)
	nonce = resp.Header.Get("Replay-Nonce")
	return enroll, privateKey, accountURL, orderResp, nonce
}

// makeOrderReady transitions the order's authorization and challenge to valid and the order to ready via direct DB updates.
func (s *integrationTestSuite) makeOrderReady(t *testing.T, orderID uint) {
	t.Helper()
	ctx := t.Context()
	_, err := s.DB.ExecContext(ctx, `UPDATE acme_challenges SET status = 'valid' WHERE acme_authorization_id IN (SELECT id FROM acme_authorizations WHERE acme_order_id = ?)`, orderID)
	require.NoError(t, err)
	_, err = s.DB.ExecContext(ctx, `UPDATE acme_authorizations SET status = 'valid' WHERE acme_order_id = ?`, orderID)
	require.NoError(t, err)
	_, err = s.DB.ExecContext(ctx, `UPDATE acme_orders SET status = 'ready' WHERE id = ?`, orderID)
	require.NoError(t, err)
}

func (s *integrationTestSuite) getAuthorization(t *testing.T, authUrl string, jwsBody []byte) (*api_http.GetAuthorizationResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	return doACMERequest[api_http.GetAuthorizationResponse](t, http.MethodPost, authUrl, jwsBody)
}

func (s *integrationTestSuite) doChallenge(t *testing.T, challengeURL string, jwsBody []byte) (*api_http.DoChallengeResponse, *types.ACMEError, *http.Response) {
	t.Helper()
	return doACMERequest[api_http.DoChallengeResponse](t, http.MethodPost, challengeURL, jwsBody)
}
