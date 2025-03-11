package sso

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const (
	cacheLifetime = 300 // five minutes
)

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

// See SAML Bindings http://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf
// Section 3.4.4.1
func deflate(xmlBuffer *bytes.Buffer) (string, error) {
	// Gzip
	var deflated bytes.Buffer
	writer, err := flate.NewWriter(&deflated, flate.DefaultCompression)
	if err != nil {
		return "", fmt.Errorf("create flate writer: %w", err)
	}

	count := xmlBuffer.Len()
	n, err := io.Copy(writer, xmlBuffer)
	if err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("compressing auth request: %w", err)
	}

	if int(n) != count {
		_ = writer.Close()
		return "", errors.New("incomplete write during compression")
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close flate writer: %w", err)
	}

	// Base64
	encbuff := deflated.Bytes()
	encoded := base64.StdEncoding.EncodeToString(encbuff)
	return encoded, nil
}

func CreateAuthorizationRequest2(ctx context.Context,
	sessionStore SessionStore,
	originalURL string,
	samlProvider *saml.ServiceProvider,
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
	// cache metadata so we can check the signatures on the response we get from the IDP
	err = sessionStore.create(samlAuthRequest.ID,
		originalURL,
		metadataWriter.String(),
		cacheLifetime,
	)
	if err != nil {
		return "", fmt.Errorf("caching metadata while creating auth request: %w", err)
	}

	idpRedirectURL, err := samlAuthRequest.Redirect("", samlProvider)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generating redirect")
	}
	return idpRedirectURL.String(), nil
}
