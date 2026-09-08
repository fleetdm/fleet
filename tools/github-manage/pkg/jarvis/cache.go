package jarvis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CacheTTL is how long a cached fetch is considered fresh. Within this window
// `gm jarvis` opens instantly from disk instead of hitting GitHub; press r (or
// pass --no-cache) to force a live pull.
const CacheTTL = 4 * time.Hour

// cacheFile is the on-disk shape of a cached fetch.
type cacheFile struct {
	FetchedAt time.Time   `json:"fetched_at"`
	Result    FetchResult `json:"result"`
}

// DefaultCachePath returns ~/.config/gm/jarvis/cache.json.
func DefaultCachePath() string {
	return configPath("cache.json")
}

// LoadCache reads the cached fetch and when it was taken. ok is false if the
// cache is missing or unreadable.
func LoadCache(path string) (res FetchResult, fetchedAt time.Time, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FetchResult{}, time.Time{}, false
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return FetchResult{}, time.Time{}, false
	}
	return cf.Result, cf.FetchedAt, true
}

// SaveCache writes a fetch result to disk stamped with the current time.
// Best-effort: a write failure is returned but callers may ignore it.
func SaveCache(path string, res FetchResult) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cacheFile{FetchedAt: time.Now(), Result: res}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
