//go:build !windows

package fsutil

import "syscall"

// openNoFollow makes OpenRegular's open fail (ELOOP) if the final path component
// is a symlink, so the root scanner cannot be raced into following one.
const openNoFollow = syscall.O_NOFOLLOW
