package publicip

import (
	"context"
)

type key int

const ipKey key = 0

// NewContext returns a new context carrying the current remote ip.
func NewContext(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ipKey, ip)
}

// FromContext extracts the remote ip from context if present.
func FromContext(ctx context.Context) string {
	ip, ok := ctx.Value(ipKey).(string)
	if !ok {
		return ""
	}
	return ip
}
