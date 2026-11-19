package ai_agent_sessions

import (
	"os"
	"syscall"
	"time"
)

// birthTime is when the file was created, or the zero time if unknown.
func birthTime(_ string, fi os.FileInfo) time.Time {
	if d, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds()).UTC()
	}
	return time.Time{}
}
