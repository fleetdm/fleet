package execuser

import (
	"fmt"
	"os"
	"os/exec"
)

// run uses macOS open command to start application as the current login user.
func run(path string, opts eopts) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path %q: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not an .app directory: %s", path)
	}
	var arg []string
	if opts.stderrPath != "" {
		arg = append(arg, "--stderr", opts.stderrPath)
	}
	for _, nv := range opts.env {
		arg = append(arg, "--env", fmt.Sprintf("%s=%s", nv[0], nv[1]))
	}
	arg = append(arg, path)
	cmd := exec.Command("open", arg...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open path %q: %w", path, err)
	}
	return nil
}
