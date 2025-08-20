package httpsig

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	sfv "github.com/dunglas/httpsfv"
)

type Algorithm string
type Digest string

// Metadata are the named signature metadata parameters
type Metadata string

type CreatedScheme int
type ExpiresScheme int
type NonceScheme int

const (
	// Supported signing algorithms
	Algo_RSA_PSS_SHA512    Algorithm = "rsa-pss-sha512"
	Algo_RSA_v1_5_sha256   Algorithm = "rsa-v1_5-sha256"
	Algo_HMAC_SHA256       Algorithm = "hmac-sha256"
	Algo_ECDSA_P256_SHA256 Algorithm = "ecdsa-p256-sha256"
	Algo_ECDSA_P384_SHA384 Algorithm = "ecdsa-p384-sha384"
	Algo_ED25519           Algorithm = "ed25519"

	DigestSHA256 Digest = "sha-256"
	DigestSHA512 Digest = "sha-512"

	// Signature metadata parameters
	MetaCreated   Metadata = "created"
	MetaExpires   Metadata = "expires"
	MetaNonce     Metadata = "nonce"
	MetaAlgorithm Metadata = "alg"
	MetaKeyID     Metadata = "keyid"
	MetaTag       Metadata = "tag"

	// DefaultSignatureLabel is the label that will be used for a signature if not label is provided in the parameters.
	// A request can contain multiple signatures therefore each signature is labeled.
	DefaultSignatureLabel = "sig1"

	// Nonce schemes
	NonceRandom32 = iota // 32 bit random nonce. Base64 encoded
)

// SigningProfile is the set of fields, metadata, and the label to include in a signature.
type SigningProfile struct {
	Algorithm       Algorithm
	Digest          Digest        // The http digest algorithm to apply. Defaults to sha-256.
	Fields          []SignedField // Fields and Derived components to sign.
	Metadata        []Metadata    // Metadata parameters to add to the signature.
	Label           string        // The signature label. Defaults to DefaultSignatureLabel.
	ExpiresDuration time.Duration // Current time plus this duration. Default duration 5 minutes. Used only if included in Metadata.
	Nonce           NonceScheme   // Scheme to use for generating the nonce if included in Metadata.
}

// SignedField indicates which part of the request or response to use for signing.
// This is the 'message component' in the specification.
type SignedField struct {
	Name       string
	Parameters map[string]any // Parameters are modifiers applied to the field that changes the way the signature is calculated.
}

// Fields turns a list of fields into the full specification. Used when the signed fields/components do not need to specify any parameters
func Fields(fields ...string) []SignedField {
	all := []SignedField{}
	for _, field := range fields {
		all = append(all, SignedField{
			Name:       strings.ToLower(field),
			Parameters: map[string]any{},
		})
	}
	return all
}

type SigningKey struct {
	Key    crypto.PrivateKey // private key for asymmetric algorithms
	Secret []byte            // Secret to use for symmetric algorithms
	// Meta fields
	MetaKeyID string // 'keyid' - Only used if 'keyid' is set in the SigningProfile. A value must be provided if the parameter is required in the SigningProfile. Metadata.
	MetaTag   string // 'tag'. Only used if 'tag' is set in the SigningProfile. A value must be provided if the parameter is required in the SigningProfile.
}

type Signer struct {
	profile SigningProfile
	skey    SigningKey
}

func NewSigner(profile SigningProfile, skey SigningKey) (*Signer, error) {
	err := profile.validate(skey)
	if err != nil {
		return nil, err
	}

	opts := profile.withDefaults()
	s := &Signer{
		profile: opts,
		skey:    skey,
	}
	return s, nil
}

func Sign(req *http.Request, params SigningProfile, skey SigningKey) error {
	s, err := NewSigner(params, skey)
	if err != nil {
		return err
	}
	return s.Sign(req)
}

// Sign signs the request and adds the signature headers to the request.
// If the signature fields includes Content-Digest and Content-Digest is not already included in the request then Sign will read the request body to calculate the digest and set the header.  The request body will be replaced with a new io.ReaderCloser.
func (s *Signer) Sign(req *http.Request) error {
	// Add the content-digest if covered by the signature and not already present
	if signedFields(s.profile.Fields).includes("content-digest") && req.Header.Get("Content-Digest") == "" {
		di, err := digestBody(s.profile.Digest, req.Body)
		if err != nil {
			return err
		}
		req.Body = di.NewBody
		digestValue, err := createDigestHeader(s.profile.Digest, di.Digest)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Digest", digestValue)
	}

	baseParams, err := s.baseParameters()
	if err != nil {
		return err
	}

	return sign(
		httpMessage{
			Req: req,
		}, sigParameters{
			Base:       baseParams,
			Algo:       s.profile.Algorithm,
			PrivateKey: s.skey.Key,
			Secret:     s.skey.Secret,
			Label:      s.profile.Label,
		})
}

func (s *Signer) SignResponse(resp *http.Response) error {
	baseParams, err := s.baseParameters()
	if err != nil {
		return err
	}

	return sign(
		httpMessage{
			IsResponse: true,
			Resp:       resp,
		}, sigParameters{
			Base:       baseParams,
			Algo:       s.profile.Algorithm,
			PrivateKey: s.skey.Key,
			Secret:     s.skey.Secret,
			Label:      s.profile.Label,
		})
}

func (s *Signer) baseParameters() (sigBaseInput, error) {
	bp := sigBaseInput{
		Components:     componentsIDs(s.profile.Fields),
		MetadataParams: s.profile.Metadata,
		MetadataValues: s,
	}

	return bp, nil
}

func (s *Signer) Created() (int, error) {
	return int(time.Now().Unix()), nil
}

func (s *Signer) Expires() (int, error) {
	return int(time.Now().Add(s.profile.ExpiresDuration).Unix()), nil
}

func (s *Signer) Nonce() (string, error) {
	switch s.profile.Nonce {
	case NonceRandom32:
		return nonceRandom32()
	}
	return "", fmt.Errorf("Invalid nonce scheme '%d'", s.profile.Nonce)
}

func (s *Signer) Alg() (string, error) {
	return string(s.profile.Algorithm), nil
}

func (s *Signer) KeyID() (string, error) {
	return s.skey.MetaKeyID, nil
}

func (s *Signer) Tag() (string, error) {
	return s.skey.MetaTag, nil
}

func (so SigningProfile) validate(skey SigningKey) error {
	if so.Algorithm == "" {
		return fmt.Errorf("Missing required signing option 'Algorithm'")
	}
	if so.Algorithm.symmetric() && len(skey.Secret) == 0 {
		return newError(ErrInvalidSignatureOptions, "Missing required 'Secret' value in SigningKey")
	}
	if !so.Algorithm.symmetric() && skey.Key == nil {
		return newError(ErrInvalidSignatureOptions, "Missing required 'Key' value in SigningKey")
	}
	if !isSafeString(so.Label) {
		return fmt.Errorf("Invalid label name '%s'", so.Label)
	}
	for _, sf := range so.Fields {
		if !isSafeString(sf.Name) {
			return fmt.Errorf("Invalid signing field name '%s'", sf.Name)
		}
	}

	for _, md := range so.Metadata {
		switch md {
		case MetaKeyID:
			if skey.MetaKeyID == "" {
				return fmt.Errorf("'keyid' metadata parameter was listed but missing MetaKeyID value'")
			}
			if !isSafeString(skey.MetaKeyID) {
				return fmt.Errorf("'keyid' metadata parameter can only contain printable characters'")
			}
		case MetaTag:
			if skey.MetaTag == "" {
				return fmt.Errorf("'tag' metadata parameter was listed but missing MetaTag value'")
			}
			if !isSafeString(skey.MetaTag) {
				return fmt.Errorf("'tag' metadata parameter can only contain printable characters'")
			}
		}
	}
	return nil
}

func (sp SigningProfile) withDefaults() SigningProfile {
	final := SigningProfile{
		Algorithm:       sp.Algorithm,
		Digest:          sp.Digest,
		Fields:          sp.Fields,
		Metadata:        sp.Metadata,
		Label:           sp.Label,
		ExpiresDuration: sp.ExpiresDuration,
		Nonce:           NonceRandom32,
	}
	// Defaults
	if final.Label == "" {
		final.Label = DefaultSignatureLabel
	}
	if final.ExpiresDuration == 0 {
		final.ExpiresDuration = time.Minute * 5
	}
	if final.Digest == "" {
		final.Digest = DigestSHA256
	}

	return final
}

func (sf SignedField) componentID() componentID {
	item := sfv.NewItem(sf.Name)
	for key, param := range sf.Parameters {
		item.Params.Add(key, param)
	}
	return componentID{
		Name: strings.ToLower(sf.Name),
		Item: item,
	}
}

type signedFields []SignedField

func (sf signedFields) includes(field string) bool {
	target := strings.ToLower(field)
	for _, fld := range sf {
		if fld.Name == target {
			return true
		}
	}
	return false
}

func (a Algorithm) symmetric() bool {
	switch a {
	case Algo_HMAC_SHA256:
		return true
	}
	return false
}
func componentsIDs(sfs []SignedField) []componentID {
	cIDs := []componentID{}
	for _, sf := range sfs {
		cIDs = append(cIDs, sf.componentID())
	}

	return cIDs
}

func nonceRandom32() (string, error) {
	nonce := make([]byte, 32)
	n, err := rand.Read(nonce)
	if err != nil || n < 32 {
		return "", fmt.Errorf("could not generate nonce")
	}
	return base64.StdEncoding.EncodeToString(nonce), nil
}

func isSafeString(s string) bool {
	for _, c := range s {
		if !unicode.IsPrint(c) {
			return false
		}
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}
