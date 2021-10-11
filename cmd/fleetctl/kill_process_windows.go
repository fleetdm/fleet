//go:build windows
// +build windows

package main

func killPID(pid int) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(b.cmd.Process.Pid))
	return kill.Run()
}
