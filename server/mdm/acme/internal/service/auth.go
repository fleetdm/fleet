package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
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
	string(jose.ES256),
	string(jose.ES384),
	string(jose.ES512),
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
	if createNewAccount != nil {
		// must have the JWK
		if message.Key == nil {
			return ctxerr.New(ctx, "missing JWK in JWS message")
		}
		// For Apple ACME purposes we only support ECDSA hardware-bound keys so validate the key is of the correct type
		// and the algorithm is of a proper type for the key (which also ensures it isn't none)
		_, ok := message.Key.Key.(*ecdsa.PublicKey)
		if !ok {
			return ctxerr.New(ctx, "JWK in JWS message is not an ECDSA public key")
		}
	}

	if otherRequest != nil {
		// must have the kid
		if message.KeyID == nil || *message.KeyID == "" {
			return ctxerr.New(ctx, "missing kid in JWS message")
		}
	}

	if !slices.Contains(acceptableSignatureAlgorithms[:], message.JWS.Signatures[0].Protected.Algorithm) {
		return ctxerr.New(ctx, "unsupported signature algorithm in JWS message")
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
		return ctxerr.New(ctx, "invalid url in JWS protected header")
	}

	// consume the nonce
	nonce := message.JWS.Signatures[0].Protected.Nonce
	nonceValid, err := s.nonces.Consume(ctx, nonce)
	if !nonceValid || err != nil {
		// if there is an error, it is a Redis/network issue, so keep it as a 500
		if err == nil {
			err = types.BadNonceError("")
		}
		return ctxerr.Wrapf(ctx, err, "invalid nonce in JWS message for identifier %s", message.Identifier)
	}

	// authenticate the enrollment identifier from the path
	enrollment, err := s.authenticateWithACMEEnrollment(ctx, message.Identifier)
	if err != nil {
		return err
	}

	webKeyToVerify := message.Key
	var account *types.Account
	if otherRequest != nil {
		accountID, err := accountIDFromKeyID(ctx, *message.KeyID, message.Identifier)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing account ID from key ID")
		}
		account, err = s.store.GetAccountByID(ctx, enrollment.ID, accountID)
		if err != nil {
			// TODO not found vs other errors, see RFC for how we should respond
			return ctxerr.Wrap(ctx, err, "fetching account by ID")
		}
		webKeyToVerify = &account.JSONWebKey
	}

	payload, err := message.JWS.Verify(webKeyToVerify)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "verifying JWS message for identifier: %s", message.Identifier)
	}

	var requestPayload any
	requestPayload = createNewAccount
	if otherRequest != nil {
		requestPayload = otherRequest
	}
	err = json.Unmarshal(payload, requestPayload)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "unmarshalling JWS payload into request for identifier: %s", message.Identifier)
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
	// TODO: return proper ACME errors for those validations...
	return s.commonAuthenticateMessage(ctx, message, request, nil)
}

func (s *Service) AuthenticateMessageFromAccount(ctx context.Context, message *api_http.JWSRequestContainer, request types.AccountAuthenticatedRequest) error {
	// TODO: return proper ACME errors for those validations...
	return s.commonAuthenticateMessage(ctx, message, nil, request)
}

func accountIDFromKeyID(ctx context.Context, keyID, pathIdentifier string) (uint, error) {
	// The key ID is the account URL, which should be in the format /api/mdm/acme/{identifier}/account/{accountID}
	// We can parse the account ID out of the URL to look up the account in the database
	urlParsed, err := url.Parse(keyID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "parsing key ID URL")
	}
	prefix := fmt.Sprintf("/api/mdm/acme/%s/account/", pathIdentifier)
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
