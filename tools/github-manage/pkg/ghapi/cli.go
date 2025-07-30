package ghapi

import (
	"bytes"
	"os/exec"
)

// RunCommandAndParseJSON runs a bash command, captures its output, and parses the output as JSON.
func RunCommandAndParseJSON(command string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
