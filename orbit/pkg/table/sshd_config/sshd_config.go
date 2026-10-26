// Package sshd_config exposes the OpenSSH server's effective configuration, as
// printed by `sshd -G`, as the `sshd_config` osquery table.
package sshd_config

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

// cacheTTL is how long a parsed dump is reused. osquery rescans an extension
// table once per row of the outer table in a join, so without the cache
// `users JOIN sshd_config` runs sshd once per user.
const cacheTTL = 5 * time.Second

type cachedConfig struct {
	entries []configEntry
	expires time.Time
}

var (
	cacheMu  sync.Mutex
	cache    = make(map[string]cachedConfig)
	inFlight singleflight.Group
)

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("keyword"),
		table.TextColumn("value"),
		table.TextColumn("connection_spec"),
	}
}

// Generate returns one row per setting in sshd's effective configuration, or no
// rows when sshd isn't installed. It fails the query when sshd rejects its
// configuration, so a broken sshd_config isn't mistaken for a host without one.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	sshd, ok := findSSHD()
	if !ok {
		log.Debug().Msg("sshd not found; reporting no sshd_config rows")
		return []map[string]string{}, nil
	}

	// Without a connection_spec constraint, report the global configuration:
	// sshd evaluates no Match blocks unless it is given -C.
	specs := connectionSpecs(queryContext)
	if len(specs) == 0 {
		specs = []string{""}
	}

	rows := []map[string]string{}
	for _, spec := range specs {
		entries, err := loadConfig(ctx, sshd, spec)
		if err != nil {
			return nil, err
		}
		rows = slices.Grow(rows, len(entries))
		for _, entry := range entries {
			rows = append(rows, map[string]string{
				"keyword":         entry.Keyword,
				"value":           entry.Value,
				"connection_spec": spec,
			})
		}
	}
	return rows, nil
}

// loadConfig returns the parsed configuration for spec, reusing a dump up to
// cacheTTL old and sharing one sshd run among concurrent callers. Errors aren't
// cached, so a fixed configuration shows up on the next query. A canceled
// caller stops waiting, but a run already started finishes for the others.
func loadConfig(ctx context.Context, sshd sshdBinary, spec string) ([]configEntry, error) {
	key := cacheKey(sshd, spec)
	if entries, ok := cachedEntries(key); ok {
		return entries, nil
	}

	// A canceled query starts no more sshd runs.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ch := inFlight.DoChan(key, func() (any, error) {
		// The run is shared, so one caller's cancellation mustn't fail the
		// others; sshdTimeout still bounds it.
		return refreshConfig(context.WithoutCancel(ctx), sshd, spec)
	})

	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		entries, _ := res.Val.([]configEntry)
		return entries, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// refreshConfig runs sshd and caches the parsed dump. It checks the cache first
// because a caller that missed it can reach singleflight after another caller's
// run has filled it and finished, which would otherwise start a second run.
func refreshConfig(ctx context.Context, sshd sshdBinary, spec string) ([]configEntry, error) {
	key := cacheKey(sshd, spec)
	if entries, ok := cachedEntries(key); ok {
		return entries, nil
	}

	output, err := dumpConfig(ctx, sshd, spec)
	if err != nil {
		return nil, err
	}
	entries := parseConfig(output)

	now := time.Now()
	cacheMu.Lock()
	defer cacheMu.Unlock()
	for k, c := range cache {
		if now.After(c.expires) {
			delete(cache, k)
		}
	}
	cache[key] = cachedConfig{entries: entries, expires: now.Add(cacheTTL)}
	return entries, nil
}

func cacheKey(sshd sshdBinary, spec string) string {
	return sshd.id + "\x00" + spec
}

// cachedEntries returns the dump cached under key if it hasn't expired.
func cachedEntries(key string) ([]configEntry, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	c, ok := cache[key]
	if !ok || !time.Now().Before(c.expires) {
		return nil, false
	}
	return c.entries, true
}

// connectionSpecs returns the distinct values of every equality constraint on
// connection_spec, which covers both `=` and `IN (...)`.
func connectionSpecs(queryContext table.QueryContext) []string {
	list, ok := queryContext.Constraints["connection_spec"]
	if !ok {
		return nil
	}

	var specs []string
	seen := make(map[string]struct{})
	for _, c := range list.Constraints {
		if c.Operator != table.OperatorEquals {
			continue
		}
		if _, dup := seen[c.Expression]; dup {
			continue
		}
		seen[c.Expression] = struct{}{}
		specs = append(specs, c.Expression)
	}
	return specs
}
