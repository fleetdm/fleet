package open

import (
	"os/exec"
	"regexp"
	"syscall"
)

var unescapedAmpsRegex = regexp.MustCompile(`([^\\^])&`)

func browser(url string) error {
	// Replace all instances of & that are not already escaped with ^.
	// This is necessary because cmd.exe treats & as a command separator.
	url = unescapedAmpsRegex.ReplaceAllString(url, "${1}^&")
	cmd := exec.Command("cmd", "/c", "start", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// HideWindow avoids a brief cmd console from opening
		// before the browser opens the URL.
		HideWindow: true,
	}
	return cmd.Run()
}
