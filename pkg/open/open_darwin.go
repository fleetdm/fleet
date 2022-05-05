package open

import "os/exec"

func browser(url string) error {
	return exec.Command("open", url).Run()
}
