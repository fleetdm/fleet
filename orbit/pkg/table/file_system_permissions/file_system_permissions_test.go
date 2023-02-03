//go:build darwin
// +build darwin

package file_system_permissions

import (
	"testing"
)

func TestXXX(t *testing.T) {
	_, _, err := GetConsoleUidGid()
	if err != nil {
		t.Fatalf(`Err expected to be nil. got %s`, err)
	}
}
