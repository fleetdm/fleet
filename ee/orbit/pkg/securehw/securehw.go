// Package securehw contains implementations of hardware-based cryptographic interfaces.
package securehw

import (
	"crypto"

	"github.com/rs/zerolog"
)

// SecureHW provides an interface for hardware-based
// cryptographic operations, such as those performed by a TPM (Trusted Platform Module).
type SecureHW interface {
	// CreateKey creates a new key in the SecureHW and returns a handle to it.
	// The implementation will automatically choose the best available key type,
	// preferring ECC P-384 if supported, otherwise falling back to ECC P-256.
	// Returns a Key interface that can be used for cryptographic operations.
	CreateKey() (Key, error)

	// LoadKey loads a previously created key from the public and private blobs saved to files.
	// The blobs are read from the file paths configured when creating the SecureHW instance.
	// The parent key is the hardcoded Storage Root Key (SRK) handle.
	LoadKey() (Key, error)

	// Close releases any resources held by the SecureHW.
	Close() error
}

// Key represents a key stored in a SecureHW that can perform cryptographic operations.
type Key interface {
	// Signer returns a crypto.Signer that uses this key for signing operations.
	// The returned Signer is safe for concurrent use.
	Signer() (crypto.Signer, error)

	// HTTPSigner returns a crypto.Signer configured for RFC 9421-compatible HTTP signatures.
	// The returned Signer produces fixed-width r||s format signatures.
	HTTPSigner() (HTTPSigner, error)

	// Public returns the public key associated with this SecureHW key.
	Public() (crypto.PublicKey, error)

	// Close releases any resources associated with this key.
	Close() error
}

type HTTPSigner interface {
	crypto.Signer
	ECCAlgorithm() ECCAlgorithm
}

type ECCAlgorithm int

const (
	ECCAlgorithmP256 ECCAlgorithm = iota + 1
	ECCAlgorithmP384
)

func New(metadataDir string, logger zerolog.Logger) (SecureHW, error) {
	logger = logger.With().Str("component", "securehw").Logger()
	return newSecureHW(metadataDir, logger)
}

// ErrKeyNotFound is returned when attempting to load a key that doesn't exist.
type ErrKeyNotFound struct {
	Message string
}

func (e ErrKeyNotFound) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "key not found in secure hardware"
}

// ErrSecureHWUnavailable is returned when the SecureHW hardware is not available.
type ErrSecureHWUnavailable struct {
	Message string
}

func (e ErrSecureHWUnavailable) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "secure hardware not available"
}
