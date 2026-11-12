package processes

import (
	"syscall"
	"unsafe"
)

// waitExit blocks until pid, an unreaped child of ours, has exited, without
// reaping it: until we Wait for it, its pid and so its process group id can't
// be reused. It reports false if the wait failed.
func waitExit(pid int) bool {
	const pPID = 1
	var siginfo [128]byte
	for {
		_, _, errno := syscall.Syscall6(syscall.SYS_WAITID, pPID, uintptr(pid), uintptr(unsafe.Pointer(&siginfo)), syscall.WEXITED|syscall.WNOWAIT, 0, 0)
		if errno != syscall.EINTR {
			return errno == 0
		}
	}
}
