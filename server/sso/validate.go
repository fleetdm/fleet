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
	"github.com/russellhaering/goxmldsig/etreeutils"
)

type Validator interface {
	ValidateSignature(auth fleet.Auth) (fleet.Auth, error)
	ValidateResponse(auth fleet.Auth) error
}

type validator struct {
	context  *dsig.ValidationContext
	clock    *dsig.Clock
	metadata Metadata
}

func Clock(clock *dsig.Clock) func(v *validator) {
	return func(v *validator) {
		v.clock = clock
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
	decoded, err := base64.StdEncoding.DecodeString(info.rawResponse())
	if err != nil {
		return nil, fmt.Errorf("base64 decode response: %w", err)
	}

	// Examine the response for attempts to exploit weaknesses in Go's
	// encoding/xml
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
	if err == dsig.ErrMissingSignature {
		// If entire document is not signed find signed assertions, remove assertions
		// that are not signed.
		err = v.validateAssertionSignature(elt)
		if err != nil {
			return nil, err
		}
		return elt, nil
	}

	return nil, err
}

func (v *validator) validateAssertionSignature(elt *etree.Element) error {
	validateAssertion := func(ctx etreeutils.NSContext, unverified *etree.Element) error {
		if unverified.Parent() != elt {
			return fmt.Errorf("assertion with unexpected parent: %s", unverified.Parent().Tag)
		}
		// Remove assertions that are not signed.
		detached, err := etreeutils.NSDetatch(ctx, unverified)
		if err != nil {
			return err
		}
		signed, err := v.context.Validate(detached)
		if err != nil {
			return err
		}
		elt.RemoveChild(unverified)
		elt.AddChild(signed)
		return nil
	}
	return etreeutils.NSFindIterate(elt, "urn:oasis:names:tc:SAML:2.0:assertion", "Assertion", validateAssertion)
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
