package dev_mode

import (
	"os"
	"testing"
)

var IsEnabled bool

var envOverrides = map[string]string{}

type GetEnv func(name string) string

func Env(name string) string {
	if !IsEnabled {
		return ""
	}
	if override, ok := envOverrides[name]; ok {
		return override
	}

	return os.Getenv(name)
}

func SetOverride(name string, value string, cleanup ...*testing.T) { // optional parameter to reset on test cleanup
	IsEnabled = true // if we're setting overrides, we're in a test environment so want to turn dev mode on
	envOverrides[name] = value

	if len(cleanup) > 0 {
		cleanup[0].Cleanup(func() {
			ClearOverride(name)
		})
	}
}

func ClearOverride(name string) {
	delete(envOverrides, name)
}

func ClearAllOverrides() {
	envOverrides = map[string]string{}
}
