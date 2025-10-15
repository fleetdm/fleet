package types

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// RenewalExtensionOID is the custom OID for the renewal extension. 63991 is Fleet's IANA private enterprise number
// 1.3.6.1.4.1.63991.1.1
var RenewalExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 63991, 1, 1}

// RenewalData represents the JSON data in the renewal extension
type RenewalData struct {
	SerialNumber string `json:"sn"`  // Hex-encoded serial number of the old certificate
	Signature    string `json:"sig"` // Base64-encoded ECDSA signature
}

type HostIdentityCertificate struct {
	SerialNumber  uint64    `db:"serial"`
	CommonName    string    `db:"name"`
	HostID        *uint     `db:"host_id"`
	NotValidAfter time.Time `db:"not_valid_after"`
	PublicKeyRaw  []byte    `db:"public_key_raw"`
	CreatedAt     time.Time `db:"created_at"`
}

func (h *HostIdentityCertificate) UnmarshalPublicKey() (*ecdsa.PublicKey, error) {
	if len(h.PublicKeyRaw) == 0 || h.PublicKeyRaw[0] != 4 { // 0x04 means this is the raw representation
		return nil, errors.New("unsupported EC point format")
	}

	curve, err := guessCurve(h.PublicKeyRaw)
	if err != nil {
		return nil, err
	}

	byteLen := (len(h.PublicKeyRaw) - 1) / 2
	x := new(big.Int).SetBytes(h.PublicKeyRaw[1 : 1+byteLen])
	y := new(big.Int).SetBytes(h.PublicKeyRaw[1+byteLen:])

	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func guessCurve(raw []byte) (elliptic.Curve, error) {
	switch len(raw) {
	case 65: // 0x04 + 32 + 32
		return elliptic.P256(), nil
	case 97: // 0x04 + 48 + 48
		return elliptic.P384(), nil
	default:
		return nil, fmt.Errorf("unknown curve: unsupported key length %d", len(raw))
	}
}

func CreateECDSAPublicKeyRaw(key *ecdsa.PublicKey) ([]byte, error) {
	var keySize int
	switch key.Curve {
	case elliptic.P256():
		keySize = 32
	case elliptic.P384():
		keySize = 48
	default:
		return nil, fmt.Errorf("unsupported curve: %s", key.Curve.Params().Name)
	}

	// Pad X and Y coordinates to the expected size
	xBytes := make([]byte, keySize)
	yBytes := make([]byte, keySize)
	key.X.FillBytes(xBytes)
	key.Y.FillBytes(yBytes)

	pubKeyRaw := append([]byte{0x04}, append(xBytes, yBytes...)...)
	return pubKeyRaw, nil
}

// CreatePublicKeyRaw creates a raw byte representation of a public key.
// For ECC keys, it returns the uncompressed point format (0x04 prefix + X + Y).
// For RSA keys, it returns the PKIX, ASN.1 DER encoded public key.
func CreatePublicKeyRaw(key any) ([]byte, error) {
	switch k := key.(type) {
	case *ecdsa.PublicKey:
		return CreateECDSAPublicKeyRaw(k)
	case *rsa.PublicKey:
		// For RSA keys, marshal to PKIX format (standard ASN.1 DER encoding)
		pubKeyBytes, err := x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("marshaling RSA public key: %w", err)
		}
		return pubKeyBytes, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %T", key)
	}
}
