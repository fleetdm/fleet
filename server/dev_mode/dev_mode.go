package dev_mode

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// Do not write this variable from concurrent goroutines; use SetOverride instead.
var IsEnabled bool

// enabledViaOverride is set atomically by SetOverride so that background goroutines
// calling Env() do not race with test code that calls SetOverride().
var enabledViaOverride atomic.Bool

var mu sync.RWMutex

var envOverrides = map[string]string{}

type GetEnv func(name string) string

func Env(name string) string {
	if !IsEnabled && !enabledViaOverride.Load() {
		return ""
	}

	mu.RLock()
	defer mu.RUnlock()

	if override, ok := envOverrides[name]; ok {
		return override
	}

	return os.Getenv(name)
}

func SetOverride(name string, value string, cleanup ...*testing.T) { // optional parameter to reset on test cleanup
	if len(cleanup) > 0 {
		cleanup[0].Setenv("FLEET_DEV_OVERRIDE_SET", "1") // triggers test deny-parallel check
		cleanup[0].Cleanup(func() {
			ClearOverride(name)
		})
	}

	enabledViaOverride.Store(true) // if we're setting overrides, we're in a test environment so want to turn dev mode on
	mu.Lock()
	defer mu.Unlock()

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

	envOverrides = map[string]string{}
}
