//go:build windows

package fsutil

// Windows has no O_NOFOLLOW; symlink creation there requires privilege, and the
// post-open os.SameFile identity check in OpenRegular still guards against a
// swapped target. The Lstat pre-check also rejects reparse points up front.
const openNoFollow = 0
