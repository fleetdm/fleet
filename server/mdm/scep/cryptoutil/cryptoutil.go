package cryptoutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"errors"
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
