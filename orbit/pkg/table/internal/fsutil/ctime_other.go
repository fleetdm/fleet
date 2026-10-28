//go:build !darwin && !linux && !windows

package fsutil

import (
	"os"
	"time"
)

// changeTime is unknown on other platforms, so the hash cache there relies on
// size, modification time and file identity alone.
func changeTime(*os.File, os.FileInfo) time.Time { return time.Time{} }
