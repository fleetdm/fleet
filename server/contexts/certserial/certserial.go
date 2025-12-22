// Package certserial provides a context key for storing client certificate serial numbers
// extracted from HTTP headers during mTLS authentication for device endpoints.
package certserial

import "context"

type key int

const certSerialKey key = 0

// NewContext returns a new context.Context with the provided certificate serial number.
func NewContext(ctx context.Context, serial uint64) context.Context {
	return context.WithValue(ctx, certSerialKey, serial)
}

// FromContext returns the certificate serial number from the context, if present.
// The second return value indicates whether a serial number was found.
func FromContext(ctx context.Context) (uint64, bool) {
	serial, ok := ctx.Value(certSerialKey).(uint64)
	return serial, ok
}
