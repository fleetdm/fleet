package keyidentifier

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"golang.org/x/crypto/ssh"
)

type KeyInfo struct {
	Type              string // Key type. rsa/dsa/etc
	Format            string // file format
	Bits              int    // number of bits in the key
	Encryption        string // key encryption algorythem
	Encrypted         *bool  // is the key encrypted
	Comment           string // comments attached to the key
	Parser            string // what parser we used to determine information
	FingerprintSHA256 string // the fingerprint of the key, as a SHA256 hash
	FingerprintMD5    string // the fingerprint of the key, as an MD5 hash
}

// keyidentifier attempts to identify a key. It uses a set of
// herusitics to try to guiess what kind, what size, and whether or
// not it's encrypted.
type KeyIdentifier struct {
	logger log.Logger
}

type Option func(*KeyIdentifier)

func WithLogger(logger log.Logger) Option {
	return func(kIdentifer *KeyIdentifier) {
		kIdentifer.logger = logger
	}
}

func New(opts ...Option) (*KeyIdentifier, error) {
	kIdentifer := &KeyIdentifier{
		logger: log.NewNopLogger(),
	}

	for _, opt := range opts {
		opt(kIdentifer)
	}

	return kIdentifer, nil
}

func (kIdentifer *KeyIdentifier) IdentifyFile(path string) (*KeyInfo, error) {
	level.Debug(kIdentifer.logger).Log(
		"msg", "starting a key identification",
		"file", path,
	)

	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	ki, err := kIdentifer.Identify(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("identifying key: %w", err)
	}

	return ki, nil
}

// Identify uses a manually curated set of heuristics to determine
// what kind of key something is. Generally speaking, we consider `err
// == nil` as success, and throw away errors as unparsable keys.
func (kIdentifer *KeyIdentifier) Identify(keyBytes []byte) (*KeyInfo, error) {
	level.Debug(kIdentifer.logger).Log(
		"msg", "starting a key identification",
		"file", "<bytestream>",
	)

	// Some magic strings for dispatching. These are the simplest.
	switch {
	case bytes.HasPrefix(keyBytes, []byte(ppkBegin)):
		return ParsePuttyPrivateKey(keyBytes)
	case bytes.HasPrefix(keyBytes, []byte(sshcomBegin)):
		return ParseSshComPrivateKey(keyBytes)
	case bytes.HasPrefix(keyBytes, []byte(ssh1LegacyBegin)):
		return ParseSsh1PrivateKey(keyBytes)
	}

	// If nothing else fits. treat it like a pem
	if ki, err := kIdentifer.attemptPem(keyBytes); err == nil {
		return ki, nil
	}

	// Out of options
	return nil, errors.New("Unable to parse key")
}

// attemptPem tries to decode the pem, and then work with the key. It's
// based on code from x/crypto's ssh.ParseRawPrivateKey, but more
// flexible in handling encryption and formats.
func (kIdentifer *KeyIdentifier) attemptPem(keyBytes []byte) (*KeyInfo, error) {
	ki := &KeyInfo{
		Format: "",
		Parser: "attemptPem",
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("pem could not parse")
	}

	ki.Encrypted = boolPtr(pemEncryptedBlock(block))

	level.Debug(kIdentifer.logger).Log(
		"msg", "pem decoded",
		"block type", block.Type,
	)

	switch block.Type {
	case "RSA PRIVATE KEY":
		ki.Type = ssh.KeyAlgoRSA
		ki.Format = "openssh"

		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			ki.Bits = len(key.PublicKey.N.Bytes()) * 8
		}

		return ki, nil

	case "PRIVATE KEY":
		// RFC5208 - https://tools.ietf.org/html/rfc5208
		ki.Encrypted = boolPtr(x509.IsEncryptedPEMBlock(block))
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			switch assertedKey := key.(type) {
			case *rsa.PrivateKey:
				ki.Bits = assertedKey.PublicKey.Size() * 8
				ki.Type = "rsa"
			case *ecdsa.PrivateKey:
				ki.Bits = assertedKey.PublicKey.Curve.Params().BitSize
				ki.Type = "ecdsa"
			}
		}
		return ki, nil

	case "EC PRIVATE KEY":
		// set the Type here, since parsing fails on encrypted keys
		ki.Type = "ecdsa"

		if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			ki.Bits = key.PublicKey.Curve.Params().BitSize
		} else {
			level.Debug(kIdentifer.logger).Log(
				"msg", "x509.ParseECPrivateKey failed to parse",
				"err", err,
			)
		}

		return ki, nil

	case "DSA PRIVATE KEY":
		if key, err := ssh.ParseDSAPrivateKey(block.Bytes); err == nil {
			ki.Bits = len(key.PublicKey.Y.Bytes()) * 8
		}
		ki.Type = ssh.KeyAlgoDSA
		ki.Format = "openssh"
		return ki, nil

	case "OPENSSH PRIVATE KEY":
		if ki, err := ParseOpenSSHPrivateKey(block.Bytes); err == nil {
			return ki, nil
		}
	}

	// Unmatched. return what we have
	level.Debug(kIdentifer.logger).Log(
		"msg", "pem failed to match block type",
		"type", block.Type,
	)
	return ki, nil
}
