package ctxdb

import (
	"context"
)

type key int

const (
	requirePrimaryKey    key = 0
	bypassCachedMysqlKey key = 1
)

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

// BypassCachedMysql returns a new context that indicates to the mysql cache
// layer if	the cache should be bypassed. This is required when reading data
// with the intention of writing it back with changes, to avoid reading stale
// data from the cache.
func BypassCachedMysql(ctx context.Context, bypass bool) context.Context {
	return context.WithValue(ctx, bypassCachedMysqlKey, bypass)
}

// IsCachedMysqlBypassed returns true if the context indicates that the mysql
// cache must be bypassed, false otherwise.
func IsCachedMysqlBypassed(ctx context.Context) bool {
	v, _ := ctx.Value(bypassCachedMysqlKey).(bool)
	return v
}
