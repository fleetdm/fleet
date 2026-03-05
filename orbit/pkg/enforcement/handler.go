package enforcement

import (
	"context"
	"errors"
)

// ErrNotSupported is returned by stub handlers on non-Windows platforms.
var ErrNotSupported = errors.New("enforcement: not supported on this platform")

// DiffResult represents the compliance check result for a single setting.
type DiffResult struct {
	SettingName  string
	Category     string
	PolicyName   string
	CISRef       string
	DesiredValue string
	CurrentValue string
	Compliant    bool
}

// ApplyResult represents the outcome of enforcing a single setting.
type ApplyResult struct {
	SettingName string
	Category    string
	Success     bool
	Error       string
}

// Handler is the interface that enforcement handlers must implement.
type Handler interface {
	// Name returns the handler category (e.g., "registry", "secpol").
	Name() string

	// Diff compares desired settings against actual system state.
	Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error)

	// Apply enforces the desired settings on the system.
	Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error)
}
