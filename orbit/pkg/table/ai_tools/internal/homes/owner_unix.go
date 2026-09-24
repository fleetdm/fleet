//go:build unix

package homes

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// statOwner returns the owning uid of the file described by fi, read from the
// underlying stat, and the account name that uid resolves to. The uid is the
// OS's own record of ownership, so it cannot be forged by naming a directory
// after another account. Naming it is best-effort: a uid with no passwd entry
// still identifies the owner and is reported with an empty username. The dir
// path is unused on Unix (the FileInfo already carries the owner).
func statOwner(_ string, fi os.FileInfo) (uid, username string, ok bool) {
	st, sok := fi.Sys().(*syscall.Stat_t)
	if !sok {
		return "", "", false
	}
	uid = strconv.FormatUint(uint64(st.Uid), 10)
	if u, err := user.LookupId(uid); err == nil {
		username = u.Username
	}
	return uid, username, true
}

// platformHomes has nothing to return on Unix. passwd records a user's home
// directory, but All() already reaches those accounts by listing the
// directories that hold them, and the owner uid it reads from a directory there
// is the OS's own record of a *user* — no separate source is needed to
// establish who a home belongs to. Only Windows has one (see owner_windows.go).
func platformHomes() []Home { return nil }
