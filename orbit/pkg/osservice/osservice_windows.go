//go:build windows
// +build windows

package osservice

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	shutdownFunctions   *[]func(err error)
	fleetDesktopPresent bool
	appDoneCh           chan struct{}
}

func (m *windowsService) gracefulShutdown() {
	// Calling interrupt functions to gracefully shutdown runners
	for _, shutdownFn := range *m.shutdownFunctions {
		shutdownFn(errors.New("service graceful shutdown"))
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

			// Best effort graceful tear down
			// Runner group's interrupt functions will be called here
			m.gracefulShutdown()

			// Wait for runners group to finish
			<-m.appDoneCh

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
func SetupServiceManagement(serviceName string, fleetDesktopPresent bool, shutdownFunctions *[]func(err error), doneCh chan struct{}) {
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
			appDoneCh:           doneCh,
		}

		// Registering our service into the SCM
		err := svc.Run(serviceName, &srvData)
		if err != nil {
			log.Info().Err(err).Msg("SCM registration failed")
		}
	}
}
