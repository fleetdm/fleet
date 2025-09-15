package open

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"

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
	// xdg-open is available on most Linux-y systems
	cmd := exec.Command("xdg-open", url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// grab environment variables necessary to determine the user's preferred browser from any
	// running Xwayland or xdg-desktop-portal process.
	cmd.Env = newEnvironmentWithVariablesFromNamedProcesses(
		[]string{"xdg-desktop-portal", "Xwayland"},
		envVarsToCopy...)
	log.Debug().Strs("command", []string{"xdg-open", url}).Strs("env", cmd.Env).Msg("env for xdg-open cmd")
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

// getProcessesByName returns a list of processes matching the provided executable path running as the
// current user.
func getProcessesByName(exePath string) ([]*gopsutil_process.Process, error) {
	processes, err := platform.GetProcessesByName(path.Base(exePath))
	if err != nil {
		return nil, err
	}

	var out []*gopsutil_process.Process
	myUid := int32(os.Getuid()) //nolint:gosec // dismiss G115

	for _, p := range processes {
		if path.IsAbs(exePath) {
			// for absolute paths, skip if no match
			exe, err := p.Exe()
			if err != nil || exe != exePath {
				if err != nil {
					log.Debug().Err(err).Msg("p.Exe")
				}
				continue
			}
		}
		if uids, err := p.Uids(); err == nil {
			for _, uid := range uids {
				if uid == myUid {
					out = append(out, p)
					break
				}
			}
		} else {
			log.Debug().Err(err).Msg("p.Uids")
		}
	}

	return out, nil
}

func localEnv() map[string]string {
	out := make(map[string]string)

	for _, envvar := range os.Environ() {
		if name, value, ok := strings.Cut(envvar, "="); ok {
			out[name] = value
		}
	}

	return out
}

// getEnvironmentVariablesFromNamedProcess retrieves the value of the requested environment variables
// from matching processes running as the current user.
func getEnvironmentVariablesFromNamedProcess(exePaths []string, envvars ...string) (vars map[string]string, err error) {
	var processes []*gopsutil_process.Process
	for _, exePath := range exePaths {
		if p, err := getProcessesByName(exePath); err == nil {
			processes = append(processes, p...)
		} else {
			log.Debug().Err(err).Msg("getProcessesByName")
		}
	}

	log.Debug().Strs("exePaths", exePaths).Msg("spying envvars from process")
	vars = make(map[string]string)

	for _, process := range processes {
		envs, err := process.Environ()
		if err != nil {
			log.Debug().Err(err).Msg("process.Environ")
			continue
		}

		for _, env := range envs {
			if name, value, ok := strings.Cut(env, "="); ok && slices.Contains(envvars, name) {
				log.Debug().Str(name, value).Msg("found envvar")
				vars[name] = value
			}
		}
	}

	return
}

// newEnvironmentWithVariablesFromNamedProcess copies the current environment, then looks for a
// process running as the current user and merges the requested environment variables into the
// local environment. The returned slice is suitable for use with (exec.Cmd).Env.
func newEnvironmentWithVariablesFromNamedProcesses(exePaths []string, envvars ...string) []string {
	localEnv := localEnv()

	if extraVars, err := getEnvironmentVariablesFromNamedProcess(exePaths, envvars...); err == nil {
		for k, v := range extraVars {
			localEnv[k] = v
		}

		// log envvars that we weren't able to find in the local or examined processes
		for _, v := range envvars {
			if _, ok := localEnv[v]; !ok {
				log.Debug().Str("envvar", v).Msg("unable to find envvar")
			}
		}
	} else {
		log.Debug().Err(err).Msg("getEnvironmentVariablesFromNamedProcess")
	}

	for _, k := range []string{"_", "LD_LIBRARY_PATH"} {
		delete(localEnv, k)
	}

	var out []string
	for k, v := range localEnv {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
