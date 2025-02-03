package open

import "os/exec"

func browser(url string) (string, error) {
	// xdg-open is available on most Linux-y systems
	cmd := exec.Command("xdg-open", url)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
