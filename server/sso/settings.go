package sso

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const RedirectBinding = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"

type Settings struct {
	Metadata *saml.EntityDescriptor
	// AssertionConsumerServiceURL is the call back on the service provider which responds
	// to the IDP
	AssertionConsumerServiceURL string
	SessionStore                SessionStore
	OriginalURL                 string
}

// ParseMetadata writes metadata xml to a struct
func ParseMetadata(metadata string) (*saml.EntityDescriptor, error) {
	return samlsp.ParseMetadata([]byte(metadata))
}

// GetMetadata retrieves information describing how to interact with a particular
// IDP via a remote URL. metadataURL is the location where the metadata is located
// and timeout defines how long to wait to get a response form the metadata
// server.
func GetMetadata(ctx context.Context, metadataURL string) (*saml.EntityDescriptor, error) {
	parsedURL, err := url.Parse(metadataURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse metadata url")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	entity, err := samlsp.FetchMetadata(ctx, client, *parsedURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetch IDP metadata")
	}
	return entity, nil
}
