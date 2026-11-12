//nolint:gosec // G103: unsafe required for Windows API calls.
package fsutil

import (
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileBasicInfo is the Win32 FILE_BASIC_INFO structure, which x/sys/windows
// doesn't define. Times are FILETIMEs: 100-ns intervals since 1601.
type fileBasicInfo struct {
	CreationTime   int64
	LastAccessTime int64
	LastWriteTime  int64
	ChangeTime     int64
	FileAttributes uint32
	_              uint32 // padding to the structure's 8-byte alignment
}

// changeTime returns the change time of the open file f. os.FileInfo doesn't
// carry it on Windows, so it's read from the handle's FILE_BASIC_INFO.
func changeTime(f *os.File, _ os.FileInfo) time.Time {
	var info fileBasicInfo
	err := windows.GetFileInformationByHandleEx(windows.Handle(f.Fd()), windows.FileBasicInfo,
		(*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		return time.Time{}
	}
	ft := windows.Filetime{LowDateTime: uint32(info.ChangeTime), HighDateTime: uint32(info.ChangeTime >> 32)}
	return time.Unix(0, ft.Nanoseconds())
}
