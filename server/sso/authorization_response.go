package sso

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// These are response status codes described in the core SAML spec section
	// 3.2.2.1 See http://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf
	Success int = iota
	Requestor
	Responder
	VersionMismatch
	AuthnFailed
	InvalidAttrNameOrValue
	InvalidNameIDPolicy
	NoAuthnContext
	NoAvailableIDP
	NoPassive
	NoSupportedIDP
	PartialLogout
	ProxyCountExceeded
	RequestDenied
	RequestUnsupported
	RequestVersionDeprecated
	RequestVersionTooHigh
	RequestVersionTooLow
	ResourceNotRecognized
	TooManyResponses
	UnknownAttrProfile
	UnknownPrincipal
	UnsupportedBinding
)

var statusMap = map[string]int{
	"urn:oasis:names:tc:SAML:2.0:status:Success":                  Success,
	"urn:oasis:names:tc:SAML:2.0:status:Requester":                Requestor,
	"urn:oasis:names:tc:SAML:2.0:status:Responder":                Responder,
	"urn:oasis:names:tc:SAML:2.0:status:VersionMismatch":          VersionMismatch,
	"urn:oasis:names:tc:SAML:2.0:status:AuthnFailed":              AuthnFailed,
	"urn:oasis:names:tc:SAML:2.0:status:InvalidAttrNameOrValue":   InvalidAttrNameOrValue,
	"urn:oasis:names:tc:SAML:2.0:status:InvalidNameIDPolicy":      InvalidNameIDPolicy,
	"urn:oasis:names:tc:SAML:2.0:status:NoAuthnContext":           NoAuthnContext,
	"urn:oasis:names:tc:SAML:2.0:status:NoAvailableIDP":           NoAvailableIDP,
	"urn:oasis:names:tc:SAML:2.0:status:NoPassive":                NoPassive,
	"urn:oasis:names:tc:SAML:2.0:status:NoSupportedIDP":           NoSupportedIDP,
	"urn:oasis:names:tc:SAML:2.0:status:PartialLogout":            PartialLogout,
	"urn:oasis:names:tc:SAML:2.0:status:ProxyCountExceeded":       ProxyCountExceeded,
	"urn:oasis:names:tc:SAML:2.0:status:RequestDenied":            RequestDenied,
	"urn:oasis:names:tc:SAML:2.0:status:RequestUnsupported":       RequestUnsupported,
	"urn:oasis:names:tc:SAML:2.0:status:RequestVersionDeprecated": RequestVersionDeprecated,
	"urn:oasis:names:tc:SAML:2.0:status:RequestVersionTooLow":     RequestVersionTooLow,
	"urn:oasis:names:tc:SAML:2.0:status:ResourceNotRecognized":    ResourceNotRecognized,
	"urn:oasis:names:tc:SAML:2.0:status:TooManyResponses":         TooManyResponses,
	"urn:oasis:names:tc:SAML:2.0:status:UnknownAttrProfile":       UnknownAttrProfile,
	"urn:oasis:names:tc:SAML:2.0:status:UnknownPrincipal":         UnknownPrincipal,
	"urn:oasis:names:tc:SAML:2.0:status:UnsupportedBinding":       UnsupportedBinding,
}

type resp struct {
	response *Response
	rawResp  string
}

func (r *resp) setResponse(val *Response) {
	r.response = val
}

func (r resp) statusDescription() string {
	if r.response != nil {
		return r.response.Status.StatusCode.Value
	}
	return "missing response"
}

func (r resp) UserID() string {
	if r.response != nil {
		return r.response.Assertion.Subject.NameID.Value
	}
	return ""
}

func (r resp) status() (int, error) {
	if r.response != nil {
		statusURI := r.response.Status.StatusCode.Value
		if code, ok := statusMap[statusURI]; ok {
			return code, nil
		}
	}
	return AuthnFailed, errors.New("malformed or missing auth response")
}

func (r resp) RequestID() string {
	if r.response != nil {
		return r.response.InResponseTo
	}
	return ""
}

func (r resp) rawResponse() string {
	return r.rawResp
}

// DecodeAuthResponse extracts SAML assertions from IDP response
func DecodeAuthResponse(samlResponse string) (fleet.Auth, error) {
	var authInfo resp
	authInfo.rawResp = samlResponse

	decoded, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("decoding saml response: %w", err)
	}
	var saml Response
	err = xml.NewDecoder(bytes.NewBuffer(decoded)).Decode(&saml)
	if err != nil {
		return nil, fmt.Errorf("decoding response xml: %w", err)
	}
	authInfo.response = &saml
	return &authInfo, nil
}
