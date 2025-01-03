package cryptoutil

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// GenerateSubjectKeyID generates Subject Key Identifier (SKI) using SHA-256
// hash of the public key bytes according to RFC 7093 section 2.
func GenerateSubjectKeyID(pub crypto.PublicKey) ([]byte, error) {
	var pubBytes []byte
	var err error
	switch pub := pub.(type) {
	case *rsa.PublicKey:
		pubBytes, err = asn1.Marshal(*pub)
		if err != nil {
			return nil, err
		}
	case *ecdsa.PublicKey:
		pubBytes = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	default:
		return nil, errors.New("only ECDSA and RSA public keys are supported")
	}

	hash := sha256.Sum256(pubBytes)

	// According to RFC 7093, The keyIdentifier is composed of the leftmost
	// 160-bits of the SHA-256 hash of the value of the BIT STRING
	// subjectPublicKey (excluding the tag, length, and number of unused bits).
	return hash[:20], nil
}

func ParsePrivateKey(ctx context.Context, privKeyPEM []byte, keyName string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(privKeyPEM)
	if block == nil {
		return nil, ctxerr.Errorf(ctx, "failed to decode %s", keyName)
	}

	// The code below is based on tls.parsePrivateKey
	// https://cs.opensource.google/go/go/+/release-branch.go1.23:src/crypto/tls/tls.go;l=355-372
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
			return key, nil
		default:
			return nil, ctxerr.Errorf(ctx, "unmarshaled PKCS8 %s is not an RSA, ECDSA, or Ed25519 private key", keyName)
		}
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, ctxerr.Errorf(ctx, "failed to parse %s of type %s", keyName, block.Type)
}
