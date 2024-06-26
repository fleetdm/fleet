package service

import "sync"

var (
	buildTagsMu sync.Mutex
	buildTags   = make(map[string]struct{})
)

func addBuildTag(tag string) { //nolint:unused
	buildTagsMu.Lock()
	defer buildTagsMu.Unlock()

	buildTags[tag] = struct{}{}
}

func hasBuildTag(tag string) bool {
	buildTagsMu.Lock()
	defer buildTagsMu.Unlock()

	_, ok := buildTags[tag]
	return ok
}
