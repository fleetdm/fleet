package sso

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GetMetadata returns the parsed IdP metadata from the config.MetadataURL or config.Metadata.
// If config.MetadataURL is set, then it's used to request the metadata.
// If config.MetadataURL is not set, then config.Metadata is used.
func GetMetadata(config *fleet.SSOProviderSettings) (*saml.EntityDescriptor, error) {
	if config.MetadataURL == "" && config.Metadata == "" {
		return nil, fmt.Errorf("missing metadata for idp %q", config.IDPName)
	}

	var xmlMetadata []byte
	if config.MetadataURL != "" {
		var err error
		xmlMetadata, err = getMetadata(config.MetadataURL)
		if err != nil {
			return nil, err
		}
	} else {
		xmlMetadata = []byte(config.Metadata)
	}

	return ParseMetadata(xmlMetadata)
}

func getMetadata(metadataURL string) ([]byte, error) {
	client := fleethttp.NewClient(fleethttp.WithTimeout(5 * time.Second))
	request, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SAML metadata server at %s returned %s", metadataURL, resp.Status)
	}
	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return xmlData, nil
}

// ParseMetadata parses SSO IdP metadata from the given raw XML bytes.
func ParseMetadata(xmlMetadata []byte) (*saml.EntityDescriptor, error) {
	var entityDescriptor saml.EntityDescriptor
	if err := xml.Unmarshal(xmlMetadata, &entityDescriptor); err != nil {
		return nil, err
	}
	return &entityDescriptor, nil
}

// SAMLProviderFromConfiguredMetadata is used in SSO initiation to create a
// crewjam/saml.ServiceProvider from the configured SSO metadata.
func SAMLProviderFromConfiguredMetadata(
	ctx context.Context,
	entityID string,
	acsURL string,
	settings *fleet.SSOProviderSettings,
) (*saml.ServiceProvider, error) {
	entityDescriptor, err := GetMetadata(settings)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to get and parse IdP metadata",
			InternalErr: err,
		})
	}
	parsedACSURL, err := url.Parse(acsURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to parse ACS URL")
	}
	return &saml.ServiceProvider{
		EntityID:    entityID,
		AcsURL:      *parsedACSURL,
		IDPMetadata: entityDescriptor,
	}, nil
}
