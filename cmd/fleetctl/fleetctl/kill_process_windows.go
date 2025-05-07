//go:build windows
// +build windows

package fleetctl

import (
	"os/exec"
	"strconv"
)

func killPID(pid int) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)) //nolint:gosec
	return kill.Run()
}
