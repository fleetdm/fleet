package execuser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// run uses macOS open command to start application as the current login user.
func run(path string, opts eopts) (lastLogs string, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat path %q: %w", path, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not an .app directory: %s", path)
	}
	var arg []string
	if opts.stderrPath != "" {
		arg = append(arg, "--stderr", opts.stderrPath)
	}

	// set environment variables
	for _, nv := range opts.env {
		arg = append(arg, "--env", fmt.Sprintf("%s=%s", nv[0], nv[1]))
	}

	// set the path to be executed
	arg = append(arg, path)

	// set the program arguments
	if len(opts.args) > 0 {
		arg = append(arg, "--args")
		for _, nv := range opts.args {
			arg = append(arg, nv[0], nv[1])
		}
	}

	cmd := exec.Command("/usr/bin/open", arg...)
	tw := &TransientWriter{}
	cmd.Stderr = io.MultiWriter(tw, os.Stderr)
	cmd.Stdout = io.MultiWriter(tw, os.Stdout)
	if err := cmd.Run(); err != nil {
		return tw.String(), fmt.Errorf("open path %q: %w", path, err)
	}
	return tw.String(), nil
}

func runWithOutput(path string, opts eopts) (output []byte, exitCode int, err error) {
	return nil, 0, errors.New("not implemented")
}
