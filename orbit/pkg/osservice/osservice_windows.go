//go:build windows
// +build windows

package osservice

import (
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	rootDir             string
	fleetDesktopPresent bool
}

func (m *windowsService) bestEffortShutdown() {
	if m.fleetDesktopPresent {
		err := platform.KillProcessByName(constant.DesktopAppExecName)
		if err != nil {
			log.Error().Err(err).Msg("The desktop app couldn't be killed")
		}
	}

	if m.rootDir != "" {
		err := platform.KillFromPIDFile(m.rootDir, constant.OsqueryPidfile, constant.OsquerydName)
		if err != nil {
			log.Error().Err(err).Msg("The osquery daemon process couldn't be killed")
		}
	}
}

func (m *windowsService) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// Accepted service operations
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	// Expected service status update during initialization
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// The listening loop below will keep listening for new SCM requests
	for req := range requests {
		switch req.Cmd {

		case svc.Interrogate:
			changes <- req.CurrentStatus

		case svc.Stop, svc.Shutdown:

			// Service shutdown was requested
			// Updating the service state to indicate stop op started
			changes <- svc.Status{State: svc.StopPending}

			// Best effort tear down
			m.bestEffortShutdown()

			// Updating the service state to indicate that stop finished
			changes <- svc.Status{State: svc.Stopped}

			// Dummy delay to allow the SCM to pick up the changes
			time.Sleep(500 * time.Millisecond)

			// Drastic teardown
			// This will generate an internal signal that will be caught by
			// the app SignalHandler() runner, which will end up forcing
			// the interrupt method to run on all runners
			os.Exit(windows.NO_ERROR)

		default:
			return false, uint32(windows.ERROR_INVALID_SERVICE_CONTROL)
		}
	}

	return false, 0
}

// SetupServiceManagement implements the dispatcher and notification logic to
// interact with the Windows Service Control Manager (SCM)
func SetupServiceManagement(serviceName string, serviceRootDir string, fleetDesktopPresent bool) {
	if serviceName == "" {
		log.Error().Msg(" service name should not be empty")
		return
	}

	if serviceRootDir == "" {
		log.Error().Msg(" service root dir should not be empty")
		return
	}

	// Ensuring that we are only calling the SCM if running as a service
	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		log.Error().Err(err).Msg("couldn't determine if running as a service")
		return
	}

	if isWindowsService {
		srvData := windowsService{
			rootDir:             serviceRootDir,
			fleetDesktopPresent: fleetDesktopPresent,
		}

		// Registering our service into the SCM
		err := svc.Run(serviceName, &srvData)
		if err != nil {
			log.Info().Err(err).Msg("SCM registration failed")
		}
	}
}
