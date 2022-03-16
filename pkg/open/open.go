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

type appOpts struct {
	env [][2]string
}

// AppOption are options to use when opening the application with App.
type AppOption func(*appOpts)

// AppWithEnv sets the environment for opening an application.
func AppWithEnv(name, value string) AppOption {
	return func(a *appOpts) {
		a.env = append(a.env, [2]string{name, value})
	}
}

// App opens an application at path with the default application.
// Returns the application process ID.
func App(path string, opts ...AppOption) error {
	var o appOpts
	for _, fn := range opts {
		fn(&o)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path %q: %w", path, err)
	}

	switch runtime.GOOS {
	case "darwin":
		if !info.IsDir() {
			return fmt.Errorf("path is not an .app directory: %s", path)
		}
		arg := []string{path}
		for _, nv := range o.env {
			arg = append(arg, "--env", fmt.Sprintf("%s=%s", nv[0], nv[1]))
		}
		if err := exec.Command("open", arg...).Run(); err != nil {
			return fmt.Errorf("open path: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("platform unsupported: %s", runtime.GOOS)
	}
}
