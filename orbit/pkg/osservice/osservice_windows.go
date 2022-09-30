//go:build windows
// +build windows

package osservice

import (
	"errors"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	shutdownFunctions   *[]func(err error)
	fleetDesktopPresent bool
}

func (m *windowsService) bestEffortShutdown() {
	serviceShutdown := errors.New("service is shutting down")

	// Calling interrupt functions to gracefully shutdown runners
	for _, interruptFn := range *m.shutdownFunctions {
		interruptFn(serviceShutdown)
	}

	// Now ensuring that no child process are left
	if m.fleetDesktopPresent {
		err := platform.KillProcessByName(constant.DesktopAppExecName)
		if err != nil {
			log.Error().Err(err).Msg("The desktop app couldn't be killed")
		}
	}

	err := platform.KillAllProcessByName(constant.OsquerydName)
	if err != nil {
		log.Error().Err(err).Msg("The child osqueryd processes cannot be killed")
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
			// Updating the service state to indicate stop
			changes <- svc.Status{State: svc.Stopped, Win32ExitCode: 0}

			// Best effort tear down
			// Runner group's interrupt functions will be called here
			m.bestEffortShutdown()

			// Dummy delay to allow the SCM to pick up the changes
			time.Sleep(500 * time.Millisecond)

			// Drastic teardown
			os.Exit(windows.NO_ERROR)

		default:
			return false, uint32(windows.ERROR_INVALID_SERVICE_CONTROL)
		}
	}

	return false, 0
}

// SetupServiceManagement implements the dispatcher and notification logic to
// interact with the Windows Service Control Manager (SCM)
func SetupServiceManagement(serviceName string, fleetDesktopPresent bool, shutdownFunctions *[]func(err error)) {
	if serviceName == "" {
		log.Error().Msg(" service name should not be empty")
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
			shutdownFunctions:   shutdownFunctions,
			fleetDesktopPresent: fleetDesktopPresent,
		}

		// Registering our service into the SCM
		err := svc.Run(serviceName, &srvData)
		if err != nil {
			log.Info().Err(err).Msg("SCM registration failed")
		}
	}
}
