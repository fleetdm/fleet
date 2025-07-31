package ghapi

import (
	"bytes"
	"fmt"
	"os/exec"
)

// RunCommandAndReturnOutput runs a bash command, captures its output, and returns the output as a byte slice.
func RunCommandAndReturnOutput(command string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running command: %s\n", out.String())
		return nil, err
	}
	return out.Bytes(), nil
}
