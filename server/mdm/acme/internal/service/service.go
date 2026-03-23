// Package service provides the service implementation for the ACME bounded context.
package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/go-jose/go-jose/v3"
)

// Service is the activity bounded context service implementation.
type Service struct {
	authz  platform_authz.Authorizer
	store  types.Datastore
	logger *slog.Logger
}

func getAcceptableSignatureAlgorithms() []string {
	return []string{
		string(jose.ES256),
		string(jose.ES384),
		string(jose.ES512),
	}
}

// NewService creates a new activity service.
func NewService(
	authz platform_authz.Authorizer,
	store types.Datastore,
	logger *slog.Logger,
) *Service {
	return &Service{
		authz:  authz,
		store:  store,
		logger: logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

func (s *Service) NewNonce(ctx context.Context, identifier string) (string, error) {
	panic("unimplemented")
}

func (s *Service) GetDirectory(ctx context.Context, identifier string) (*types.Directory, error) {
	panic("unimplemented")
}

func (s *Service) CreateAccount(ctx context.Context, account *types.Account, onlyReturnExisting bool) (*types.Account, error) {
	account, err := s.store.CreateAccount(ctx, account, onlyReturnExisting)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating account in datastore")
	}
	return account, nil
}

func (s *Service) CreateOrder(ctx context.Context, order *types.Order) (*types.Order, error) {
}

func (s *Service) validateAndConsumeNonce(ctx context.Context, message api_http.JWSRequestContainer) error {
	if message.JWS.Signatures[0].Protected.Nonce == "" {
		return ctxerr.New(ctx, "missing nonce in JWS message")
	}
	// TODO Check if the nonce is in redis and delete it if so, returning an error if it doesn't exist
	panic("unimplemented")
}

func (s *Service) AuthenticateNewAccountMessage(ctx context.Context, message api_http.JWSRequestContainer, request *api_http.CreateNewAccountRequest) error {
	// Validate the JWS message includes a JWK
	if message.Key == nil {
		// TODO: we should always get a key here
		return ctxerr.New(ctx, "missing JWK in JWS message")
	}
	// For Apple ACME purposes we only support ECDSA hardware-bound keys so validate the key is of the correct type
	// and the algorithm is of a proper type for the key(which also ensures it isn't none)
	_, ok := message.Key.Key.(*ecdsa.PublicKey)
	if !ok {
		return ctxerr.New(ctx, "JWK in JWS message is not an ECDSA public key")
	}
	switch message.JWS.Signatures[0].Protected.Algorithm {
	case string(jose.ES256), string(jose.ES384), string(jose.ES512):
		// All acceptable algorithms
	default:
		return ctxerr.New(ctx, "unsupported signature algorithm in JWS message")
	}
	if message.JWS.Signatures[0].Protected.Nonce == "" {
		return ctxerr.New(ctx, "missing nonce in JWS message")
	}
	// First fetch the enrollment associated with the identifier, 404 if not exists.
	enrollment, err := s.store.GetEnrollmentByIdentifier(ctx, message.Identifier)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching enrollment for identifier")
	}
	if enrollment.Revoked {
		// TODO err
	}
	if enrollment.NotValidAfter.Before(time.Now()) {
		// TODO err
	}
	err = s.validateAndConsumeNonce(ctx, message)
	if err != nil {
		// TODO make sure to return a new nonce
		return ctxerr.Wrap(ctx, err, "invalid nonce in JWS message", "identifier", message.Identifier)
	}
	payload, err := message.JWS.Verify(message.Key)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "verifying JWS message", "identifier", message.Identifier)
	}
	err = json.Unmarshal(payload, request)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling JWS payload into CreateNewAccountRequest", "identifier", message.Identifier)
	}
	request.JSONWebKey = message.Key
	request.Enrollment = enrollment
	return nil
}

func (s *Service) AuthenticateMessageFromAccount(ctx context.Context, message api_http.JWSRequestContainer, request types.AccountAuthenticatedRequest) error {
	if message.KeyID == nil || *message.KeyID == "" {
		// TODO: we should always get a key ID here
		return ctxerr.New(ctx, "missing JWK in JWS message")
	}
	if message.JWS.Signatures[0].Protected.Nonce == "" {
		return ctxerr.New(ctx, "missing nonce in JWS message")
	}
	switch message.JWS.Signatures[0].Protected.Algorithm {
	case string(jose.ES256), string(jose.ES384), string(jose.ES512):
		// All acceptable algorithms
	default:
		return ctxerr.New(ctx, "unsupported signature algorithm in JWS message")
	}
	// First fetch the enrollment associated with the identifier, 404 if not exists.
	enrollment, err := s.store.GetEnrollmentByIdentifier(ctx, message.Identifier)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching enrollment for identifier", "identifier", message.Identifier)
	}
	if enrollment.Revoked {
		// TODO err
	}
	if enrollment.NotValidAfter.Before(time.Now()) {
		// TODO err
	}
	err = s.validateAndConsumeNonce(ctx, message)
	if err != nil {
		// TODO make sure to return a new nonce
		return ctxerr.Wrap(ctx, err, "invalid nonce in JWS message", "identifier", message.Identifier)
	}
	accountID, err := accountIDFromKeyID(ctx, *message.KeyID, message.Identifier)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing account ID from key ID")
	}
	account, err := s.store.GetAccountByID(ctx, enrollment.ID, accountID)
	if err != nil {
		// TODO not found vs other errors
		return ctxerr.Wrap(ctx, err, "fetching account by ID")
	}
	payload, err := message.JWS.Verify(account.JSONWebKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "verifying JWS message", "identifier", message.Identifier)
	}
	err = json.Unmarshal(payload, request)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling JWS payload into request", "identifier", message.Identifier)
	}
	return nil
}

func accountIDFromKeyID(ctx context.Context, keyID, enrollmentID string) (uint, error) {
	// The key ID is the account URL, which should be in the format /api/mdm/acme/{identifier}/account/{accountID}
	// We can parse the account ID out of the URL to look up the account in the database
	urlParsed, err := url.Parse(keyID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "parsing key ID URL")
	}
	prefix := fmt.Sprintf("/api/mdm/acme/%s/account/", enrollmentID)
	if !strings.HasPrefix(urlParsed.Path, prefix) {
		return 0, ctxerr.New(ctx, "invalid key ID URL format")
	}
	accountIDStr := strings.TrimPrefix(urlParsed.Path, prefix)
	accountID, err := strconv.ParseUint(accountIDStr, 10, 64)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "parsing account ID from key ID URL")
	}
	return uint(accountID), nil
}
