//go:build !unix && !windows

package homes

import "os"

// statOwner has no implementation on platforms that are neither Unix (no
// syscall.Stat_t) nor Windows. All() then reports ownership as unknown rather
// than trusting the directory name.
func statOwner(_ string, _ os.FileInfo) (uid, username string, ok bool) { return "", "", false }

// platformHomes has nothing to return here either: only Windows keeps a record
// of homes by account (see owner_windows.go).
func platformHomes() []Home { return nil }
