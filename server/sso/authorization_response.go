package sso

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Since there's not a standard for display names, I have collected the most
// commonly used attribute names for it.
//
// Most of the items here come from:
//
//   - https://docs.ldap.com/specs/rfc2798.txt
//   - https://docs.microsoft.com/en-us/windows-server/identity/ad-fs/technical-reference/the-role-of-claims
var validDisplayNameAttrs = map[string]struct{}{
	"name":            {},
	"displayname":     {},
	"cn":              {},
	"urn:oid:2.5.4.3": {},
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": {},
}

type resp struct {
	response *saml.Response
	rawResp  []byte
}

var _ fleet.Auth = resp{}

// UserID partially implements the fleet.Auth interface.
func (r resp) UserID() string {
	if r.response != nil {
		return r.response.Assertion.Subject.NameID.Value
	}
	return ""
}

// UserDisplayName partially implements the fleet.Auth interface.
func (r resp) UserDisplayName() string {
	if r.response != nil {
		for _, attrStatement := range r.response.Assertion.AttributeStatements {
			for _, attr := range attrStatement.Attributes {
				if _, ok := validDisplayNameAttrs[strings.ToLower(attr.Name)]; ok {
					for _, vv := range attr.Values {
						if vv.Value != "" {
							return vv.Value
						}
					}
				}
			}
		}
	}
	return ""
}

func (r resp) status() (string, error) {
	if r.response != nil {
		return r.response.Status.StatusCode.Value, nil
	}
	return saml.StatusAuthnFailed, errors.New("malformed or missing auth response")
}

// RequestID partially implements the fleet.Auth interface.
func (r resp) RequestID() string {
	if r.response != nil {
		return r.response.InResponseTo
	}
	return ""
}

// AssertionAttributes partially implements the fleet.Auth interface.
func (r resp) AssertionAttributes() []fleet.SAMLAttribute {
	if r.response == nil {
		return nil
	}
	var attrs []fleet.SAMLAttribute
	for _, attrStatement := range r.response.Assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			var values []fleet.SAMLAttributeValue
			for _, value := range attr.Values {
				values = append(values, fleet.SAMLAttributeValue{
					Type:  value.Type,
					Value: value.Value,
				})
			}
			attrs = append(attrs, fleet.SAMLAttribute{
				Name:   attr.Name,
				Values: values,
			})
		}
	}
	return attrs
}

func (r resp) RawResponse() []byte {
	return r.rawResp
}

// DecodeAuthResponse extracts SAML assertions from the IDP response (base64 encoded).
func DecodeAuthResponse(samlResponse string) (fleet.Auth, error) {
	decoded, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("decoding saml response: %w", err)
	}

	return decodeSAMLResponse(decoded)
}

func decodeSAMLResponse(rawXML []byte) (fleet.Auth, error) {
	var samlResponse saml.Response
	if err := xml.NewDecoder(bytes.NewBuffer(rawXML)).Decode(&samlResponse); err != nil {
		return nil, fmt.Errorf("decoding response xml: %w", err)
	}
	return &resp{
		response: &samlResponse,
		rawResp:  rawXML,
	}, nil
}
