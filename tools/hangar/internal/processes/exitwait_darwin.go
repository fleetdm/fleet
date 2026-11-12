package processes

import "syscall"

// waitExit blocks until pid, an unreaped child of ours, has exited, without
// reaping it: until we Wait for it, its pid and so its process group id can't
// be reused. It reports false if the wait couldn't be set up.
func waitExit(pid int) bool {
	kq, err := syscall.Kqueue()
	if err != nil {
		return false
	}
	defer syscall.Close(kq)
	change := []syscall.Kevent_t{{
		Ident: uint64(pid), Filter: syscall.EVFILT_PROC, Flags: syscall.EV_ADD | syscall.EV_ONESHOT, Fflags: syscall.NOTE_EXIT,
	}}
	for {
		_, err = syscall.Kevent(kq, change, nil, nil)
		if err != syscall.EINTR {
			break
		}
	}
	if err == syscall.ESRCH {
		return true // already a zombie
	}
	if err != nil {
		return false
	}
	events := make([]syscall.Kevent_t, 1)
	for {
		n, err := syscall.Kevent(kq, nil, events, nil)
		if err != syscall.EINTR {
			return err == nil && n > 0
		}
	}
}
