// Package securehw contains implementations of hardware-based cryptographic interfaces.
package securehw

import (
	"crypto"

	"github.com/rs/zerolog"
)

// TEE (Trusted Execution Environment) provides an interface for hardware-based
// cryptographic operations, such as those performed by a TPM (Trusted Platform Module).
type TEE interface {
	// CreateKey creates a new key in the TEE and returns a handle to it.
	// The implementation will automatically choose the best available key type,
	// preferring ECC P-384 if supported, otherwise falling back to ECC P-256.
	// Returns a Key interface that can be used for cryptographic operations.
	CreateKey() (Key, error)

	// LoadKey loads a previously created key from the public and private blobs saved to files.
	// The blobs are read from the file paths configured when creating the TEE instance.
	// The parent key is the hardcoded Storage Root Key (SRK) handle.
	LoadKey() (Key, error)

	// Close releases any resources held by the TEE.
	Close() error
}

// Key represents a key stored in a TEE that can perform cryptographic operations.
type Key interface {
	// Signer returns a crypto.Signer that uses this key for signing operations.
	// The returned Signer is safe for concurrent use.
	Signer() (crypto.Signer, error)

	// Public returns the public key associated with this TEE key.
	Public() (crypto.PublicKey, error)

	// Close releases any resources associated with this key.
	Close() error
}

func New(metadataDir string, logger zerolog.Logger) (TEE, error) {
	logger = logger.With().Str("component", "securehw").Logger()
	return newTEE(metadataDir, logger)
}

// ErrKeyNotFound is returned when attempting to load a key that doesn't exist.
type ErrKeyNotFound struct {
	Message string
}

func (e ErrKeyNotFound) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "key not found in TPM/TEE"
}

// ErrTEEUnavailable is returned when the TEE hardware is not available.
type ErrTEEUnavailable struct {
	Message string
}

func (e ErrTEEUnavailable) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "secure hardware not available"
}
