//go:build windows
// +build windows

package service

import (
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	rootDir string
}

func (m *windowsService) bestEffortShutdown(serviceRootDir string) {
	err := platform.KillProcessByName(constant.DesktopAppExecName)
	if err != nil {
		log.Info().Err(err).Msg("The desktop app couldn't be killed")
	}

	if serviceRootDir != "" {
		err = platform.KillFromPIDFile(serviceRootDir, constant.OsqueryPidfile, constant.OsquerydName)
		if err != nil {
			log.Info().Err(err).Msg("The osquery daemon process couldn't be killed")
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
			m.bestEffortShutdown(m.rootDir)

			// Updating the service state to indicate that stop finished
			changes <- svc.Status{State: svc.Stopped}

			// Dummy delay to allow the SCM to pick up the changes
			time.Sleep(500 * time.Millisecond)

			// Drastic teardown
			// This will generate an internal signal that will be caught by
			// the app SignalHandler() runner, which will end up forcing
			// the interrupt method to run on all runners
			os.Exit(0)

		default:
			return false, 1052 // 1052: ERROR_INVALID_SERVICE_CONTROL
		}
	}

	return false, 0
}

// This function implements the dispatcher and notification logic to
// interact with the Windows Service Control Manager (SCM)
func SetupServiceManagement(serviceName string, serviceRootDir string) {
	if (serviceName != "") && (serviceRootDir != "") {
		// Ensuring that we are only calling the SCM if running as a service
		isWindowsService, err := svc.IsWindowsService()
		if err != nil {
			log.Info().Err(err).Msg("couldn't determine if running as a service")
		}

		if isWindowsService {
			srvData := windowsService{rootDir: serviceRootDir}
			// Registering our service into the SCM
			err := svc.Run(serviceName, &srvData)
			if err != nil {
				log.Info().Err(err).Msg("SCM registration failed")
			}
		}
	}
}
