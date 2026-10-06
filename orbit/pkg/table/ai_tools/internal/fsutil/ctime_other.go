//go:build !darwin && !linux

package fsutil

import (
	"os"
	"time"
)

// changeTime is unavailable through os.FileInfo on Windows, so the hash cache
// there relies on size, modification time and file identity alone.
func changeTime(os.FileInfo) time.Time { return time.Time{} }
