//go:build darwin
// +build darwin

package common

import (
	"testing"
)

func TestGetConsoleUidGid(t *testing.T) {
	_, _, err := GetConsoleUidGid()
	if err != nil {
		t.Fatalf(`Err expected to be nil. got %s`, err)
	}
}
