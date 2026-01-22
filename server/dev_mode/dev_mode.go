package dev_mode

import "os"

var IsEnabled bool

var EnvOverrides = map[string]string{}

type GetEnv func(name string) string

func Env(name string) string {
	if !IsEnabled {
		return ""
	}
	if override, ok := EnvOverrides[name]; ok {
		return override
	}

	return os.Getenv(name)
}
