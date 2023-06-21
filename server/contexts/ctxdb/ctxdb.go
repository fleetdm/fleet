package ctxdb

import (
	"context"
)

type key int

const requirePrimaryKey key = 0

// RequirePrimary returns a new context that indicates to the database layer if
// the primary instance must always be used instead of the replica, even for
// reads (to be able to read recent writes).
func RequirePrimary(ctx context.Context, requirePrimary bool) context.Context {
	return context.WithValue(ctx, requirePrimaryKey, requirePrimary)
}

// IsPrimaryRequired returns true if the context indicates that the primary
// instance is required for reads, false otherwise.
func IsPrimaryRequired(ctx context.Context) bool {
	v, _ := ctx.Value(requirePrimaryKey).(bool)
	return v
}
