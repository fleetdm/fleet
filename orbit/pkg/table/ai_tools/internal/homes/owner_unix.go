//go:build unix

package homes

import (
	"os"
	"strconv"
	"syscall"
)

// statOwnerUID returns the owning uid of the file described by fi, read from the
// underlying stat. This is the OS's own record of ownership, so it cannot be
// forged by naming a directory after another account. The dir path is unused on
// Unix (the FileInfo already carries the owner).
func statOwnerUID(_ string, fi os.FileInfo) (string, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return "", false
	}
	return strconv.FormatUint(uint64(st.Uid), 10), true
}
