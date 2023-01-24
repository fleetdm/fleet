package update

import (
	"fmt"
	"os/exec"
)

var _ = runCmdCollectErr // just to avoid unused errors on non-darwin platforms

func runCmdCollectErr(exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) > 0 {
		// just as a precaution, limit the length of the output
		if len(out) > 512 {
			out = out[:512]
		}
		err = fmt.Errorf("%w: %s", err, string(out))
	}
	return err
}
