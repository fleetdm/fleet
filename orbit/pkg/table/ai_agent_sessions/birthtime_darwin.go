package ai_agent_sessions

import (
	"os"
	"syscall"
	"time"
)

// birthTime is when the file was created, or the zero time if unknown.
func birthTime(_ string, fi os.FileInfo) time.Time {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Birthtimespec.Unix()).UTC()
	}
	return time.Time{}
}
