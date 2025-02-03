package open

import "os/exec"

func browser(url string) (string, error) {
	cmd := exec.Command("open", url)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
