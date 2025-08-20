package keyutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

const (
	KeyTypeEC  = "EC"
	KeyTypeOct = "oct"
)

func ReadJWKFile(jwkFile string) (JWK, error) {
	keyBytes, err := os.ReadFile(jwkFile)
	if err != nil {
		return JWK{}, fmt.Errorf("Failed to read jwk key file '%s': %w", jwkFile, err)
	}
	return ReadJWK(keyBytes)
}

func ReadJWK(jwkBytes []byte) (JWK, error) {
	base := jwk{}
	err := json.Unmarshal(jwkBytes, &base)
	if err != nil {
		return JWK{}, fmt.Errorf("Failed to json parse JWK: %w", err)
	}
	jwk := JWK{
		KeyType:   base.KeyType,
		Algorithm: base.Algo,
		KeyID:     base.KeyID,
	}
	switch base.KeyType {
	case KeyTypeEC:
		jec := jwkEC{}
		err := json.Unmarshal(jwkBytes, &jec)
		if err != nil {
			return JWK{}, fmt.Errorf("Failed to json parse JWK: %w", err)
		}
		jwk.jwtImpl = jec
	case KeyTypeOct:
		jsym := jwkSymmetric{}
		err := json.Unmarshal(jwkBytes, &jsym)
		if err != nil {
			return JWK{}, fmt.Errorf("Failed to json parse JWK: %w", err)
		}
		jwk.jwtImpl = jsym
	default:
		return JWK{}, fmt.Errorf("Unsupported key type/kty - '%s'", base.KeyType)
	}

	return jwk, nil
}

// ReadJWKFromPEM converts a PEM encoded private key to JWK. 'kty' is set based on the passed in PrivateKey type.
func ReadJWKFromPEM(pkeyBytes []byte) (JWK, error) {
	pkey, err := ReadPrivateKey(pkeyBytes)
	if err != nil {
		return JWK{}, err
	}
	return FromPrivateKey(pkey)
}

// FromPrivateKey creates a JWK from a crypto.PrivateKey. 'kty' is set based on the passed in PrivateKey.
func FromPrivateKey(pkey crypto.PrivateKey) (JWK, error) {
	switch key := pkey.(type) {
	case *ecdsa.PrivateKey:
		jec := jwkEC{
			jwk: jwk{
				KeyType: KeyTypeEC,
			},
			Curve: key.Curve.Params().Name,
			X:     &octet{*key.X},
			Y:     &octet{*key.Y},
			D:     &octet{*key.D},
		}

		return JWK{
			KeyType: KeyTypeEC,
			jwtImpl: jec,
		}, nil
	default:
		return JWK{}, fmt.Errorf("Unsupported private key type '%T'", pkey)
	}
}

// JWK provides basic data and usage for a JWK.
type JWK struct {
	// Common fields are duplicated as struct members for better usability.
	KeyType   string // 'kty' - "EC", "RSA", "oct"
	Algorithm string // 'alg'
	KeyID     string // 'kid'
	jwtImpl   any    // the specific implementation of JWK based on KeyType.
}

func (ji *JWK) PublicKey() (crypto.PublicKey, error) {
	switch ji.KeyType {
	case ji.KeyType: // ECC
		if jec, ok := ji.jwtImpl.(jwkEC); ok {
			return jec.PublicKey()
		}
	}

	return nil, fmt.Errorf("Unsupported key type for PublicKey - '%s'", ji.KeyType)
}

func (ji *JWK) PublicKeyJWK() (JWK, error) {
	switch ji.KeyType {
	case KeyTypeEC:
		if jec, ok := ji.jwtImpl.(jwkEC); ok {
			jec.jwk.Algo = ji.Algorithm
			jec.jwk.KeyID = ji.KeyID
			return jec.PublicKeyJWK()
		}
	}

	return JWK{}, fmt.Errorf("Unsupported key type for PublicKey'%s'", ji.KeyType)
}

func (ji *JWK) PrivateKey() (crypto.PrivateKey, error) {
	switch ji.KeyType {
	case KeyTypeEC:
		if jec, ok := ji.jwtImpl.(jwkEC); ok {
			return jec.PrivateKey()
		}
	}
	return nil, fmt.Errorf("Unsupported key type PrivateKey - '%s'", ji.KeyType)
}

func (ji *JWK) SecretKey() ([]byte, error) {
	switch ji.KeyType {
	case KeyTypeOct:
		if jsym, ok := ji.jwtImpl.(jwkSymmetric); ok {
			return jsym.Key(), nil
		}

	}
	return nil, fmt.Errorf("Unsupported key type for Secret '%s'", ji.KeyType)
}

// octet represents the data for base64 URL encoded data as specified by JWKs.
type octet struct {
	big.Int
}

func (ob octet) MarshalJSON() ([]byte, error) {
	out := fmt.Sprintf("\"%s\"", base64.RawURLEncoding.EncodeToString(ob.Bytes()))
	return []byte(out), nil
}

func (ob *octet) UnmarshalJSON(data []byte) error {
	// data is the json value and must be unmarshaled into a go string first
	encoded := ""
	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return err
	}

	rawBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("Failed to base64 decode: %w", err)
	}

	x := new(big.Int)
	x.SetBytes(rawBytes)
	*ob = octet{*x}

	return nil
}

type jwk struct {
	KeyType string `json:"kty"`           // kty  algorithm family used with the key such as "RSA" or "EC".
	Algo    string `json:"alg,omitempty"` // alg identifies the algorithm intended for use with the key.
	KeyID   string `json:"kid,omitempty"` // Used to match a specific key
}

type jwkEC struct {
	jwk
	Curve string `json:"crv"`         // The curve used with the key e.g. P-256
	X     *octet `json:"x"`           // x coordinate of the curve.
	Y     *octet `json:"y"`           // y coordinate of the curve.
	D     *octet `json:"d,omitempty"` // For private keys.
}

func (ec *jwkEC) params() (crv elliptic.Curve, byteLen int, e error) {
	switch ec.Curve {
	case "P-256":
		crv = elliptic.P256()
	case "P-384":
		crv = elliptic.P384()
	case "P-521":
		crv = elliptic.P521()
	default:
		return nil, 0, fmt.Errorf("Unsupported ECC curve '%s'", ec.Curve)
	}
	return crv, crv.Params().BitSize / 8, nil
}

func (ec *jwkEC) PublicKey() (*ecdsa.PublicKey, error) {
	crv, byteLen, err := ec.params()
	if err != nil {
		return nil, err
	}

	if len(ec.X.Bytes()) != byteLen {
		return nil, fmt.Errorf("X coordinate must be %d byte length for curve '%s'. Got '%d'", byteLen, ec.Curve, len(ec.X.Bytes()))
	}
	if len(ec.Y.Bytes()) != byteLen {
		return nil, fmt.Errorf("Y coordinate must be %d byte length for curve '%s'. Got '%d'", byteLen, ec.Curve, len(ec.Y.Bytes()))
	}

	return &ecdsa.PublicKey{
		Curve: crv,
		X:     &ec.X.Int,
		Y:     &ec.Y.Int,
	}, nil
}

func (ec *jwkEC) PublicKeyJWK() (JWK, error) {
	return JWK{
		KeyType:   ec.KeyType,
		Algorithm: ec.Algo,
		KeyID:     ec.KeyID,
		jwtImpl: jwkEC{
			jwk:   ec.jwk,
			Curve: ec.Curve,
			X:     ec.X,
			Y:     ec.Y,
		},
	}, nil
}

func (ec *jwkEC) PrivateKey() (*ecdsa.PrivateKey, error) {
	if ec.D == nil {
		return nil, fmt.Errorf("JWK does not contain a private key")
	}
	pubkey, err := ec.PublicKey()
	if err != nil {
		return nil, err
	}
	_, byteLen, err := ec.params()
	if err != nil {
		return nil, err
	}

	if len(ec.D.Bytes()) != byteLen {
		return nil, fmt.Errorf("D coordinate must be %d byte length for curve '%s'. Got '%d'", byteLen, ec.Curve, len(ec.D.Bytes()))
	}

	return &ecdsa.PrivateKey{
		PublicKey: *pubkey,
		D:         &ec.D.Int,
	}, nil
}

type jwkSymmetric struct {
	jwk
	K *octet `json:"k" ` // Symmetric key
}

func (js *jwkSymmetric) Key() []byte {
	return js.K.Bytes()
}

func (j JWK) MarshalJSON() ([]byte, error) {
	// Set the Algo and KeyID in case the JWK fields have changed
	switch jt := j.jwtImpl.(type) {
	case jwkEC:
		jt.jwk.Algo = j.Algorithm
		jt.jwk.KeyID = j.KeyID
		return json.Marshal(jt)
	case jwkSymmetric:
		jt.jwk.Algo = j.Algorithm
		jt.jwk.KeyID = j.KeyID
		return json.Marshal(jt)
	}

	return json.Marshal(j.jwtImpl)
}
