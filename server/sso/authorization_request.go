package sso

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"
)

const (
	samlVersion   = "2.0"
	cacheLifetime = 300 // five minutes
)

// RelayState sets optional relay state
func RelayState(v string) func(*opts) {
	return func(o *opts) {
		o.relayState = v
	}
}

type opts struct {
	relayState string
}

// CreateAuthorizationRequest creates a url suitable for use to satisfy the SAML
// redirect binding. It returns the URL of the identity provider, configured to
// initiate the authentication request.
// See http://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf Section 3.4
func CreateAuthorizationRequest(settings *Settings, issuer string, options ...func(o *opts)) (string, error) {
	var optionalParams opts
	for _, opt := range options {
		opt(&optionalParams)
	}
	if settings.Metadata == nil {
		return "", errors.New("missing settings metadata")
	}
	requestID, err := generateSAMLValidID()
	if err != nil {
		return "", fmt.Errorf("creating auth request id: %w", err)
	}
	destinationURL, err := getDestinationURL(settings)
	if err != nil {
		return "", fmt.Errorf("creating auth request: %w", err)
	}
	request := AuthnRequest{
		XMLName: xml.Name{
			Local: "samlp:AuthnRequest",
		},
		ID:                          requestID,
		SAMLP:                       "urn:oasis:names:tc:SAML:2.0:protocol",
		SAML:                        "urn:oasis:names:tc:SAML:2.0:assertion",
		AssertionConsumerServiceURL: settings.AssertionConsumerServiceURL,
		Destination:                 destinationURL,
		IssueInstant:                time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Version:                     samlVersion,
		ProviderName:                "Fleet",
		Issuer: Issuer{
			XMLName: xml.Name{
				Local: "saml:Issuer",
			},
			Url: issuer,
		},
	}

	var reader bytes.Buffer
	err = xml.NewEncoder(&reader).Encode(settings.Metadata)
	if err != nil {
		return "", fmt.Errorf("encoding metadata creating auth request: %w", err)
	}

	// cache metadata so we can check the signatures on the response we get from the IDP
	err = settings.SessionStore.create(requestID,
		settings.OriginalURL,
		reader.String(),
		cacheLifetime,
	)
	if err != nil {
		return "", fmt.Errorf("caching metadata while creating auth request: %w", err)
	}

	u, err := url.Parse(destinationURL)
	if err != nil {
		return "", fmt.Errorf("parsing destination url: %w", err)
	}
	qry := u.Query()

	var writer bytes.Buffer
	err = xml.NewEncoder(&writer).Encode(request)
	if err != nil {
		return "", fmt.Errorf("encoding auth request xml: %w", err)
	}

	authQueryVal, err := deflate(&writer)
	if err != nil {
		return "", fmt.Errorf("unable to compress auth info: %w", err)
	}

	qry.Set("SAMLRequest", authQueryVal)
	if optionalParams.relayState != "" {
		qry.Set("RelayState", optionalParams.relayState)
	}

	u.RawQuery = qry.Encode()
	return u.String(), nil
}

func getDestinationURL(settings *Settings) (string, error) {
	for _, sso := range settings.Metadata.IDPSSODescriptor.SingleSignOnService {
		if sso.Binding == RedirectBinding {
			return sso.Location, nil
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
