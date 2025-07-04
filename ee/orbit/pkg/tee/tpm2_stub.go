//go:build !linux
// +build !linux

package tee

import (
	"github.com/rs/zerolog"
)

// TPM2Option is a functional option for configuring a TPM2 TEE
type TPM2Option func(*interface{})

// WithLogger sets the logger for the TPM2 TEE (stub)
func WithLogger(logger zerolog.Logger) TPM2Option {
	return func(t *interface{}) {
		// No-op for stub
	}
}

// WithPublicBlobPath sets the path where the TPM public blob will be saved (stub)
func WithPublicBlobPath(path string) TPM2Option {
	return func(t *interface{}) {
		// No-op for stub
	}
}

// WithPrivateBlobPath sets the path where the TPM private blob will be saved (stub)
func WithPrivateBlobPath(path string) TPM2Option {
	return func(t *interface{}) {
		// No-op for stub
	}
}

// NewTPM2 creates a new TEE instance using TPM 2.0.
// This is a stub implementation for non-Linux platforms.
func NewTPM2(opts ...TPM2Option) (TEE, error) {
	return nil, ErrTEEUnavailable{Message: "TPM 2.0 is only supported on Linux"}
}
