//go:build windows
// +build windows

package main

import (
	"os/exec"
	"stdconv"
)

func killPID(pid int) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	return kill.Run()
}
