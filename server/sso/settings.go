package sso

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"

	dsigtypes "github.com/russellhaering/goxmldsig/types"
)

// NOTE(mna): saml.EntityDescriptor
type Metadata struct {
	XMLName          xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID         string           `xml:"entityID,attr"`
	IDPSSODescriptor IDPSSODescriptor `xml:"IDPSSODescriptor"`
}

// NOTE(mna): saml.IDPSSODescriptor, saml.SSODescriptor, saml.RoleDescriptor
type IDPSSODescriptor struct {
	XMLName             xml.Name              `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	KeyDescriptors      []KeyDescriptor       `xml:"KeyDescriptor"`
	NameIDFormats       []NameIDFormat        `xml:"NameIDFormat"`
	SingleSignOnService []SingleSignOnService `xml:"SingleSignOnService"`
	Attributes          []Attribute           `xml:"Attribute"`
}

type KeyDescriptor struct {
	XMLName xml.Name          `xml:"urn:oasis:names:tc:SAML:2.0:metadata KeyDescriptor"`
	Use     string            `xml:"use,attr"`
	KeyInfo dsigtypes.KeyInfo `xml:"KeyInfo"`
}

type NameIDFormat struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata NameIDFormat"`
	Value   string   `xml:",chardata"`
}

type SingleSignOnService struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleSignOnService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
}

const (
	PasswordProtectedTransport = "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport"
	RedirectBinding            = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
)

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
