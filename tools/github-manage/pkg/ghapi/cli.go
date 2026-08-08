package ghapi

import (
	"bytes"
	"os/exec"
	"strings"
	"time"

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

// RunGH runs the gh CLI with explicit arguments (no shell), which is safe for
// user-supplied input like comment bodies. Combined stdout+stderr is returned.
func RunGH(args ...string) ([]byte, error) {
	logger.Debugf("Running gh %s", strings.Join(args, " "))
	cmd := exec.Command("gh", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		logger.Errorf("gh error: %s", out.String())
		return out.Bytes(), err
	}
	return out.Bytes(), nil
}

// isTransient reports whether command output looks like a retryable GitHub error
// (transient 5xx / gateway / timeout), as opposed to a permanent failure.
func isTransient(out []byte) bool {
	s := string(out)
	for _, marker := range []string{"HTTP 502", "HTTP 503", "HTTP 504", "Bad Gateway", "Service Unavailable", "Gateway Timeout", "timeout", "EOF"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// RunCommandWithRetry runs a command, retrying with exponential backoff when the
// failure looks transient (GitHub 5xx/gateway/timeout). Permanent errors return
// immediately. The output of the last attempt is returned alongside the error.
func RunCommandWithRetry(command string, attempts int) ([]byte, error) {
	if attempts < 1 {
		attempts = 1
	}
	var out []byte
	var err error
	for i := 0; i < attempts; i++ {
		out, err = RunCommandAndReturnOutput(command)
		if err == nil {
			return out, nil
		}
		if i == attempts-1 || !isTransient(out) {
			return out, err
		}
		backoff := time.Duration(1<<i) * time.Second
		logger.Debugf("transient error, retrying in %s (attempt %d/%d)", backoff, i+1, attempts)
		time.Sleep(backoff)
	}
	return out, err
}
