// Package installersize provides context handling for the maximum software installer size.
package installersize

import (
	"context"

	"github.com/docker/go-units"
)

// Human formats a byte size into a human-readable string.
// It evaluates both SI units (KB, MB, GB) and binary units (KiB, MiB, GiB)
// and returns whichever representation is shorter.
func Human(bytes int64) string {
	si := units.HumanSize(float64(bytes))
	binary := units.BytesSize(float64(bytes))

	if len(binary) < len(si) {
		return binary
	}
	return si
}

type key struct{}

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
