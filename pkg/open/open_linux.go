package open

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/rs/zerolog/log"
)

func browser(url string) error {
	// xdg-open requires XAUTHORITY set when running on a Wayland session (compatibility mode).
	// We get XAUTHORITY from the Xwayland process environment.
	//
	// We have to do this here instead of when executing fleet-desktop because the Xwayland process
	// may not be running yet when orbit is executing fleet-desktop.
	xAuthority, err := getXWaylandAuthority()
	log.Info().Str("XAUTHORITY", xAuthority).Err(err).Msg("Xwayland process")
	if err == nil {
		os.Setenv("XAUTHORITY", xAuthority)
	}
	// xdg-open is available on most Linux-y systems
	cmd := exec.Command("xdg-open", url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Must be asynchronous (Start, not Run) because xdg-open will continue running
	// and block this goroutine if it was the process that opened the browser.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xdg-open failed to start: %w", err)
	}
	go func() {
		// We must call wait to avoid defunct processes.
		cmd.Wait() //nolint:errcheck
	}()
	return nil
}

// getXWaylandAuthority retrieves the X authority file path from
// the running XWayland process environment.
func getXWaylandAuthority() (xAuthorityPath string, err error) {
	xWaylandProcess, err := platform.GetProcessesByName("Xwayland")
	if err != nil {
		return "", fmt.Errorf("get process by name: %w", err)
	}
	executablePath, err := xWaylandProcess[0].Exe()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	if executablePath != "/usr/bin/Xwayland" {
		return "", fmt.Errorf("invalid Xwayland path: %q", executablePath)
	}
	envs, err := xWaylandProcess[0].Environ()
	if err != nil {
		return "", fmt.Errorf("get environment: %w", err)
	}
	for _, env := range envs {
		if strings.HasPrefix(env, "XAUTHORITY=") {
			return strings.TrimPrefix(env, "XAUTHORITY="), nil
		}
	}
	return "", errors.New("XAUTHORITY not found")
}
