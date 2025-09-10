package sso

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"slices"
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
	assertion *saml.Assertion
}

var _ fleet.Auth = resp{}

// UserID partially implements the fleet.Auth interface.
func (r resp) UserID() string {
	if r.assertion != nil && r.assertion.Subject != nil && r.assertion.Subject.NameID != nil {
		return r.assertion.Subject.NameID.Value
	}
	return ""
}

// UserDisplayName partially implements the fleet.Auth interface.
func (r resp) UserDisplayName() string {
	if r.assertion == nil {
		return ""
	}
	for _, attrStatement := range r.assertion.AttributeStatements {
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
	return ""
}

// AssertionAttributes partially implements the fleet.Auth interface.
func (r resp) AssertionAttributes() []fleet.SAMLAttribute {
	if r.assertion == nil {
		return nil
	}
	var attrs []fleet.SAMLAttribute
	for _, attrStatement := range r.assertion.AttributeStatements {
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

// DecodeSAMLResponse base64-decodes the SAMLResponse.
func DecodeSAMLResponse(samlResponse string) ([]byte, error) {
	decodedSAMLResponse, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("decoding SAMLResponse: %w", err)
	}
	return decodedSAMLResponse, nil
}

// validateAudiences validates that the audience restrictions of the assertion is one of the expected audiences.
func validateAudiences(assertion *saml.Assertion, expectedAudiences []string) error {
	if assertion.Conditions == nil {
		return errors.New("missing conditions in assertion")
	}
	if len(assertion.Conditions.AudienceRestrictions) == 0 {
		return errors.New("missing audience restrictions")
	}
	for _, audienceRestriction := range assertion.Conditions.AudienceRestrictions {
		if slices.Contains(expectedAudiences, audienceRestriction.Audience.Value) {
			return nil
		}
	}
	return fmt.Errorf("wrong audience: %+v", assertion.Conditions.AudienceRestrictions)
}

// ParseAndVerifySAMLResponse runs the parsing and validation of SAMLResponses.
func ParseAndVerifySAMLResponse(samlProvider *saml.ServiceProvider, samlResponse []byte, requestID string, acsURL *url.URL) (fleet.Auth, error) {
	verifiedAssertion, err := samlProvider.ParseXMLResponse(samlResponse, []string{requestID}, *acsURL)
	if err != nil {
		if samlErr, ok := err.(*saml.InvalidResponseError); ok {
			err = samlErr.PrivateErr
		}
		return nil, err
	}
	return &resp{
		assertion: verifiedAssertion,
	}, nil
}
