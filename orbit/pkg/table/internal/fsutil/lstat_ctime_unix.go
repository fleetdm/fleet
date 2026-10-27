//go:build unix

package fsutil

import (
	"os"
	"time"
)

// LstatChangeTime stats path without following a final link and returns its
// info with its change time, which the stat already carries here.
func LstatChangeTime(path string) (os.FileInfo, time.Time, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	return fi, changeTime(nil, fi), nil
}
