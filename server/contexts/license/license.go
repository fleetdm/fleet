// Package license provides an API to create a context with the current license
// stored in it, and to retrieve the license from the context.
package license

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const licenseKey key = 0

// NewContext creates a new context.Context with the license.
func NewContext(ctx context.Context, lic *fleet.LicenseInfo) context.Context {
	return context.WithValue(ctx, licenseKey, lic)
}

// FromContext returns the license from the context and true, or nil and false
// if there is no license.
func FromContext(ctx context.Context) (*fleet.LicenseInfo, bool) {
	v, ok := ctx.Value(licenseKey).(*fleet.LicenseInfo)
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

func IsAllowDisableTelemetry(ctx context.Context) bool {
	if lic, ok := FromContext(ctx); ok {
		return lic.IsAllowDisableTelemetry()
	}
	return false
}
