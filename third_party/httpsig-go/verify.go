package httpsig

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"time"

	sfv "github.com/dunglas/httpsfv"
)

var (
	DefaultVerifyProfile = VerifyProfile{
		SignatureLabel:       DefaultSignatureLabel,
		AllowedAlgorithms:    []Algorithm{Algo_ECDSA_P256_SHA256, Algo_ECDSA_P384_SHA384, Algo_ED25519, Algo_HMAC_SHA256},
		RequiredFields:       DefaultRequiredFields,
		RequiredMetadata:     []Metadata{MetaCreated, MetaKeyID},
		DisallowedMetadata:   []Metadata{MetaAlgorithm}, // The algorithm should be looked up from the keyid not an explicit setting.
		CreatedValidDuration: time.Minute * 5,           // Signatures must have been created within within the last 5 minutes
		DateFieldSkew:        time.Minute,               // If the created parameter is present, the Date header cannot be more than a minute off.
	}

	// DefaultRequiredFields covers the request body with 'content-digest' the method and full URI.
	// As per the specification Date is not covered in favor of using the 'created' metadata parameter.
	DefaultRequiredFields = Fields("content-digest", "@method", "@target-uri")

	ctxKeyAddDebug = struct{}{}
)

// KeySpec is the per-key information needed to verify a signature.
type KeySpec struct {
	KeyID  string
	Algo   Algorithm
	PubKey crypto.PublicKey
	Secret []byte // shared secret for symmetric algorithms
}

// KeySpec implements KeySpecer
func (ks KeySpec) KeySpec() (KeySpec, error) {
	return ks, nil
}

// KeySpecer should be implemented by your key/credential store
type KeySpecer interface {
	KeySpec() (KeySpec, error)
}

type KeyErrorReason string
type KeyError struct {
	error
	Reason  KeyErrorReason
	Message string
}

type KeyFetcher interface {
	// FetchByKeyID looks up a KeySpec from the 'keyid' metadata parameter on the signature.
	FetchByKeyID(ctx context.Context, rh http.Header, keyID string) (KeySpecer, error)
	// Fetch looks up a KeySpec when the keyid is not in the signature.
	Fetch(ctx context.Context, rh http.Header, md MetadataProvider) (KeySpecer, error)
}

// VerifyProfile sets the parameters for a fully valid request or response.
// A valid signature is a relatively easy accomplishment. Did the signature include all the important parts of the request? Did it use a strong enough algorithm? Was it signed 41 days ago?  There are choices to make about what constitutes a valid signed request or response beyond just a verified signature.
type VerifyProfile struct {
	SignatureLabel     string // Which signature this profile applies to. '*' applies to all
	RequiredFields     []SignedField
	RequiredMetadata   []Metadata
	DisallowedMetadata []Metadata
	AllowedAlgorithms  []Algorithm // Which algorithms are allowed, either from keyid meta or in the KeySpec

	// Timing enforcement options
	DisableTimeEnforcement       bool          // If true do no time enforcement on any fields
	DisableExpirationEnforcement bool          // If expiration is present default to enforce
	CreatedValidDuration         time.Duration // Duration allowed for between time.Now and the created time
	ExpiredSkew                  time.Duration // Maximum duration allowed between time.Now and the expired time
	DateFieldSkew                time.Duration // Maximum duration allowed between Date field and created
}

type VerifyResult struct {
	Verified  bool
	Label     string
	KeySpecer KeySpecer
	DebugInfo VerifyDebugInfo // Present if the verifier debug flag is set and the signature was valid.
	MetadataProvider
}

type VerifyDebugInfo struct {
	SignatureBase string // The signature base derived from the request.
}

type Verifier struct {
	keys    KeyFetcher
	profile VerifyProfile
}

// Verify validates the signatures in a request and ensured the signature meets the required profile.
func Verify(req *http.Request, kf KeyFetcher, profile VerifyProfile) (VerifyResult, error) {
	ver, err := NewVerifier(kf, profile)
	if err != nil {
		return VerifyResult{}, err
	}
	return ver.Verify(req)
}

func VerifyResponse(resp *http.Response, kf KeyFetcher, profile VerifyProfile) (VerifyResult, error) {
	ver, err := NewVerifier(kf, profile)
	if err != nil {
		return VerifyResult{}, err
	}
	return ver.VerifyResponse(resp)
}

func NewVerifier(kf KeyFetcher, profile VerifyProfile) (*Verifier, error) {
	if kf == nil {
		return nil, newError(ErrSigKeyFetch, "KeyFetcher cannot be nil")
	}
	return &Verifier{
		keys:    kf,
		profile: profile,
	}, nil
}

// Verify verifies the signature(s) in an http request. Any invalid signature will return an error.
// A valid VerifyResult is returned even if error is also returned.
func (ver *Verifier) Verify(req *http.Request) (VerifyResult, error) {
	return ver.verify(httpMessage{
		Req: req,
	})
}

func (ver *Verifier) VerifyResponse(resp *http.Response) (VerifyResult, error) {
	return ver.verify(httpMessage{
		IsResponse: true,
		Resp:       resp,
	})
}

// verify verifies the request or response.
func (ver *Verifier) verify(hrr httpMessage) (VerifyResult, error) {
	vr := VerifyResult{}

	/* calculate content digest if needed */
	if hrr.Headers().Get("Content-Digest") != "" {
		digestAlgo, expectedDigest, err := getSupportedDigestFromHeader(hrr.Headers().Values("Content-Digest"))
		if err != nil {
			return vr, err
		}

		di, err := digestBody(digestAlgo, hrr.Body())
		if err != nil {
			return vr, err
		}
		hrr.SetBody(di.NewBody)
		if !bytes.Equal(expectedDigest, di.Digest) {
			return vr, newError(ErrNoSigWrongDigest, "Digest does not match")
		}
	}

	/* parse and extract the signature */
	sigsfv, err := parseSignaturesFromRequest(hrr.Headers())
	if err != nil {
		return vr, err
	}

	sig, err := unmarshalSignature(sigsfv, ver.profile.SignatureLabel)
	if err != nil {
		return vr, err
	}
	vr.Label = sig.Label
	vr.MetadataProvider = sig.Input.MetadataValues

	/* verify and validate */
	base, err := calculateSignatureBase(hrr, sig.Input)
	if err != nil {
		return vr, err
	}

	if hrr.isDebug() {
		vr.DebugInfo = VerifyDebugInfo{
			SignatureBase: string(base.base),
		}
	}
	keyspec, err := ver.verifySignature(hrr, sig, base)
	vr.KeySpecer = keyspec
	if err != nil {
		return vr, err
	}

	if err := ver.profile.validate(sig); err != nil {
		return vr, err
	}

	return VerifyResult{
		Verified:         true,
		Label:            sig.Label,
		KeySpecer:        keyspec,
		MetadataProvider: sig.Input.MetadataValues,
		DebugInfo:        vr.DebugInfo,
	}, nil
}

type extractedSignature struct {
	Label     string
	Signature []byte
	Input     sigBaseInput
}

// signaturesSFV is the structured field value representation of all the signatures of a request.
type signaturesSFV struct {
	SigInputs *sfv.Dictionary
	Sigs      *sfv.Dictionary
}

func parseSignaturesFromRequest(headers http.Header) (signaturesSFV, error) {
	psd := signaturesSFV{}
	/* Pull signature and signature-input header */
	sigHeader := headers.Get("signature")
	if sigHeader == "" {
		return psd, newError(ErrNoSigMissingSignature, "Missing signature header")
	}
	sigInputHeader := headers.Get("signature-input")
	if sigInputHeader == "" {
		return psd, newError(ErrNoSigMissingSignature, "Missing signature-input header")
	}

	/* Parse headers into their appropriate HTTP structured field values */
	// signature-input must be a HTTP structured field value of type Dictionary.
	var err error
	psd.SigInputs, err = sfv.UnmarshalDictionary([]string{sigInputHeader})
	if err != nil {
		return psd, newError(ErrNoSigInvalidSignature, "Invalid signature-input header. Not a valid Dictionary", err)
	}
	// signature must be a HTTP structured field value of type Dictionary.
	psd.Sigs, err = sfv.UnmarshalDictionary([]string{sigHeader})
	if err != nil {
		return psd, newError(ErrNoSigInvalidSignature, "Invalid signature header. Not a valid Dictionary", err)
	}
	return psd, nil
}

// unmarshalSignature unmarshals a signature from hhtp structured field value (sfv) format.
func unmarshalSignature(sigs signaturesSFV, label string) (extractedSignature, error) {
	sigInfo := extractedSignature{
		Label: label,
	}

	sigMember, found := sigs.Sigs.Get(label)
	if !found {
		return sigInfo, newError(ErrNoSigMissingSignature, fmt.Sprintf("The signature for label '%s' not found", label))
	}
	sigInputMember, found := sigs.SigInputs.Get(label)
	if !found {
		return sigInfo, newError(ErrNoSigMissingSignature, fmt.Sprintf("The signature-input for label '%s' not found", label))
	}

	// The signature must be of sfv type 'Item'
	sigItem, isItem := sigMember.(sfv.Item)
	if !isItem {
		return sigInfo, newError(ErrSigInvalidSignature, fmt.Sprintf("The signature for label '%s' must be type Item. It was type %T", label, sigMember))
	}
	// Signatures must be byte sequences. The sfv library uses []byte for byte sequences.
	sigBytes, isByteSequence := sigItem.Value.([]byte)
	if !isByteSequence {
		return sigInfo, newError(ErrSigInvalidSignature, fmt.Sprintf("The signature for label '%s' was not a byte sequence. It was type %T", label, sigItem.Value))

	}
	sigInfo.Signature = sigBytes

	// The signature input must be of sfv type InnerList
	sigInputList, isList := sigInputMember.(sfv.InnerList)
	if !isList {
		return sigInfo, newError(ErrSigInvalidSignature, fmt.Sprintf("The signature-input for label '%s' must be type InnerList. It was type '%T'.", label, sigInputMember))

	}
	cIDs := []componentID{}
	for _, componentItem := range sigInputList.Items {
		name, ok := componentItem.Value.(string)
		if !ok {
			return sigInfo, newError(ErrSigInvalidSignature, fmt.Sprintf("signature components must be string types"))
		}
		cIDs = append(cIDs, componentID{
			Name: name,
			Item: componentItem,
		})
	}
	mds := []Metadata{}
	for _, name := range sigInputList.Params.Names() {
		mds = append(mds, Metadata(name))
	}
	sigInfo.Input = sigBaseInput{
		Components:     cIDs,
		MetadataParams: mds,
		MetadataValues: metadataProviderFromParams{sigInputList.Params},
	}

	return sigInfo, nil
}

func (ver *Verifier) verifySignature(r httpMessage, sig extractedSignature, base signatureBase) (KeySpecer, error) {
	var specer KeySpecer
	var ks KeySpec
	var err error
	// Get keyspec
	if slices.Contains(sig.Input.MetadataParams, MetaKeyID) {
		keyid, err := sig.Input.MetadataValues.KeyID()
		if err != nil {
			return nil, newError(ErrSigKeyFetch, "Could not get keyid from signature metadata", err)
		}

		specer, err = ver.keys.FetchByKeyID(r.Context(), r.Headers(), keyid)
		if err != nil {
			return nil, newError(ErrSigKeyFetch, fmt.Sprintf("Failed to fetch key for keyid '%s'", keyid), err)
		}
		ks, err = specer.KeySpec()
		if err != nil {
			return nil, newError(ErrSigKeyFetch, fmt.Sprintf("Failed to fetch key for keyid '%s'", keyid), err)
		}
	} else {
		specer, err = ver.keys.Fetch(r.Context(), r.Headers(), sig.Input.MetadataValues)
		if err != nil {
			return specer, newError(ErrSigKeyFetch, fmt.Sprintf("Failed to fetch key for signature without a keyid and with label '%s'\n", sig.Label), err)
		}
		ks, err = specer.KeySpec()
		if err != nil {
			return specer, newError(ErrSigKeyFetch, fmt.Sprintf("Failed to fetch key for signature without a keyid and with label '%s'\n", sig.Label), err)
		}
	}

	switch ks.Algo {
	case Algo_RSA_PSS_SHA512:
		if rsapub, ok := ks.PubKey.(*rsa.PublicKey); ok {
			opts := &rsa.PSSOptions{
				SaltLength: 64,
				Hash:       crypto.SHA512,
			}
			msgHash := sha512.Sum512(base.base)
			err := rsa.VerifyPSS(rsapub, crypto.SHA512, msgHash[:], sig.Signature, opts)
			if err != nil {
				return specer, newError(ErrSigVerification, fmt.Sprintf("Signature did not verify for algo '%s'", ks.Algo), err)
			}
			return specer, nil
		} else {
			return specer, newError(ErrSigPublicKey, fmt.Sprintf("Invalid public key. Requires rsa.PublicKey but got type: %T", ks.PubKey))
		}
	case Algo_RSA_v1_5_sha256:
		if rsapub, ok := ks.PubKey.(*rsa.PublicKey); ok {
			msgHash := sha256.Sum256(base.base)
			err := rsa.VerifyPKCS1v15(rsapub, crypto.SHA256, msgHash[:], sig.Signature)
			if err != nil {
				return specer, newError(ErrSigVerification, fmt.Sprintf("Signature did not verify for algo '%s'", ks.Algo), err)
			}
			return specer, nil
		} else {
			return specer, newError(ErrSigPublicKey, fmt.Sprintf("Invalid public key. Requires rsa.PublicKey but got type: %T", ks.PubKey))
		}
	case Algo_HMAC_SHA256:
		if len(ks.Secret) == 0 {
			return specer, newError(ErrInvalidSignatureOptions, fmt.Sprintf("No secret provided for symmetric algorithm '%s'", Algo_HMAC_SHA256))
		}
		msgHash := hmac.New(sha256.New, ks.Secret)
		msgHash.Write(base.base) // write does not return an error per hash.Hash documentation
		calcualtedSignature := msgHash.Sum(nil)
		if !hmac.Equal(calcualtedSignature, sig.Signature) {
			return specer, newError(ErrSigVerification, fmt.Sprintf("Signature did not verify for algo '%s'", ks.Algo), err)
		}
	case Algo_ECDSA_P256_SHA256:
		if epub, ok := ks.PubKey.(*ecdsa.PublicKey); ok {
			if len(sig.Signature) != 64 {
				return specer, newError(ErrSigInvalidSignature, fmt.Sprintf("Signature must be 64 bytes for algorithm '%s'", Algo_ECDSA_P256_SHA256))
			}
			msgHash := sha256.Sum256(base.base)
			// Concatenate r and s to form the signature as per the spec. r and s and *not* ANS1 encoded.
			r := new(big.Int)
			r.SetBytes(sig.Signature[0:32])
			s := new(big.Int)
			s.SetBytes(sig.Signature[32:64])
			if !ecdsa.Verify(epub, msgHash[:], r, s) {
				return specer, newError(ErrSigVerification, fmt.Sprintf("Signature did not verify for algo '%s'", ks.Algo), err)
			}
		} else {
			return specer, newError(ErrSigPublicKey, fmt.Sprintf("Invalid public key. Requires *ecdsa.PublicKey but got type: %T", ks.PubKey))
		}
	case Algo_ECDSA_P384_SHA384:
		if epub, ok := ks.PubKey.(*ecdsa.PublicKey); ok {
			if len(sig.Signature) != 96 {
				return specer, newError(ErrSigInvalidSignature, fmt.Sprintf("Signature must be 96 bytes for algorithm '%s'", Algo_ECDSA_P256_SHA256))
			}
			msgHash := sha512.Sum384(base.base)
			// Concatenate r and s to form the signature as per the spec. r and s and *not* ANS1 encoded.
			r := new(big.Int)
			r.SetBytes(sig.Signature[0:48])
			s := new(big.Int)
			s.SetBytes(sig.Signature[48:96])
			if !ecdsa.Verify(epub, msgHash[:], r, s) {
				return specer, newError(ErrSigVerification, fmt.Sprintf("Signature did not verify for algo '%s'", ks.Algo), err)
			}
		} else {
			return specer, newError(ErrSigPublicKey, fmt.Sprintf("Invalid public key. Requires *ecdsa.PublicKey but got type: %T", ks.PubKey))
		}
	case Algo_ED25519:
		if edpubkey, ok := ks.PubKey.(ed25519.PublicKey); ok {
			if !ed25519.Verify(edpubkey, base.base, sig.Signature) {
				return specer, newError(ErrSigVerification, fmt.Sprintf("Signature did not verify for algo '%s'", ks.Algo), err)
			}
		} else {
			return specer, newError(ErrSigPublicKey, fmt.Sprintf("Invalid public key. Requires ed25519.PublicKey but got type: %T", ks.PubKey))
		}
	default:
		return specer, newError(ErrSigUnsupportedAlgorithm, fmt.Sprintf("Invalid verification algorithm '%s'", ks.Algo))
	}
	return specer, nil
}

func (vp VerifyProfile) validate(sig extractedSignature) error {
	return nil
}

type metadataProviderFromParams struct {
	Params *sfv.Params
}

func (mp metadataProviderFromParams) Created() (int, error) {
	if val, ok := mp.Params.Get(string(MetaCreated)); ok {
		return int(val.(int64)), nil
	}
	return 0, fmt.Errorf("No created value")
}

func (mp metadataProviderFromParams) Expires() (int, error) {
	if val, ok := mp.Params.Get(string(MetaExpires)); ok {
		return int(val.(int64)), nil
	}
	return 0, fmt.Errorf("No expires value")
}

func (mp metadataProviderFromParams) Nonce() (string, error) {
	if val, ok := mp.Params.Get(string(MetaNonce)); ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No nonce value")
}

func (mp metadataProviderFromParams) Alg() (string, error) {
	if val, ok := mp.Params.Get(string(MetaAlgorithm)); ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No alg value")
}

func (mp metadataProviderFromParams) KeyID() (string, error) {
	if val, ok := mp.Params.Get(string(MetaKeyID)); ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No keyid value")
}

func (mp metadataProviderFromParams) Tag() (string, error) {
	if val, ok := mp.Params.Get(string(MetaTag)); ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No tag value")
}

func SetAddDebugInfo(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyAddDebug, true)
}
