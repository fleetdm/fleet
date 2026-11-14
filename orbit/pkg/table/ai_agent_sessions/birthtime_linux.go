package ai_agent_sessions

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// birthTime is when the file was created, from statx, or the zero time where
// the kernel or filesystem doesn't record it.
func birthTime(path string, _ os.FileInfo) time.Time {
	var st unix.Statx_t
	if unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &st) != nil || st.Mask&unix.STATX_BTIME == 0 {
		return time.Time{}
	}
	return time.Unix(st.Btime.Sec, int64(st.Btime.Nsec)).UTC()
}
