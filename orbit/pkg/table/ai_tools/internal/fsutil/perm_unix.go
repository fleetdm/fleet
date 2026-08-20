//go:build !windows

package fsutil

import "os"

// statPerm reads path's POSIX mode bits. Lstat, not Stat: the scanner runs as
// root over user-writable homes, and a symlink's target permissions say nothing
// about who can tamper with the path we actually reported.
func statPerm(path string) Perm {
	fi, err := os.Lstat(path)
	if err != nil {
		return Perm{}
	}
	m := fi.Mode().Perm()
	return Perm{
		WorldReadable: m&0o044 != 0,
		WorldWritable: m&0o022 != 0,
		Known:         true,
	}
}
