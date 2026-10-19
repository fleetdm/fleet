//go:build windows

// Package amcache implements osquery tables over the Windows Amcache hive
// (C:\Windows\AppCompat\Programs\Amcache.hve): executables the Compatibility Appraiser has
// inventoried (with SHA-1), installed applications and shortcuts, driver binaries and
// packages, and PnP device history.
//
// Based on https://github.com/karmine05/amcache.
package amcache

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

const (
	// cacheTTL bounds how stale a row can be and, on a host whose hive can't be read, how often
	// a raw volume read is attempted.
	cacheTTL = 5 * time.Minute

	// The context osquery passes to Generate has no deadline on Windows, so this bounds a load.
	// It starts once the load holds the cache, so time spent waiting can't make a load fail and
	// get cached as too slow.
	loadTimeout = 30 * time.Second

	// Longer constraint expressions can't be a hash, path or device ID and fall back to a scan.
	maxExprLen = 4096
)

var errLoadTooSlow = errors.New("amcache: loading the hive exceeded the deadline")

// snapshot is one parse plus equality indices over it. idx is keyed by "<table>.<column>",
// then by the lowercased decoded value, to record indexes.
type snapshot struct {
	inv *inventory
	idx map[string]map[string][]int32
}

// tables holds the state shared by all seven tables: osquery promotes every extension column
// to an index by default, and a JOIN on an indexed column calls Generate once per outer row,
// so without a shared cache a single query could read and parse the hive hundreds of times.
type tables struct {
	logger zerolog.Logger

	read    func(context.Context) (hiveResult, error)
	stat    func() (time.Time, int64, error)
	timeout time.Duration

	// A one-slot channel rather than a mutex so a waiter can give up when its context ends;
	// the holder may be doing a raw volume read and full parse, which takes seconds.
	sem chan struct{}

	// Guarded by sem.
	snap  *snapshot
	err   error
	at    time.Time
	mtime time.Time
	size  int64
	timer *time.Timer

	statWarned   sync.Once
	decodeWarned sync.Map
}

// TablePlugins returns the seven amcache table plugins, sharing one parsed hive.
func TablePlugins(logger zerolog.Logger) []*table.Plugin {
	return newTables(logger, readHive, statHive).plugins()
}

func newTables(logger zerolog.Logger, read func(context.Context) (hiveResult, error), stat func() (time.Time, int64, error)) *tables {
	return &tables{
		logger:  logger.With().Str("component", "amcache").Logger(),
		read:    read,
		stat:    stat,
		timeout: loadTimeout,
		sem:     make(chan struct{}, 1),
	}
}

func (t *tables) plugins() []*table.Plugin {
	out := make([]*table.Plugin, 0, len(specs))
	for _, s := range specs {
		out = append(out, table.NewPlugin(s.table, s.columnDefs(), t.generator(s)))
	}
	return out
}

// get returns the shared snapshot, parsing the hive if the cached one is missing, expired or
// stale. A nil snapshot with a nil error means the host has no hive.
func (t *tables) get(ctx context.Context) (*snapshot, error) {
	select {
	case t.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-t.sem }()

	if mtime, size, err := t.stat(); err != nil {
		t.statWarned.Do(func() {
			t.logger.Debug().Err(err).Msg("amcache: hive freshness check unavailable; relying on the cache TTL")
		})
	} else if mtime != t.mtime || size != t.size {
		t.drop()
		t.mtime, t.size = mtime, size
	}

	if (t.snap == nil && t.err == nil) || time.Since(t.at) > cacheTTL {
		if err := t.load(ctx); err != nil {
			return nil, err
		}
	}
	if errors.Is(t.err, errHiveNotFound) {
		return nil, nil
	}
	if t.err != nil {
		return nil, t.err
	}
	return t.snap, nil
}

// load replaces what is cached. Failures are cached too, including a load that exceeds its
// own deadline, so a host whose hive can't be read doesn't attempt a raw volume read per query.
// The exception is the caller going away, which says nothing about the hive.
func (t *tables) load(ctx context.Context) error {
	lctx, cancel := context.WithTimeoutCause(ctx, t.timeout, errLoadTooSlow)
	defer cancel()

	snap, err := t.parse(lctx)
	switch {
	case err == nil:
	case errors.Is(err, context.DeadlineExceeded) && errors.Is(context.Cause(lctx), errLoadTooSlow):
		err = errLoadTooSlow
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	}

	t.snap, t.err, t.at = snap, err, time.Now()
	if t.timer == nil {
		t.timer = time.AfterFunc(cacheTTL, t.expire)
	} else {
		t.timer.Reset(cacheTTL)
	}
	return nil
}

// parse recovers from panics itself so that load caches them like any other failure.
func (t *tables) parse(ctx context.Context) (snap *snapshot, err error) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error().Msgf("amcache: recovered from %T while parsing the hive", r)
			snap, err = nil, fmt.Errorf("amcache: parsing the hive panicked: %v", r)
		}
	}()

	res, err := t.read(ctx)
	if err != nil {
		return nil, err
	}
	dirty, err := hiveDirty(res.data)
	if err != nil {
		return nil, err
	}
	if dirty {
		return nil, errHiveDirty
	}
	inv, err := parseInventory(ctx, t.logger, res.data)
	if err != nil {
		return nil, err
	}
	t.logger.Debug().Bool("raw_volume_read", res.raw).Msg("amcache: hive parsed")
	return &snapshot{inv: inv, idx: buildIndex(inv)}, nil
}

// expire drops an idle snapshot so a parsed hive doesn't stay in orbit's memory between
// investigations.
func (t *tables) expire() {
	t.sem <- struct{}{}
	defer func() { <-t.sem }()
	// A reload after the timer fired has already re-armed it.
	if time.Since(t.at) < cacheTTL {
		return
	}
	t.drop()
}

func (t *tables) drop() { t.snap, t.err = nil, nil }

// buildIndex indexes the decoded values, which is what users type (the 40-character sha1, the
// backslashed device ID), rather than the raw hive values.
func buildIndex(inv *inventory) map[string]map[string][]int32 {
	idx := make(map[string]map[string][]int32)
	for _, s := range specs {
		recs := inv.keys[s.key]
		if len(recs) == 0 {
			continue
		}
		for _, c := range s.cols {
			if !c.push {
				continue
			}
			byVal := make(map[string][]int32)
			for i, rec := range recs {
				v, ok := cell(c, rec)
				// Empty values aren't indexed; an equality on "" falls back to a scan.
				if !ok || v == "" {
					continue
				}
				k := strings.ToLower(v)
				byVal[k] = append(byVal[k], int32(i)) // #nosec G115 -- bounded by maxSubkeys
			}
			idx[s.table+"."+c.name] = byVal
		}
	}
	return idx
}

// selected returns the record indexes a query's equality constraints narrow to, or nil for all
// records. SQLite re-filters the rows we return, so a superset is always correct and a subset
// never is; every uncertain branch returns more rows.
func selected(qc table.QueryContext, idx map[string]map[string][]int32, tbl string) []int32 {
	var out []int32
	narrowed := false

	for name, cl := range qc.Constraints {
		byVal, ok := idx[tbl+"."+name]
		if !ok {
			continue
		}

		var hits []int32
		usable := false
		for _, cst := range cl.Constraints {
			if cst.Operator != table.OperatorEquals || cst.Expression == "" || len(cst.Expression) > maxExprLen {
				continue
			}
			usable = true
			hits = append(hits, byVal[strings.ToLower(cst.Expression)]...)
		}
		if !usable {
			continue
		}

		// Union within a column (an IN list may arrive as several equalities), intersection
		// across columns.
		slices.Sort(hits)
		hits = slices.Compact(hits)
		if !narrowed {
			out, narrowed = hits, true
			continue
		}
		out = intersect(out, hits)
	}

	if !narrowed {
		return nil
	}
	if out == nil {
		return []int32{}
	}
	return out
}

func intersect(a, b []int32) []int32 {
	out := make([]int32, 0, min(len(a), len(b)))
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i] < b[j]:
			i++
		case a[i] > b[j]:
			j++
		default:
			out = append(out, a[i])
			i++
			j++
		}
	}
	return out
}

func (t *tables) generator(s spec) table.GenerateFunc {
	return func(ctx context.Context, qc table.QueryContext) (rows []map[string]string, err error) {
		// The hive is attacker-influenced input and a panic here would take down orbit.
		defer func() {
			if r := recover(); r != nil {
				t.logger.Error().Str("table", s.table).Msgf("amcache: recovered from %T", r)
				rows, err = nil, fmt.Errorf("%s: recovered from panic: %v", s.table, r)
			}
		}()

		// Bounds the wait for another table's load plus our own load.
		ctx, cancel := context.WithTimeout(ctx, 2*t.timeout)
		defer cancel()

		snap, err := t.get(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.table, err)
		}
		if snap == nil {
			return nil, nil
		}
		recs := snap.inv.keys[s.key]
		if len(recs) == 0 {
			return nil, nil
		}
		return t.rowsFor(ctx, s, recs, selected(qc, snap.idx, s.table))
	}
}
