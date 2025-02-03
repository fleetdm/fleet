package open

import (
	"os/exec"
	"syscall"
)

func browser(url string) (string, error) {
	cmd := exec.Command("cmd", "/c", "start", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// HideWindow avoids a brief cmd console from opening
		// before the browser opens the URL.
		HideWindow: true,
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
