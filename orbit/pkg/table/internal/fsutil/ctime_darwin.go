package fsutil

import (
	"os"
	"syscall"
	"time"
)

// changeTime returns the change time of the open file f, whose stat is fi.
func changeTime(_ *os.File, fi os.FileInfo) time.Time {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Ctimespec.Unix())
	}
	return time.Time{}
}
