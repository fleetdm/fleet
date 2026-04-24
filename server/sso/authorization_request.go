package sso

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const cacheLifetimeSeconds = uint(300) // in seconds (5 minutes)

func getDestinationURL(idpMetadata *saml.EntityDescriptor) (string, error) {
	for _, ssoDescriptor := range idpMetadata.IDPSSODescriptors {
		for _, ssos := range ssoDescriptor.SingleSignOnServices {
			if ssos.Binding == saml.HTTPRedirectBinding {
				return ssos.Location, nil
			}
		}
	}
	return "", errors.New("IDP does not support redirect binding")
}

// CreateAuthorizationRequest creates a new SAML AuthnRequest and creates a new session in sessionStore.
// It will generate and return the session identifier.
// (the IdP will send it again to Fleet in the callback, and that's how Fleet will authenticate the session).
// If sessionTTLSeconds is 0 then a default of 5 minutes of TTL is used.
func CreateAuthorizationRequest(
	ctx context.Context,
	samlProvider *saml.ServiceProvider,
	sessionStore SessionStore,
	originalURL string,
	sessionTTLSeconds uint,
	requestData SSORequestData,
) (sessionID string, idpURL string, err error) {
	idpURL, err = getDestinationURL(samlProvider.IDPMetadata)
	if err != nil {
		return "", "", fmt.Errorf("get idp url: %w", err)
	}
	samlAuthRequest, err := samlProvider.MakeAuthenticationRequest(
		idpURL,
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "make auth request")
	}
	// We can modify the samlAuthRequest because it's not signed
	// (not a requirement when using "HTTPRedirectBinding" binding for the request)
	samlAuthRequest.ProviderName = "Fleet"

	var metadataWriter bytes.Buffer
	err = xml.NewEncoder(&metadataWriter).Encode(samlProvider.IDPMetadata)
	if err != nil {
		return "", "", fmt.Errorf("encoding metadata creating auth request: %w", err)
	}

	sessionID, err = generateSessionID()
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "generate session ID")
	}

	sessionLifetimeSeconds := cacheLifetimeSeconds
	if sessionTTLSeconds > 0 {
		sessionLifetimeSeconds = sessionTTLSeconds
	}

	// Store the session with the generated ID.
	// We cache the metadata so we can check the signatures on the response we get from the IdP.
	err = sessionStore.create(
		sessionID,
		samlAuthRequest.ID,
		originalURL,
		metadataWriter.String(),
		sessionLifetimeSeconds,
		requestData,
	)
	if err != nil {
		return "", "", fmt.Errorf("caching SSO session while creating auth request: %w", err)
	}

	// Pass the session ID as RelayState so the callback can retrieve it even
	// when the SSO cookie is not available (e.g. when a custom Apple MDM URL
	// causes the IdP to send the callback to a different domain than the one
	// that set the cookie). The session ID is an opaque, random, single-use
	// token that references server-side state — it does not grant access by
	// itself.
	idpRedirectURL, err := samlAuthRequest.Redirect(sessionID, samlProvider)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "generating redirect")
	}
	return sessionID, idpRedirectURL.String(), nil
}

func generateSessionID() (string, error) {
	// Use 24 random bytes hex-encoded (48 chars). Hex encoding produces only
	// [0-9a-f] which is safe for URLs, cookies, and form-encoded POST bodies
	// without any escaping. This avoids issues with base64's +, / and =
	// characters being corrupted during SAML RelayState round-trips through
	// IdPs (where the value travels as a URL query parameter and is echoed
	// back in an application/x-www-form-urlencoded POST).
	const sessionIDLength = 24
	key := make([]byte, sessionIDLength)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("create random session ID: %w", err)
	}
	return hex.EncodeToString(key), nil
}
