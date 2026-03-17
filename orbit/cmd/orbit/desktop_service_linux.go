//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
	"github.com/rs/zerolog/log"
)

const (
	// desktopEnvFilePath is where orbit writes the env file that the fleet-desktop
	// systemd user service reads via EnvironmentFile=.
	desktopEnvFilePath = "/opt/orbit/desktop.env"
)

// desktopUserServiceManager manages Fleet Desktop as a systemd user service on Linux.
// Instead of launching fleet-desktop via sudo/runuser (the old desktopRunner approach),
// this writes an environment file that the systemd user service reads, and manages
// the service lifecycle via systemctl.
type desktopUserServiceManager struct {
	desktopPath                 string
	fleetURL                    string
	trw                         *token.ReadWriter
	fleetRootCA                 string
	insecure                    bool
	fleetClientCrt              []byte
	fleetClientKey              []byte
	fleetAlternativeBrowserHost string
	updateRoot                  string

	interruptCh   chan struct{}
	executeDoneCh chan struct{}
}

func newDesktopUserServiceManager(
	desktopPath, fleetURL, fleetRootCA string,
	insecure bool,
	trw *token.ReadWriter,
	fleetClientCrt, fleetClientKey []byte,
	fleetAlternativeBrowserHost string,
	updateRoot string,
) *desktopUserServiceManager {
	return &desktopUserServiceManager{
		desktopPath:                 desktopPath,
		fleetURL:                    fleetURL,
		trw:                         trw,
		fleetRootCA:                 fleetRootCA,
		insecure:                    insecure,
		fleetClientCrt:              fleetClientCrt,
		fleetClientKey:              fleetClientKey,
		fleetAlternativeBrowserHost: fleetAlternativeBrowserHost,
		updateRoot:                  updateRoot,
		interruptCh:                 make(chan struct{}),
		executeDoneCh:               make(chan struct{}),
	}
}

// Execute writes the desktop environment file and ensures the systemd user service
// is running. It then periodically refreshes the env file (e.g. if the token changes)
// and restarts the service for logged-in users when the desktop binary is updated.
func (m *desktopUserServiceManager) Execute() error {
	defer close(m.executeDoneCh)

	log.Info().Str("path", m.desktopPath).Msg("managing fleet-desktop as systemd user service")

	// Write initial env file.
	if err := m.writeEnvFile(); err != nil {
		return fmt.Errorf("write desktop env file: %w", err)
	}

	// Ensure the token file is readable by all users so the user service can read it.
	if err := m.ensureTokenReadable(); err != nil {
		log.Error().Err(err).Msg("failed to make token file readable")
	}

	// Restart (or start) the service for any currently logged-in users.
	// We use restart rather than start so that after an orbit restart (e.g. due to a
	// TUF update of the desktop binary), the user service picks up the new binary.
	m.restartServiceForLoggedInUsers()

	// Periodically refresh the env file and ensure token permissions.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.interruptCh:
			return nil
		case <-ticker.C:
			// Refresh env file (token may have changed).
			if err := m.writeEnvFile(); err != nil {
				log.Error().Err(err).Msg("refresh desktop env file")
			}

			// Ensure token permissions stay correct.
			if err := m.ensureTokenReadable(); err != nil {
				log.Error().Err(err).Msg("failed to make token file readable")
			}
		}
	}
}

// Interrupt signals the Execute loop to stop and disables the service for logged-in users.
func (m *desktopUserServiceManager) Interrupt(err error) {
	close(m.interruptCh)
	<-m.executeDoneCh

	m.stopServiceForLoggedInUsers()
}

// writeEnvFile writes the environment file that the fleet-desktop systemd user service reads.
func (m *desktopUserServiceManager) writeEnvFile() error {
	var lines []string

	addLine := func(key, value string) {
		if value != "" {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	}

	addLine("FLEET_DESKTOP_FLEET_URL", m.fleetURL)
	addLine("FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH", m.trw.Path)
	addLine("FLEET_DESKTOP_FLEET_TLS_CLIENT_CERTIFICATE", string(m.fleetClientCrt))
	addLine("FLEET_DESKTOP_FLEET_TLS_CLIENT_KEY", string(m.fleetClientKey))
	addLine("FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST", m.fleetAlternativeBrowserHost)
	addLine("FLEET_DESKTOP_TUF_UPDATE_ROOT", m.updateRoot)
	addLine("FLEET_DESKTOP_FLEET_ROOT_CA", m.fleetRootCA)
	if m.insecure {
		addLine("FLEET_DESKTOP_INSECURE", "1")
	}

	content := strings.Join(lines, "\n") + "\n"

	// Write atomically: write to temp file then rename.
	tmpPath := desktopEnvFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write temp env file: %w", err)
	}
	if err := os.Rename(tmpPath, desktopEnvFilePath); err != nil {
		return fmt.Errorf("rename env file: %w", err)
	}

	return nil
}

// ensureTokenReadable makes the device token file readable by all users
// so the fleet-desktop user service can read it.
func (m *desktopUserServiceManager) ensureTokenReadable() error {
	tokenPath := m.trw.Path
	info, err := os.Stat(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Token not yet written, that's okay.
		}
		return fmt.Errorf("stat token file: %w", err)
	}

	// Make readable by all if not already.
	if info.Mode().Perm()&0044 != 0044 {
		if err := os.Chmod(tokenPath, 0644); err != nil {
			return fmt.Errorf("chmod token file: %w", err)
		}
		log.Info().Str("path", tokenPath).Msg("made token file readable for desktop user service")
	}

	return nil
}

// getLoggedInUserIDs returns the UIDs of all currently logged-in non-system users.
func getLoggedInUserIDs() []string {
	out, err := exec.Command("loginctl", "list-users", "--no-legend", "--no-pager").Output()
	if err != nil {
		log.Debug().Err(err).Msg("loginctl list-users failed")
		return nil
	}

	var uids []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		username := fields[1]
		// Skip system/display manager users.
		if username == "gdm" || username == "root" || username == "gdm-greeter" {
			continue
		}
		uids = append(uids, fields[0])
	}
	return uids
}

// runUserSystemctl runs systemctl --user commands for the specified user UID.
// It uses runuser to switch to the user context and sets XDG_RUNTIME_DIR.
func runUserSystemctl(uid string, args ...string) error {
	// Build: sudo runuser -u <username> -- env XDG_RUNTIME_DIR=/run/user/<uid> systemctl --user <args...>
	// First we need to get the username from the UID.
	usernameOut, err := exec.Command("id", "-nu", uid).Output()
	if err != nil {
		return fmt.Errorf("get username for uid %s: %w", uid, err)
	}
	username := strings.TrimSpace(string(usernameOut))

	envVar := fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%s", uid)
	cmdArgs := []string{"-u", username, "--", "env", envVar, "systemctl", "--user"}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("runuser", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("runuser systemctl for user %s (uid %s): %w (output: %s)", username, uid, err, string(output))
	}
	return nil
}

// stopServiceForLoggedInUsers stops the fleet-desktop user service for all logged-in users.
func (m *desktopUserServiceManager) stopServiceForLoggedInUsers() {
	uids := getLoggedInUserIDs()
	for _, uid := range uids {
		if err := runUserSystemctl(uid, "stop", constant.DesktopAppExecName+".service"); err != nil {
			log.Debug().Err(err).Str("uid", uid).Msg("stop fleet-desktop user service")
		} else {
			log.Info().Str("uid", uid).Msg("stopped fleet-desktop user service")
		}
	}
}

// restartServiceForLoggedInUsers restarts the fleet-desktop user service for all logged-in users.
func (m *desktopUserServiceManager) restartServiceForLoggedInUsers() {
	uids := getLoggedInUserIDs()
	for _, uid := range uids {
		if err := runUserSystemctl(uid, "daemon-reload"); err != nil {
			log.Debug().Err(err).Str("uid", uid).Msg("daemon-reload for user")
		}
		if err := runUserSystemctl(uid, "restart", constant.DesktopAppExecName+".service"); err != nil {
			log.Error().Err(err).Str("uid", uid).Msg("restart fleet-desktop user service")
		} else {
			log.Info().Str("uid", uid).Msg("restarted fleet-desktop user service")
		}
	}
}
