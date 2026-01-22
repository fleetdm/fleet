package dev_mode

import "os"

var IsEnabled bool

func Env(name string) string {
	if !IsEnabled {
		return ""
	}

	return os.Getenv(name)
}
