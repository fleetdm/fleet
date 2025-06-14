package sso

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"

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
// It will generate a RelayState token that will be used as session identifier
// (the IdP will send it again to Fleet in the callback, and that's how Fleet will authenticate the session).
// If sessionTTLSeconds is 0 then a default of 5 minutes of TTL is used.
func CreateAuthorizationRequest(
	ctx context.Context,
	samlProvider *saml.ServiceProvider,
	sessionStore SessionStore,
	originalURL string,
	sessionTTLSeconds uint,
) (string, error) {
	idpURL, err := getDestinationURL(samlProvider.IDPMetadata)
	if err != nil {
		return "", fmt.Errorf("get idp url: %w", err)
	}
	samlAuthRequest, err := samlProvider.MakeAuthenticationRequest(
		idpURL,
		"HTTPRedirectBinding",
		"HTTPPostBinding",
	)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "make auth request")
	}
	// We can modify the samlAuthRequest because it's not signed
	// (not a requirement when using "HTTPRedirectBinding" binding for the request)
	samlAuthRequest.ProviderName = "Fleet"

	var metadataWriter bytes.Buffer
	err = xml.NewEncoder(&metadataWriter).Encode(samlProvider.IDPMetadata)
	if err != nil {
		return "", fmt.Errorf("encoding metadata creating auth request: %w", err)
	}

	relayStateToken, err := generateFleetRelayStateToken()
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generate RelayState token")
	}

	sessionLifetimeSeconds := cacheLifetimeSeconds
	if sessionTTLSeconds > 0 {
		sessionLifetimeSeconds = sessionTTLSeconds
	}

	// Store the session with RelayState as session identifier.
	// We cache the metadata so we can check the signatures on the response we get from the IdP.
	err = sessionStore.create(
		relayStateToken,
		samlAuthRequest.ID,
		originalURL,
		metadataWriter.String(),
		sessionLifetimeSeconds,
	)
	if err != nil {
		return "", fmt.Errorf("caching SSO session while creating auth request: %w", err)
	}

	// Escape RelayState (crewjam/saml is not escaping it)
	relayStateToken = url.QueryEscape(relayStateToken)

	idpRedirectURL, err := samlAuthRequest.Redirect(relayStateToken, samlProvider)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generating redirect")
	}
	return idpRedirectURL.String(), nil
}

const relayStateTokenPrefix = "fleet_"

func generateFleetRelayStateToken() (string, error) {
	// Create RelayState token to identify the session.
	const (
		relayStateTokenLength = 24
	)
	token, err := server.GenerateRandomText(relayStateTokenLength)
	if err != nil {
		return "", fmt.Errorf("create random RelayState token: %w", err)
	}
	return relayStateTokenPrefix + token, nil
}

func checkFleetRelayStateToken(relayState string) bool {
	return strings.HasPrefix(relayState, relayStateTokenPrefix)
}
