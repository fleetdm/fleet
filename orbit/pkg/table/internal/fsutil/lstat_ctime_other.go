//go:build !unix

package fsutil

import (
	"os"
	"time"
)

// LstatChangeTime stats path without following a final link and returns its
// info with its change time. Only an open handle carries the change time
// here, so the file is opened as OpenRegular does.
func LstatChangeTime(path string) (os.FileInfo, time.Time, error) {
	f, err := OpenRegular(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	return fi, changeTime(f, fi), nil
}
