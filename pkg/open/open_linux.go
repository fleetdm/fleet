package open

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/rs/zerolog/log"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
)

var envVarsToCopy = []string{
	// XDG_CURRENT_DESKTOP hints which xdg-desktop-portal driver to use, which is how the user's
	// preferences are retrieved from their specific desktop environment (GNOME, MATE, KDE, etc).
	"XDG_CURRENT_DESKTOP",
	// XDG_RUNTIME_DIR is required for xdg-open to discover user settings via dconf.
	"XDG_RUNTIME_DIR",
	// Hermetic app packagers (e.g. Snap, Flatpak) tend to place `.desktop` files in alternate
	// locations which are published in this variable.
	"XDG_DATA_DIRS",
	// Copying the PATH ensures that snap-installed aliases in /snap/bin are accessible to
	// xdg-open.
	"PATH",
	// xdg-open requires XAUTHORITY set when running on a Wayland session (compatibility mode).
	// We get XAUTHORITY from the Xwayland process environment.
	//
	// We have to do this here instead of when executing fleet-desktop because the Xwayland process
	// may not be running yet when orbit is executing fleet-desktop.
	"XAUTHORITY",
}

func browser(url string) error {
	spy := &environmentSpy{}
	for _, varName := range envVarsToCopy {
		val, err := spy.GetEnvironmentVariableFromNamedProcess("/usr/bin/Xwayland", varName)
		if val != "" && err == nil {
			os.Setenv(varName, val)
			log.Info().Str(varName, val).Err(err).Msg("Xwayland process")
		}
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

// environmentSpy caches lookups of named processes to speed up retrieval of their environment variables.
type environmentSpy struct {
	processes   map[string][]*gopsutil_process.Process
	processesMu sync.Mutex
}

// GetProcessesByName returns a list of processes matching the provided executable path running as the
// current user.
func (es *environmentSpy) GetProcessesByName(exePath string) ([]*gopsutil_process.Process, error) {
	es.processesMu.Lock()
	defer es.processesMu.Unlock()

	if es.processes == nil {
		es.processes = make(map[string][]*gopsutil_process.Process)
	}

	if p, ok := es.processes[exePath]; ok {
		return p, nil
	}

	processes, err := platform.GetProcessesByName(path.Base(exePath))
	if err != nil {
		return nil, err
	}

	es.processes[exePath] = nil
	myUid := int32(os.Getuid())

	for _, p := range processes {
		exe, err := p.Exe()
		if err != nil || exe != exePath {
			continue
		}
		if uids, err := p.Uids(); err == nil {
			for _, uid := range uids {
				if uid == myUid {
					es.processes[exePath] = append(es.processes[exePath], p)
					break
				}
			}
		}
	}

	return es.processes[exePath], nil
}

// GetEnvironmentVariableFromNamedProcess retrieves the value of the requested environment variable
// from matching processes running as the current user.
func (es *environmentSpy) GetEnvironmentVariableFromNamedProcess(exePath, envvar string) (value string, err error) {
	processes, err := es.GetProcessesByName(exePath)
	if err != nil {
		return "", err
	}

	prefix := envvar + "="

	for _, process := range processes {
		envs, err := process.Environ()
		if err != nil {
			continue
		}

		for _, env := range envs {
			if strings.HasPrefix(env, prefix) {
				return strings.TrimPrefix(env, prefix), nil
			}
		}
	}

	return "", fmt.Errorf("cannot find %s envvar in any running %s process", envvar, exePath)
}
