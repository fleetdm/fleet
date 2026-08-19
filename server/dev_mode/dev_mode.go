package dev_mode

import (
	"os"
	"sync"
)

// testingTB is the subset of *testing.T that SetOverride needs in order to
// register cleanup. *testing.T satisfies it without an explicit conversion,
// so callers continue to pass `t` as-is. Defining this interface locally
// keeps the "testing" package out of dev_mode's production import graph.
type testingTB interface {
	Setenv(name, value string)
	Cleanup(f func())
}

// IsEnabled should be configured once at process startup (e.g., via flags) and then treated as read-only.
// It must not be written from concurrent goroutines; SetOverride only affects enabledViaOverride/env overrides.
var IsEnabled bool

// enabledViaOverride is set by SetOverride and protected by mu so that it is
// always observed consistently with envOverrides.
var enabledViaOverride bool

var mu sync.RWMutex

var envOverrides = map[string]string{}

type GetEnv func(name string) string

func Env(name string) string {
	mu.RLock()
	defer mu.RUnlock()

	if !IsEnabled && !enabledViaOverride {
		return ""
	}

	if override, ok := envOverrides[name]; ok {
		return override
	}

	return os.Getenv(name)
}

func SetOverride(name string, value string, cleanup ...testingTB) { // optional parameter to reset on test cleanup
	if len(cleanup) > 0 {
		cleanup[0].Setenv("FLEET_DEV_OVERRIDE_SET", "1") // triggers test deny-parallel check
		cleanup[0].Cleanup(func() {
			ClearOverride(name)
		})
	}

	mu.Lock()
	defer mu.Unlock()

	enabledViaOverride = true // if we're setting overrides, we're in a test environment so want to turn dev mode on
	envOverrides[name] = value
}

func ClearOverride(name string) {
	mu.Lock()
	defer mu.Unlock()

	delete(envOverrides, name)
}

func ClearAllOverrides() {
	mu.Lock()
	defer mu.Unlock()

	enabledViaOverride = false
	envOverrides = map[string]string{}
}
