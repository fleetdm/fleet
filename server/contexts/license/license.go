// Package license provides an API to create a context with the current license
// stored in it, and to retrieve the license from the context.
package license

import (
	"context"
)

// LicenseChecker is the interface for checking license properties.
// This interface is implemented by fleet.LicenseInfo
type LicenseChecker interface {
	IsPremium() bool
	IsAllowDisableTelemetry() bool
	// GetTier returns the license tier (e.g., "free", "premium", "trial").
	GetTier() string
	// GetOrganization returns the name of the licensed organization.
	GetOrganization() string
	// GetDeviceCount returns the number of licensed devices.
	GetDeviceCount() int
}

type key int

const licenseKey key = 0

// NewContext creates a new context.Context with the license.
func NewContext(ctx context.Context, lic LicenseChecker) context.Context {
	return context.WithValue(ctx, licenseKey, lic)
}

// FromContext returns the license from the context as a LicenseChecker interface.
// Use this when you only need to check license properties via the interface methods.
func FromContext(ctx context.Context) (LicenseChecker, bool) {
	v, ok := ctx.Value(licenseKey).(LicenseChecker)
	return v, ok
}

// IsPremium is a convenience function that returns true if the license stored
// in the context is for a premium tier, false otherwise (including if there
// is no license in the context).
func IsPremium(ctx context.Context) bool {
	if lic, ok := FromContext(ctx); ok {
		return lic.IsPremium()
	}
	return false
}

// IsAllowDisableTelemetry returns true if telemetry can be disabled based on
// the license in the context.
func IsAllowDisableTelemetry(ctx context.Context) bool {
	if lic, ok := FromContext(ctx); ok {
		return lic.IsAllowDisableTelemetry()
	}
	return false
}
