package ghapi

import (
	"bytes"
	"os/exec"

	"fleetdm/gm/pkg/logger"
)

// RunCommandAndReturnOutput runs a bash command, captures its output, and returns the output as a byte slice.
func RunCommandAndReturnOutput(command string) ([]byte, error) {
	logger.Debugf("Running COMMAND: %s", command)
	cmd := exec.Command("bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		logger.Errorf("Error running command: %s", out.String())
		return nil, err
	}
	return out.Bytes(), nil
}
