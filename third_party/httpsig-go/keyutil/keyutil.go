/*
Package keyutil provides some basic PEM and JWK key handling without dependencies.  It is not meant as a replacement for a full key handling library.
*/
package keyutil

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"os"
)

var (
	oidPublicKeyRSAPSS = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
)

// MustReadPublicKeyFile reads a PEM encoded public key file or panics
func MustReadPublicKeyFile(pubkeyFile string) crypto.PublicKey {
	pk, err := ReadPublicKeyFile(pubkeyFile)
	if err != nil {
		panic(err)
	}
	return pk
}

// ReadPublicKeyFile reads a PEM encdoded public key file and parses into crypto.PublicKey
func ReadPublicKeyFile(pubkeyFile string) (crypto.PublicKey, error) {
	keyBytes, err := os.ReadFile(pubkeyFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read public key file '%s': %w", pubkeyFile, err)
	}
	return ReadPublicKey(keyBytes)
}

// ReadPublicKey decodes a PEM encoded public key and parses into crypto.PublicKey
func ReadPublicKey(encodedPubkey []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(encodedPubkey)
	if block == nil {
		return nil, fmt.Errorf("Failed to PEM decode public key")
	}
	var key crypto.PublicKey
	var err error

	switch block.Type {
	case "PUBLIC KEY":
		key, err = x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		key, err = x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, fmt.Errorf("Unsupported pubkey format '%s'", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to parse public key with format '%s': %w", block.Type, err)
	}

	return key, nil
}

// MustReadPrivateKeyFile decodes a PEM encoded private key file and parses into a crypto.PrivateKey or panics.
func MustReadPrivateKeyFile(pkFile string) crypto.PrivateKey {
	pk, err := ReadPrivateKeyFile(pkFile)
	if err != nil {
		panic(err)
	}
	return pk
}

func MustReadPrivateKey(encodedPrivateKey []byte) crypto.PrivateKey {
	pkey, err := ReadPrivateKey(encodedPrivateKey)
	if err != nil {
		panic(err)
	}
	return pkey
}

// ReadPrivateKeyFile opens the given file and calls ReadPrivateKey to return a crypto.PrivateKey
func ReadPrivateKeyFile(pkFile string) (crypto.PrivateKey, error) {
	keyBytes, err := os.ReadFile(pkFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read private key file '%s': %w", pkFile, err)
	}
	return ReadPrivateKey(keyBytes)
}

// ReadPrivateKey decoded a PEM encoded private key and parses into a crypto.PrivateKey.
func ReadPrivateKey(encodedPrivateKey []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(encodedPrivateKey)

	if block == nil {
		return nil, fmt.Errorf("Failed to PEM decode private key")
	}

	var key crypto.PrivateKey
	var err error

	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			// Try to handle RSAPSS
			psskey, psserr := parseRSAPSS(block)
			if psserr == nil {
				// success
				key = psskey
				err = psserr
			}
		}
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("Unsupported private key format '%s'", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key with format '%s': %w", block.Type, err)
	}
	return key, nil
}

func parseRSAPSS(block *pem.Block) (crypto.PrivateKey, error) {
	// The rsa-pss key is PKCS8 encoded but the golang 1.19 parser doesn't recognize the algorithm and gives 'PKCS#8 wrapping contained private key with unknown algorithm: 1.2.840.113549.1.1.10

	// Instead do the asn1 unmarshaling and check here.
	pkcs8 := struct {
		Version    int
		Algo       pkix.AlgorithmIdentifier
		PrivateKey []byte
	}{}

	_, err := asn1.Unmarshal(block.Bytes, &pkcs8)
	if err != nil {
		return nil, fmt.Errorf("Failed to ans1 unmarshal private key: %w", err)
	}

	if !pkcs8.Algo.Algorithm.Equal(oidPublicKeyRSAPSS) {
		return nil, fmt.Errorf("PKCS#8 wrapping contained private key with unknown algorithm: %s", pkcs8.Algo.Algorithm)
	}
	return x509.ParsePKCS1PrivateKey(pkcs8.PrivateKey)
}
