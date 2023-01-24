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

func TestGetRootUidGid(t *testing.T) {
	uid, gid, err := GetRootUidGid()
	if err != nil {
		t.Fatalf(`Err expected to be nil. got %s.  uid: %d, gid: %d`, err, uid, gid)
	}
}
