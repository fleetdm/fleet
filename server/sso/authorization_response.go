package sso

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// maxSAMLResponseDepth bounds how deeply nested the SAMLResponse XML may
	// be. Legitimate SAML responses are shallow; deep nesting is the signature
	// of a canonicalization bomb, which runs before any certificate or signature
	// check on the unauthenticated SSO callback endpoints.
	maxSAMLResponseDepth = 100
	// maxSAMLResponseElements bounds the total number of XML elements in the
	// SAMLResponse, for the same reason (a bomb can be wide rather than deep).
	maxSAMLResponseElements = 5000
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

// validateSAMLResponseShape parses the decoded SAMLResponse XML and rejects
// documents that are excessively deep or have too many elements before they
// reach goxmldsig's pre-signature canonicalization, which (as of the time of writing)
// has no traversal limit of its own.
func validateSAMLResponseShape(samlResponse []byte) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(samlResponse); err != nil {
		return fmt.Errorf("parsing SAMLResponse XML: %w", err)
	}
	root := doc.Root()
	if root == nil {
		return errors.New("SAMLResponse has no root element")
	}

	count := 0
	var walk func(el *etree.Element, depth int) error
	walk = func(el *etree.Element, depth int) error {
		if depth > maxSAMLResponseDepth {
			return fmt.Errorf("SAMLResponse exceeds maximum nesting depth of %d", maxSAMLResponseDepth)
		}
		count++
		if count > maxSAMLResponseElements {
			return fmt.Errorf("SAMLResponse exceeds maximum element count of %d", maxSAMLResponseElements)
		}
		for _, child := range el.ChildElements() {
			if err := walk(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, 1)
}

// ParseAndVerifySAMLResponse runs the parsing and validation of SAMLResponses.
func ParseAndVerifySAMLResponse(samlProvider *saml.ServiceProvider, samlResponse []byte, requestID string, acsURL *url.URL) (fleet.Auth, error) {
	// Reject oversized/over-nested documents before handing them to
	// crewjam/saml -> goxmldsig, whose pre-signature canonicalization is (at the time of writing)
	// unbounded and runs without authentication.
	if err := validateSAMLResponseShape(samlResponse); err != nil {
		return nil, err
	}

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
