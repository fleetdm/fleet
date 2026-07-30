//go:build !unix && !windows

package homes

import "os"

// statOwnerUID has no implementation on platforms that are neither Unix (no
// syscall.Stat_t) nor Windows. owner() then reports ownership as unknown rather
// than trusting the directory name.
func statOwnerUID(_ string, _ os.FileInfo) (string, bool) { return "", false }
