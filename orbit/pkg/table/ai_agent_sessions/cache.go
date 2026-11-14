package ai_agent_sessions

import (
	"os"
	"sync"
	"sync/atomic"
)

// stamp identifies one version of a file; fi is nil when the file was absent.
type stamp struct {
	fi os.FileInfo
}

func stampOf(path string) stamp {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return stamp{}
	}
	return stamp{fi: fi}
}

// same reports whether two stamps describe the same file content: size,
// modification time and file identity all match. Agents only append or
// replace their files, so a same-size rewrite that restores the modification
// time isn't worth the change time it would take to catch.
func (a stamp) same(b stamp) bool {
	if a.fi == nil || b.fi == nil {
		return a.fi == nil && b.fi == nil
	}
	return a.fi.Size() == b.fi.Size() && a.fi.ModTime().Equal(b.fi.ModTime()) && os.SameFile(a.fi, b.fi)
}

type cacheEntry struct {
	stamps  []stamp
	session Session
	ok      bool
}

// sessionCache keeps parsed sessions for the orbit process's lifetime, so a
// scheduled query only re-reads transcripts that changed.
type sessionCache struct {
	mu sync.Mutex
	m  map[string]cacheEntry
}

var cache = &sessionCache{m: map[string]cacheEntry{}}

// cacheKey names a session by its files and the account it was read for, so
// a home reattributed or renamed is parsed again under its current user.
func cacheKey(src *source) string {
	h := src.sc.home
	return src.agent.name + "\x00" + src.files[0] + "\x00" + src.id + "\x00" + h.UID + "\x00" + h.Username
}

// session returns the source's Session, parsing it unless an unchanged copy
// is cached, and marks its key in seen.
func (c *sessionCache) session(src *source, seen map[string]struct{}) (Session, bool) {
	key := cacheKey(src)
	seen[key] = struct{}{}

	files := append(append([]string{}, src.files...), src.deps...)
	stamps := make([]stamp, len(files))
	for i, p := range files {
		stamps[i] = stampOf(p)
	}

	c.mu.Lock()
	e, hit := c.m[key]
	c.mu.Unlock()
	if hit && len(e.stamps) == len(stamps) {
		for i := range stamps {
			if !e.stamps[i].same(stamps[i]) {
				hit = false
				break
			}
		}
		if hit {
			return e.session, e.ok
		}
	}

	if src.sc.reads.exhausted {
		return Session{}, false
	}
	s, ok := parseSession(src)
	if src.sc.reads.exhausted {
		// The parse may have been cut short; read it again next time.
		return Session{}, false
	}
	c.mu.Lock()
	c.m[key] = cacheEntry{stamps: stamps, session: s, ok: ok}
	c.mu.Unlock()
	return s, ok
}

// evict drops entries for transcripts a complete scan no longer found.
func (c *sessionCache) evict(seen map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if _, ok := seen[k]; !ok {
			delete(c.m, k)
		}
	}
}

// derivedCache keeps a value computed from one file (a session id read from
// its first record, an index of a log) while the file is unchanged, so
// discovery doesn't re-read files the session cache already covers.
type derivedCache[T any] struct {
	mu sync.Mutex
	m  map[string]derivedEntry[T]
}

type derivedEntry[T any] struct {
	stamp stamp
	v     T
	gen   uint64
}

// derivedGen is bumped by each complete scan; entries not touched since the
// previous one are dropped.
var (
	derivedGen atomic.Uint64
	// derivedCaches is filled by package initialisation only.
	derivedCaches []interface {
		sweep(before uint64)
		clear()
	}
)

func newDerivedCache[T any]() *derivedCache[T] {
	c := &derivedCache[T]{m: map[string]derivedEntry[T]{}}
	derivedCaches = append(derivedCaches, c)
	return c
}

// get returns the value for path, computing it unless the file is unchanged
// since it was last computed. A value computed once the call's read budget
// ran out is a fallback, and isn't kept.
func (c *derivedCache[T]) get(sc *scope, path string, compute func() T) T {
	st := stampOf(path)
	gen := derivedGen.Load()
	c.mu.Lock()
	e, hit := c.m[path]
	if hit && e.stamp.same(st) {
		e.gen = gen
		c.m[path] = e
		c.mu.Unlock()
		return e.v
	}
	c.mu.Unlock()
	v := compute()
	if sc.reads.exhausted {
		return v
	}
	c.mu.Lock()
	c.m[path] = derivedEntry[T]{stamp: st, v: v, gen: gen}
	c.mu.Unlock()
	return v
}

func (c *derivedCache[T]) sweep(before uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.m {
		if e.gen < before {
			delete(c.m, k)
		}
	}
}

func (c *derivedCache[T]) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.m)
}

// sweepDerived drops derived values not touched since before the scan that
// started at generation start; a scan still running keeps what it touched.
func sweepDerived(start uint64) {
	for _, c := range derivedCaches {
		c.sweep(start)
	}
	derivedGen.Add(1)
}

// resetState empties the caches and the size budget.
func resetState() {
	cache.mu.Lock()
	cache.m = map[string]cacheEntry{}
	cache.mu.Unlock()
	for _, c := range derivedCaches {
		c.clear()
	}
	messageBudget.reset()
}
