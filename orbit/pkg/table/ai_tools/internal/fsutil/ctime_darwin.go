package fsutil

import (
	"os"
	"syscall"
	"time"
)

func changeTime(fi os.FileInfo) time.Time {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Ctimespec.Unix())
	}
	return time.Time{}
}
