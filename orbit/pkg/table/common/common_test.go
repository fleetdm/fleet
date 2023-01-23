//go:build darwin
// +build darwin

package common

import (
	"testing"
)

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestGetConsoleUidGid(t *testing.T) {
	_, _, err := GetConsoleUidGid()
	if err != nil {
		t.Fatalf(`Err expected to be nil. got %s`, err)
	}
}
