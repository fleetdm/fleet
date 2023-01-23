//go:build darwin
// +build darwin

package common

import (
	"fmt"
	"os"
	"syscall"
)

// GetActiveUserGroup gets the uid and gid of the current (or more accurately, most recently logged
// in) *console* user. In most scenarios this should be the currently logged in user on the system.
// Note that getting the current user of the Orbit process is typically going to return root and we
// need the underlying user.
func GetConsoleUidGid() (uid uint32, gid uint32, err error) {
	info, err := os.Stat("/dev/console")
	if err != nil {
		return 0, 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected type %T", info.Sys())
	}
	return stat.Uid, stat.Gid, nil
}
