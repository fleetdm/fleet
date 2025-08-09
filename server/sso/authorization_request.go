package sso

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server"
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
	relayState string,
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
	)
	if err != nil {
		return "", "", fmt.Errorf("caching SSO session while creating auth request: %w", err)
	}

	//relayState := "" // Fleet currently doesn't use/set RelayState
	// NOTE(mna): the 3rd-party SAML package does not properly encode the relay
	// state query string, we must ensure it is encoded before passing it on.
	idpRedirectURL, err := samlAuthRequest.Redirect(url.QueryEscape(relayState), samlProvider)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "generating redirect")
	}
	return sessionID, idpRedirectURL.String(), nil
}

func generateSessionID() (string, error) {
	const sessionIDLength = 24
	sessionID, err := server.GenerateRandomText(sessionIDLength)
	if err != nil {
		return "", fmt.Errorf("create random session ID: %w", err)
	}
	return sessionID, nil
}
