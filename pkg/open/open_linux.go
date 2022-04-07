package open

import "os/exec"

func browser(url string) error {
	// xdg-open is available on most Linux-y systems
	return exec.Command("xdg-open", url).Run()
}
