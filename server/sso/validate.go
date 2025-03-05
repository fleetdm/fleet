package sso

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/fleetdm/fleet/v4/server/fleet"
	rtvalidator "github.com/mattermost/xml-roundtrip-validator"
	dsig "github.com/russellhaering/goxmldsig"
)

type Validator interface {
	ValidateSignature(auth fleet.Auth) (fleet.Auth, error)
	ValidateResponse(auth fleet.Auth) error
}

type validator struct {
	context           *dsig.ValidationContext
	clock             *dsig.Clock
	metadata          Metadata
	expectedAudiences []string
}

func Clock(clock *dsig.Clock) func(v *validator) {
	return func(v *validator) {
		v.clock = clock
	}
}

func WithExpectedAudience(audiences ...string) func(v *validator) {
	return func(v *validator) {
		v.expectedAudiences = audiences
	}
}

// NewValidator is used to validate the response to an auth request.
// metadata is from the IDP.
func NewValidator(metadata Metadata, opts ...func(v *validator)) (Validator, error) {
	v := validator{
		metadata: metadata,
	}

	var idpCertStore dsig.MemoryX509CertificateStore
	for _, key := range v.metadata.IDPSSODescriptor.KeyDescriptors {
		if len(key.KeyInfo.X509Data.X509Certificates) == 0 {
			return nil, errors.New("missing x509 cert")
		}
		certData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key.KeyInfo.X509Data.X509Certificates[0].Data))
		if err != nil {
			return nil, fmt.Errorf("decoding idp x509 cert: %w", err)
		}
		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			return nil, fmt.Errorf("parsing idp x509 cert: %w", err)
		}
		idpCertStore.Roots = append(idpCertStore.Roots, cert)
	}
	for _, opt := range opts {
		opt(&v)
	}
	if v.clock == nil {
		v.clock = dsig.NewRealClock()
	}
	v.context = dsig.NewDefaultValidationContext(&idpCertStore)
	v.context.Clock = v.clock
	return &v, nil
}

func (v *validator) ValidateResponse(auth fleet.Auth) error {
	info := auth.(*resp)
	// make sure response is current
	onOrAfter, err := time.Parse(time.RFC3339, info.response.Assertion.Conditions.NotOnOrAfter)
	if err != nil {
		return fmt.Errorf("missing timestamp from condition: %w", err)
	}
	notBefore, err := time.Parse(time.RFC3339, info.response.Assertion.Conditions.NotBefore)
	if err != nil {
		return fmt.Errorf("missing timestamp from condition: %w", err)
	}
	currentTime := v.clock.Now()
	if currentTime.After(onOrAfter) {
		return errors.New("response expired")
	}
	if currentTime.Before(notBefore) {
		return errors.New("response too early")
	}

	verifiesAudience := false
	for _, audience := range v.expectedAudiences {
		if info.response.Assertion.Conditions.AudienceRestriction.Audience == audience {
			verifiesAudience = true
			break
		}
	}
	if !verifiesAudience {
		return errors.New("wrong audience:" + info.response.Assertion.Conditions.AudienceRestriction.Audience)
	}

	if auth.UserID() == "" {
		return errors.New("missing user id")
	}
	return nil
}

func (v *validator) ValidateSignature(auth fleet.Auth) (fleet.Auth, error) {
	info := auth.(*resp)
	status, err := info.status()
	if err != nil {
		return nil, errors.New("missing or malformed response")
	}
	if status != Success {
		return nil, fmt.Errorf("response status %s", info.statusDescription())
	}

	// Examine the response for attempts to exploit weaknesses in Go's
	// encoding/xml
	decoded := info.rawResponse()
	err = rtvalidator.Validate(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("response XML failed validation: %w", err)
	}

	doc := etree.NewDocument()
	err = doc.ReadFromBytes(decoded)
	if err != nil || doc.Root() == nil {
		return nil, fmt.Errorf("parsing xml response: %w", err)
	}
	elt := doc.Root()
	signed, err := v.validateSignature(elt)
	if err != nil {
		return nil, fmt.Errorf("signing verification failed: %w", err)
	}
	// We've verified that the response hasn't been tampered with at this point
	signedDoc := etree.NewDocument()
	signedDoc.SetRoot(signed)
	buffer, err := signedDoc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("creating signed doc buffer: %w", err)
	}
	var response Response
	err = xml.Unmarshal(buffer, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling signed doc: %w", err)
	}
	info.setResponse(&response)
	return info, nil
}

func (v *validator) validateSignature(elt *etree.Element) (*etree.Element, error) {
	validated, err := v.context.Validate(elt)
	if err == nil {
		// If entire doc is signed, success, we're done.
		return validated, nil
	}
	// Some IdPs (like Google) do not sign the root, and only sign the Assertion.
	if err == dsig.ErrMissingSignature {
		if err := v.validateAssertionSignature(elt); err != nil {
			return nil, err
		}
		return elt, nil
	}
	return nil, err
}

var (
	errMissingAssertion              = errors.New("missing Assertion element under namespace urn:oasis:names:tc:SAML:2.0:assertion")
	errMultipleAssertions            = errors.New("multiple Assertions elements found")
	errAssertionWithInvalidNamespace = errors.New("Assertion with invalid namespace found")
)

// validateAssertionSignature validates that one "Assertion" child element exists under
// the "urn:oasis:names:tc:SAML:2.0:assertion" namespace and that it's signed by the IdP.
// It returns:
//   - errMissingAssertion if there is no "Assertion" child element under the given tree.
//   - errMultipleAssertions if there's more than one "Assertion" element under the given tree.
//   - errAssertionWithInvalidNamespace if an "Assertion" element has a namespace that's not
//     "urn:oasis:names:tc:SAML:2.0:assertion"
//   - an error if the signature of the one "Assertion" element is invalid.
func (v *validator) validateAssertionSignature(elt *etree.Element) error {
	var assertion *etree.Element
	for _, child := range elt.ChildElements() {
		if child.Tag == "Assertion" {
			if child.NamespaceURI() != "urn:oasis:names:tc:SAML:2.0:assertion" {
				return errAssertionWithInvalidNamespace
			}
			if assertion != nil {
				return errMultipleAssertions
			}
			assertion = child
		}
	}
	if assertion == nil {
		return errMissingAssertion
	}
	if _, err := v.context.Validate(assertion); err != nil {
		return fmt.Errorf("failed to validate assertion signature: %w", err)
	}
	return nil
}

const (
	idPrefix   = "id"
	idSize     = 16
	idAlphabet = `1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
)

// There isn't anything in the SAML spec that tells us what is valid inside an
// ID other than expecting that it has to be unique and valid XML. ADFS blows
// up on '=' in the ID, so we are using an alphabet that we know works.
//
// Azure IdP requires that the ID begin with a character so we use the constant
// prefix.
func generateSAMLValidID() (string, error) {
	randomBytes := make([]byte, idSize)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	for i := 0; i < idSize; i++ {
		randomBytes[i] = idAlphabet[randomBytes[i]%byte(len(idAlphabet))]
	}
	return idPrefix + string(randomBytes), nil
}

func ValidateAudiences(metadata Metadata, auth fleet.Auth, audiences ...string) error {
	validator, err := NewValidator(metadata, WithExpectedAudience(audiences...))
	if err != nil {
		return fmt.Errorf("create validator from metadata: %w", err)
	}
	// make sure the response hasn't been tampered with
	auth, err = validator.ValidateSignature(auth)
	if err != nil {
		return fmt.Errorf("signature validation failed: %w", err)
	}
	// make sure the response isn't stale
	err = validator.ValidateResponse(auth)
	if err != nil {
		return fmt.Errorf("response validation failed: %w", err)
	}

	return nil
}
