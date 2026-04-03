package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	"go.step.sm/crypto/jose"
)

var acceptableSignatureAlgorithms = [...]string{
	jose.ES256,
	jose.ES384,
	jose.ES512,
}

func (s *Service) authenticateWithACMEEnrollment(ctx context.Context, identifier string) (*types.Enrollment, error) {
	enrollment, err := s.store.GetACMEEnrollment(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if !enrollment.IsValid() {
		err = types.EnrollmentNotFoundError(fmt.Sprintf("ACME enrollment with path identifier %s not found", identifier))
		return nil, ctxerr.Wrap(ctx, err)
	}
	return enrollment, nil
}

// common authentication logic for both AuthenticateNewAccountMessage and AuthenticateMessageFromAccount, only
// one of createNewAccount or otherRequest must be non-nil.
func (s *Service) commonAuthenticateMessage(ctx context.Context, message *api_http.JWSRequestContainer, createNewAccount *api_http.CreateNewAccountRequest, otherRequest types.AccountAuthenticatedRequest) error {
	var err error

	// consume the nonce as first validation
	nonce := message.JWS.Signatures[0].Protected.Nonce
	nonceValid, err := s.nonces.Consume(ctx, nonce)
	if !nonceValid || err != nil {
		// if there is an error, it is a Redis/network issue, so keep it as a 500
		if err == nil {
			err = types.BadNonceError("")
		}
		return ctxerr.Wrapf(ctx, err, "invalid nonce in JWS message for identifier %s", message.Identifier)
	}

	if createNewAccount != nil {
		// must have the JWK
		if message.Key == nil {
			err = types.UnauthorizedError("missing JWK in JWS message for new account creation")
			return ctxerr.Wrap(ctx, err)
		}
		// For Apple ACME purposes we only support ECDSA hardware-bound keys so validate the key is of the correct type
		// and the algorithm is of a proper type for the key (which also ensures it isn't none)
		_, ok := message.Key.Key.(*ecdsa.PublicKey)
		if !ok {
			err = types.BadPublicKeyError("JWK in JWS message for new account creation is not an ECDSA public key")
			return ctxerr.Wrap(ctx, err)
		}
	}

	if otherRequest != nil {
		// must have the kid
		if message.KeyID == nil || *message.KeyID == "" {
			err = types.UnauthorizedError("missing kid in JWS message for account-authenticated request")
			return ctxerr.Wrap(ctx, err)
		}
	}

	if !slices.Contains(acceptableSignatureAlgorithms[:], message.JWS.Signatures[0].Protected.Algorithm) {
		err = types.BadSignatureAlgorithmError(fmt.Sprintf("unsupported signature algorithm %s in JWS message", message.JWS.Signatures[0].Protected.Algorithm))
		return ctxerr.Wrap(ctx, err)
	}

	// "url" field validation: https://datatracker.ietf.org/doc/html/rfc8555/#section-6.4.1
	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return ctxerr.New(ctx, "get base ACME URL")
	}
	expectedURL, err := commonmdm.ResolveURL(baseURL, message.HTTPPath, true)
	if err != nil {
		return ctxerr.New(ctx, "get expected ACME URL")
	}
	if message.JWSHeaderURL != expectedURL {
		err = types.UnauthorizedError("invalid url in JWS protected header")
		return ctxerr.Wrap(ctx, err)
	}

	// authenticate the enrollment identifier from the path
	enrollment, err := s.authenticateWithACMEEnrollment(ctx, message.Identifier)
	if err != nil {
		return err
	}

	webKeyToVerify := message.Key
	var account *types.Account
	if otherRequest != nil {
		accountID, err := s.accountIDFromKeyID(ctx, *message.KeyID, message.Identifier)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing account ID from key ID")
		}
		account, err = s.store.GetAccountByID(ctx, enrollment.ID, accountID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "fetching account by ID")
		}
		webKeyToVerify = &account.JSONWebKey
	}

	payload, err := message.JWS.Verify(webKeyToVerify)
	if err != nil {
		err = types.UnauthorizedError(err.Error()) // I think it's safe to return the error as details here?
		return ctxerr.Wrap(ctx, err)
	}
	if message.PostAsGet && len(payload) != 0 {
		err = types.MalformedError("payload must be empty for POST-as-GET requests")
		return ctxerr.Wrap(ctx, err)
	}

	var requestPayload any
	requestPayload = createNewAccount
	if otherRequest != nil {
		requestPayload = otherRequest
	}
	// From the RFC, for a POST-as-GET request, the payload is an empty string (absent),
	// which would fail to unmarshal into an object, so we check it explicitly.
	if len(payload) != 0 {
		err = json.Unmarshal(payload, requestPayload)
		if err != nil {
			err = types.MalformedError(fmt.Sprintf("Failed to unmarshal JWS payload: %v", err))
			return ctxerr.Wrap(ctx, err)
		}
	}

	if createNewAccount != nil {
		createNewAccount.JSONWebKey = message.Key
		createNewAccount.Enrollment = enrollment
	}
	if otherRequest != nil {
		otherRequest.SetEnrollmentAndAccount(enrollment, account)
	}
	return nil
}

func (s *Service) AuthenticateNewAccountMessage(ctx context.Context, message *api_http.JWSRequestContainer, request *api_http.CreateNewAccountRequest) error {
	return s.commonAuthenticateMessage(ctx, message, request, nil)
}

func (s *Service) AuthenticateMessageFromAccount(ctx context.Context, message *api_http.JWSRequestContainer, request types.AccountAuthenticatedRequest) error {
	return s.commonAuthenticateMessage(ctx, message, nil, request)
}

func (s *Service) accountIDFromKeyID(ctx context.Context, keyID, pathIdentifier string) (uint, error) {
	// The key ID is the account URL, which should be in the format /api/mdm/acme/{identifier}/accounts/{accountID}
	// We can parse the account ID out of the URL to look up the account in the database
	urlParsed, err := url.Parse(keyID)
	if err != nil {
		err = types.UnauthorizedError("Invalid key ID URL")
		return 0, ctxerr.Wrap(ctx, err)
	}

	expectedURL, err := s.getACMEURL(ctx, pathIdentifier, "accounts")
	if err != nil {
		// this is not an ACME error, it's a server error
		return 0, ctxerr.Wrap(ctx, err, "getting expected account URL")
	}
	expectedParsed, err := url.Parse(expectedURL)
	if err != nil {
		// same here, not an error for a client-provided value
		return 0, ctxerr.Wrap(ctx, err, "parsing expected account URL")
	}

	prefix := expectedParsed.Path + "/"
	if !strings.HasPrefix(urlParsed.Path, prefix) {
		err = types.UnauthorizedError("Invalid key ID URL")
		return 0, ctxerr.Wrap(ctx, err)
	}
	accountIDStr := strings.TrimPrefix(urlParsed.Path, prefix)
	accountID, err := strconv.ParseUint(accountIDStr, 10, 64)
	if err != nil {
		err = types.UnauthorizedError("Invalid key ID URL")
		return 0, ctxerr.Wrap(ctx, err)
	}

	if accountID > uint64(math.MaxUint) {
		err = types.UnauthorizedError("Invalid key ID URL")
		return 0, ctxerr.Wrap(ctx, err)
	}

	return uint(accountID), nil
}
