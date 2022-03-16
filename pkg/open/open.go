package open

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Browser opens the default browser at the given url and returns.
func Browser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // xdg-open is available on most Linux-y systems
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open in browser: %w", err)
	}
	return nil
}

// App opens an application at path with the default application.
func App(path string, args ...string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path %q: %w", path, err)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		if !info.IsDir() {
			return fmt.Errorf("path is not an .app directory: %s", path)
		}
		arg := []string{path}
		if len(args) > 0 {
			arg = append(append(arg, "--args"), args...)
		}
		cmd = exec.Command("open", arg...)
	default:
		return fmt.Errorf("platform unsupported: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open path: %w", err)
	}
	return nil
}
