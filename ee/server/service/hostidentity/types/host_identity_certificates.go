package types

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// RenewalExtensionOID is the custom OID for the renewal extension
// 1.3.6.1.4.1.99999.1.1
// TODO: Replace 99999 with Fleet's IANA private enterprise number once it is issued
var RenewalExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1, 1}

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
