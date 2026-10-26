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
// cached, so a fixed configuration shows up on the next query.
func loadConfig(ctx context.Context, sshd sshdBinary, spec string) ([]configEntry, error) {
	key := sshd.id + "\x00" + spec

	cacheMu.Lock()
	cached, ok := cache[key]
	cacheMu.Unlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.entries, nil
	}

	v, err, _ := inFlight.Do(key, func() (any, error) {
		// Callers waiting on this run shouldn't fail because the first one was
		// canceled; sshdTimeout still bounds it.
		output, err := dumpConfig(context.WithoutCancel(ctx), sshd, spec)
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
	})
	if err != nil {
		return nil, err
	}
	entries, _ := v.([]configEntry)
	return entries, nil
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
