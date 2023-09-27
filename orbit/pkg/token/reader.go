package token

import (
	"os"
	"sync"
	"time"
)

// Reader is used to read the token value from a file
type Reader struct {
	Path string // a path to a file containing a token

	mu     sync.Mutex // ensures atomic reads/writes; protects the following fields
	cached string     // the value of the token, a.k.a. the contents of the file
	mtime  time.Time  // the mtime of the file when the token was last read
}

// Read returns the token value from the file only if the file is
// expired or the cached value is empty
func (r *Reader) Read() (string, error) {
	changed, err := r.HasChanged()
	if err != nil {
		return "", err
	}

	if changed || r.GetCached() == "" {
		if err := r.readFile(); err != nil {
			return "", err
		}
	}

	return r.GetCached(), nil
}

// HasChanged checks if the in-memory `value` has changed by comparing
// the chached `r.mtime` value with the `mtime` of the file at
// `r.path`
func (r *Reader) HasChanged() (bool, error) {
	info, err := os.Stat(r.Path)
	if err != nil {
		return false, err
	}
	mtime := info.ModTime()
	r.mu.Lock()
	defer r.mu.Unlock()
	return !mtime.Equal(r.mtime), nil
}

// HasExpired checks if 1 hour has passed since the last recorded `mtime` of
// the token file. It returns true, 0 if the token has expired, or false and
// the duration until it does expire.
func (r *Reader) HasExpired() (bool, time.Duration) {
	const expirationDuration = 1 * time.Hour

	r.mu.Lock()
	defer r.mu.Unlock()
	since := time.Since(r.mtime)
	if since > expirationDuration {
		return true, 0
	}
	return false, expirationDuration - since
}

func (r *Reader) GetMtime() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.mtime
}

// GetCached returns the cached token value
func (r *Reader) GetCached() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cached
}

func (r *Reader) readFile() error {
	f, err := os.ReadFile(r.Path)
	if err != nil {
		return err
	}
	info, err := os.Stat(r.Path)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.mtime = info.ModTime()
	r.cached = string(f)
	r.mu.Unlock()
	return nil
}
