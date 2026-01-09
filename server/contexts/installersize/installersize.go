// Package installersize provides context handling for the maximum software installer size.
package installersize

import (
	"context"

	"github.com/docker/go-units"
)

type key struct{}

// DefaultMaxInstallerSizeStr is the default maximum size allowed for software installers
// as a human-readable string (10 GiB). Use this for configuration defaults.
const DefaultMaxInstallerSizeStr = "10GiB"

// DefaultMaxInstallerSize is the default maximum size allowed for software installers (10 GiB).
const DefaultMaxInstallerSize int64 = 10 * units.GiB

// NewContext returns a new context with the max installer size value.
func NewContext(ctx context.Context, maxSize int64) context.Context {
	return context.WithValue(ctx, key{}, maxSize)
}

// FromContext returns the max installer size from the context if present.
// If not present, returns DefaultMaxInstallerSize.
func FromContext(ctx context.Context) int64 {
	v, ok := ctx.Value(key{}).(int64)
	if !ok {
		return DefaultMaxInstallerSize
	}
	return v
}
