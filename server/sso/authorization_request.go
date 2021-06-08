package sso

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"net/url"
	"time"

	"github.com/pkg/errors"
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
// redirect binding.
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
		return "", errors.Wrap(err, "creating auth request id")
	}
	destinationURL, err := getDestinationURL(settings)
	if err != nil {
		return "", errors.Wrap(err, "creating auth request")
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
		return "", errors.Wrap(err, "encoding metadata creating auth request")
	}
	// cache metadata so we can check the signatures on the response we get from the IDP
	err = settings.SessionStore.create(requestID,
		settings.OriginalURL,
		reader.String(),
		cacheLifetime,
	)
	if err != nil {
		return "", errors.Wrap(err, "caching cert while creating auth request")
	}
	u, err := url.Parse(destinationURL)
	if err != nil {
		return "", errors.Wrap(err, "parsing destination url")
	}
	qry := u.Query()

	var writer bytes.Buffer
	err = xml.NewEncoder(&writer).Encode(request)
	if err != nil {
		return "", errors.Wrap(err, "encoding auth request xml")
	}
	authQueryVal, err := deflate(&writer)
	if err != nil {
		return "", errors.Wrap(err, "unable to compress auth info")
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
		return "", errors.Wrap(err, "create flate writer")
	}
	n, err := writer.Write(xmlBuffer.Bytes())
	if n != xmlBuffer.Len() {
		_ = writer.Close()
		return "", errors.New("incomplete write during compression")
	}
	if err != nil {
		_ = writer.Close()
		return "", errors.Wrap(err, "compressing auth request")
	}
	if err := writer.Close(); err != nil {
		return "", errors.Wrap(err, "close flate writer")
	}

	// Base64
	encbuff := deflated.Bytes()
	encoded := base64.StdEncoding.EncodeToString(encbuff)
	return encoded, nil
}
