// Log the panic under windows to the log file
//
// Code from minix, via
//
// https://play.golang.org/p/kLtct7lSUg

//go:build windows
// +build windows

package paniclog

import (
	"errors"
	"os"
	"syscall"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
	procGetStdHandle = kernel32.MustFindProc("GetStdHandle")
)

func dupFD(fd uintptr) (syscall.Handle, error) {
	// Cribbed from https://github.com/golang/go/blob/go1.8/src/syscall/exec_windows.go#L303.
	p, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, err
	}
	var h syscall.Handle
	return h, syscall.DuplicateHandle(p, syscall.Handle(fd), p, &h, 0, true, syscall.DUPLICATE_SAME_ACCESS)
}

func getStdHandle(stdHandle int32) (syscall.Handle, error) {
	r0, _, e1 := syscall.Syscall(procGetStdHandle.Addr(), 2, uintptr(stdHandle), 0, 0)
	rh0 := syscall.Handle(r0)
	if rh0 == syscall.InvalidHandle {
		if e1 != 0 {
			return syscall.InvalidHandle, error(e1)
		}
		return syscall.InvalidHandle, syscall.EINVAL
	}
	return syscall.Handle(r0), nil
}

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}

func redirectStderr(f *os.File) (UndoFunction, error) {
	stderrFd, err := getStdHandle(syscall.STD_ERROR_HANDLE)
	if err != nil {
		return nil, errors.New("Failed to redirect stderr to file: " + err.Error())
	}

	// duplicate the handle to match unix behavior
	fHandle, err := dupFD(f.Fd())
	if err != nil {
		return nil, errors.New("Failed to duplicate file: " + err.Error())
	}

	err = setStdHandle(syscall.STD_ERROR_HANDLE, fHandle)
	if err != nil {
		return nil, errors.New("Failed to redirect stderr to file: " + err.Error())
	}

	undo := func() error {
		err := setStdHandle(syscall.STD_ERROR_HANDLE, stderrFd)
		if err != nil {
			return errors.New("Failed to redirect stderr to file: " + err.Error())
		}
		err = syscall.CloseHandle(fHandle)
		if err != nil {
			return errors.New("Failed to close STD_ERROR handle: " + err.Error())
		}
		return nil
	}

	return undo, nil
}
