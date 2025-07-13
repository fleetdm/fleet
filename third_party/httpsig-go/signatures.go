package httpsig

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"time"

	sfv "github.com/dunglas/httpsfv"
)

// derived component names
type derived string

const (
	sigparams derived = "@signature-params"
	method    derived = "@method"
	path      derived = "@path"
	targetURI derived = "@target-uri"
	authority derived = "@authority"
)

// MetadataProvider allows customized functions for metadata parameter values. Not needed for default usage.
type MetadataProvider interface {
	Created() (int, error)
	Expires() (int, error)
	Nonce() (string, error)
	Alg() (string, error)
	KeyID() (string, error)
	Tag() (string, error)
}

type signatureBase struct {
	base           []byte // The full signature base. Use this as input to signing and verification
	signatureInput string // The signature-input line
}

type sigParameters struct {
	Base       sigBaseInput
	Algo       Algorithm
	Label      string
	PrivateKey crypto.PrivateKey
	Secret     []byte
}

func sign(hrr httpMessage, sp sigParameters) error {
	base, err := calculateSignatureBase(hrr, sp.Base)
	if err != nil {
		return err
	}

	var sigBytes []byte
	switch sp.Algo {

	case Algo_RSA_PSS_SHA512:
		msgHash := sha512.Sum512(base.base)
		opts := &rsa.PSSOptions{
			SaltLength: 64,
			Hash:       crypto.SHA512,
		}
		switch rsapk := sp.PrivateKey.(type) {
		case *rsa.PrivateKey:
			sigBytes, err = rsa.SignPSS(rand.Reader, rsapk, crypto.SHA512, msgHash[:], opts)
			if err != nil {
				return err
			}
		case crypto.Signer:
			sigBytes, err = rsapk.Sign(rand.Reader, msgHash[:], crypto.SHA512)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Invalid private key. Requires *rsa.PrivateKey or crypto.Signer: %T", sp.PrivateKey)
		}
	case Algo_RSA_v1_5_sha256:
		msgHash := sha256.Sum256(base.base)
		switch rsapk := sp.PrivateKey.(type) {
		case *rsa.PrivateKey:
			sigBytes, err = rsa.SignPKCS1v15(rand.Reader, rsapk, crypto.SHA256, msgHash[:])
			if err != nil {
				return err
			}
		case crypto.Signer:
			sigBytes, err = rsapk.Sign(rand.Reader, msgHash[:], crypto.SHA256)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Invalid private key. Requires *rsa.PrivateKey or crypto.Signer: %T", sp.PrivateKey)
		}
	case Algo_ECDSA_P256_SHA256:
		msgHash := sha256.Sum256(base.base)
		switch eccpk := sp.PrivateKey.(type) {
		case *ecdsa.PrivateKey:
			r, s, err := ecdsa.Sign(rand.Reader, eccpk, msgHash[:])
			if err != nil {
				return newError(ErrInternal, "Failed to sign with ecdsa private key", err)
			}
			// Concatenate r and s to make the signature as per the spec. r and s are *not* encoded in ASN1 format
			sigBytes = make([]byte, 64)
			r.FillBytes(sigBytes[0:32])
			s.FillBytes(sigBytes[32:64])
		case crypto.Signer:
			sigBytes, err = eccpk.Sign(rand.Reader, msgHash[:], crypto.SHA256)
			if err != nil {
				return newError(ErrInternal, "Failed to sign with ecdsa custom signer", err)
			}
		default:
			return fmt.Errorf("Invalid private key. Requires *ecdsa.PrivateKey or crypto.Signer: %T", sp.PrivateKey)
		}
	case Algo_ECDSA_P384_SHA384:
		msgHash := sha512.Sum384(base.base)
		switch eccpk := sp.PrivateKey.(type) {
		case *ecdsa.PrivateKey:
			r, s, err := ecdsa.Sign(rand.Reader, eccpk, msgHash[:])
			if err != nil {
				return newError(ErrInternal, "Failed to sign with ecdsa private key", err)
			}
			// Concatenate r and s to make the signature as per the spec. r and s are *not* encoded in ASN1 format
			sigBytes = make([]byte, 96)
			r.FillBytes(sigBytes[0:48])
			s.FillBytes(sigBytes[48:96])
		case crypto.Signer:
			sigBytes, err = eccpk.Sign(rand.Reader, msgHash[:], crypto.SHA384)
			if err != nil {
				return newError(ErrInternal, "Failed to sign with ecdsa custom signer", err)
			}
		default:
			return fmt.Errorf("Invalid private key. Requires *ecdsa.PrivateKey or crypto.Signer: %T", sp.PrivateKey)
		}
	case Algo_ED25519:
		switch edpk := sp.PrivateKey.(type) {
		case ed25519.PrivateKey:
			sigBytes = ed25519.Sign(edpk, base.base)
		case crypto.Signer:
			sigBytes, err = edpk.Sign(rand.Reader, base.base, crypto.Hash(0))
			if err != nil {
				return newError(ErrInternal, "Failed to sign with ed25519 custom signer", err)
			}
		default:
			return fmt.Errorf("Invalid private key. Requires ed25519.PrivateKey or crypto.Signer: %T", sp.PrivateKey)
		}
	case Algo_HMAC_SHA256:
		if len(sp.Secret) == 0 {
			return newError(ErrInvalidSignatureOptions, fmt.Sprintf("No secret provided for symmetric algorithm '%s'", Algo_HMAC_SHA256))
		}
		msgHash := hmac.New(sha256.New, sp.Secret)
		msgHash.Write(base.base) // write does not return an error per hash.Hash documentation
		sigBytes = msgHash.Sum(nil)
	default:
		return newError(ErrInvalidSignatureOptions, fmt.Sprintf("Signing algorithm not supported: '%s'", sp.Algo))
	}
	sigField := sfv.NewDictionary()
	sigField.Add(sp.Label, sfv.NewItem(sigBytes))
	signature, err := sfv.Marshal(sigField)
	if err != nil {
		return newError(ErrInternal, fmt.Sprintf("Failed to marshal signature for label '%s'", sp.Label), err)
	}
	hrr.Headers().Set("Signature-Input", fmt.Sprintf("%s=%s", sp.Label, base.signatureInput))
	hrr.Headers().Set("Signature", signature)
	return nil
}

func timestamp(nowtime func() time.Time) int {
	return int(nowtime().Unix())
}
