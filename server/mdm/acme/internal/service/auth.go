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
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"go.step.sm/crypto/jose"
)

var acceptableSignatureAlgorithms = [...]string{
	string(jose.ES256),
	string(jose.ES384),
	string(jose.ES512),
}

func (s *Service) authenticateWithACMEEnrollment(ctx context.Context, identifier string) (*types.ACMEEnrollment, error) {
	enrollment, err := s.store.GetACMEEnrollment(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if !enrollment.IsValid() {
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("ACME enrollment").WithName(identifier))
	}
	return enrollment, nil
}

func (s *Service) AuthenticateNewAccountMessage(ctx context.Context, message *api_http.JWSRequestContainer, request *api_http.CreateNewAccountRequest) error {
	// Validate the JWS message includes a JWK
	if message.Key == nil {
		return ctxerr.New(ctx, "missing JWK in JWS message")
	}
	// For Apple ACME purposes we only support ECDSA hardware-bound keys so validate the key is of the correct type
	// and the algorithm is of a proper type for the key (which also ensures it isn't none)
	_, ok := message.Key.Key.(*ecdsa.PublicKey)
	if !ok {
		return ctxerr.New(ctx, "JWK in JWS message is not an ECDSA public key")
	}
	if !slices.Contains(acceptableSignatureAlgorithms[:], message.JWS.Signatures[0].Protected.Algorithm) {
		return ctxerr.New(ctx, "unsupported signature algorithm in JWS message")
	}

	// consume the nonce
	nonce := message.JWS.Signatures[0].Protected.Nonce
	nonceValid, err := s.nonces.Consume(ctx, nonce)
	if !nonceValid || err != nil {
		// TODO(mna): we should return a new nonce here as per:
		// https://datatracker.ietf.org/doc/html/rfc8555/#section-6.5
		// When a server rejects a request because its nonce value was
		// unacceptable (or not present), it MUST provide HTTP status code 400
		// (Bad Request), and indicate the ACME error type
		// "urn:ietf:params:acme:error:badNonce".  An error response with the
		// "badNonce" error type MUST include a Replay-Nonce header field with a
		// fresh nonce that the server will accept in a retry of the original
		// query (and possibly in other requests, according to the server's
		// nonce scoping policy).
		err = &platform_http.BadRequestError{
			Message:     "badNonce", // TODO(mna): some ACME-specific errors so we render them as expected by the RFC
			InternalErr: err,
		}
		return ctxerr.Wrapf(ctx, err, "invalid nonce in JWS message for identifier %s", message.Identifier)
	}

	// authenticate the enrollment identifier from the path
	enrollment, err := s.authenticateWithACMEEnrollment(ctx, message.Identifier)
	if err != nil {
		return err
	}

	payload, err := message.JWS.Verify(message.Key)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "verifying JWS message", "identifier", message.Identifier)
	}
	err = json.Unmarshal(payload, request)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "unmarshalling JWS payload into CreateNewAccountRequest with identifier: %s", message.Identifier)
	}

	request.JSONWebKey = message.Key
	request.Enrollment = enrollment
	return nil
}

func (s *Service) AuthenticateMessageFromAccount(ctx context.Context, message *api_http.JWSRequestContainer, request types.AccountAuthenticatedRequest) error {
	if message.KeyID == nil || *message.KeyID == "" {
		return ctxerr.New(ctx, "missing kid in JWS message")
	}
	if !slices.Contains(acceptableSignatureAlgorithms[:], message.JWS.Signatures[0].Protected.Algorithm) {
		return ctxerr.New(ctx, "unsupported signature algorithm in JWS message")
	}

	// consume the nonce
	nonce := message.JWS.Signatures[0].Protected.Nonce
	nonceValid, err := s.nonces.Consume(ctx, nonce)
	if !nonceValid || err != nil {
		// TODO(mna): we should return a new nonce here as per:
		// https://datatracker.ietf.org/doc/html/rfc8555/#section-6.5
		// When a server rejects a request because its nonce value was
		// unacceptable (or not present), it MUST provide HTTP status code 400
		// (Bad Request), and indicate the ACME error type
		// "urn:ietf:params:acme:error:badNonce".  An error response with the
		// "badNonce" error type MUST include a Replay-Nonce header field with a
		// fresh nonce that the server will accept in a retry of the original
		// query (and possibly in other requests, according to the server's
		// nonce scoping policy).
		err = &platform_http.BadRequestError{
			Message:     "badNonce", // TODO(mna): some ACME-specific errors so we render them as expected by the RFC
			InternalErr: err,
		}
		return ctxerr.Wrapf(ctx, err, "invalid nonce in JWS message for identifier %s", message.Identifier)
	}

	// authenticate the enrollment identifier from the path
	enrollment, err := s.authenticateWithACMEEnrollment(ctx, message.Identifier)
	if err != nil {
		return err
	}

	accountID, err := accountIDFromKeyID(ctx, *message.KeyID, message.Identifier)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing account ID from key ID")
	}
	account, err := s.store.GetAccountByID(ctx, enrollment.ID, accountID)
	if err != nil {
		// TODO not found vs other errors, see RFC for how we should respond
		return ctxerr.Wrap(ctx, err, "fetching account by ID")
	}

	payload, err := message.JWS.Verify(account.JSONWebKey)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "verifying JWS message for identifier: %s", message.Identifier)
	}
	err = json.Unmarshal(payload, request)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "unmarshalling JWS payload into request for identifier: %s", message.Identifier)
	}

	return nil
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
