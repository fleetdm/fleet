//go:build windows
// +build windows

package osservice

import (
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	interruptCh chan struct{}
	appDoneCh   chan struct{}
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
			log.Info().Msg("Service Interrogate Requested")
			changes <- req.CurrentStatus

		case svc.Stop, svc.Shutdown:
			log.Info().Msg("Service Stop Requested")

			// Updating the service state to indicate stop
			changes <- svc.Status{State: svc.Stopped, Win32ExitCode: 0}

			// Best effort graceful tear down
			// Runner group's will be interrupted after signaling them
			close(m.interruptCh)

			// Wait for runners group to finish
			<-m.appDoneCh

			// Drastic teardown
			os.Exit(windows.NO_ERROR)

		default:
			log.Info().Msg("Unknown service request")
			return false, uint32(windows.ERROR_INVALID_SERVICE_CONTROL)
		}
	}

	return false, 0
}

// SetupServiceManagement implements the dispatcher and notification logic to
// interact with the Windows Service Control Manager (SCM)
func SetupServiceManagement(serviceName string, interruptCh chan struct{}, doneCh chan struct{}) {
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
			interruptCh: interruptCh,
			appDoneCh:   doneCh,
		}

		// Registering our service into the SCM
		err := svc.Run(serviceName, &srvData)
		if err != nil {
			log.Info().Err(err).Msg("SCM registration failed")
		}
	}
}
