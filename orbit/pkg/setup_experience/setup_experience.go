package setupexperience

import (
	"context"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/swiftdialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// Client is the minimal interface needed to communicate with the Fleet server.
type Client interface {
	GetSetupExperienceStatus() (*fleet.SetupExperienceStatusPayload, error)
}

// SetupExperiencer is the type that manages the Fleet setup experience flow during macOS Setup
// Assistant. It uses swiftDialog as a UI for showing the status of software installations and
// script execution that are configured to run before the user has full access to the device.
type SetupExperiencer struct {
	OrbitClient Client
	openChan    chan struct{}
	closeChan   chan struct{}
	rootDirPath string
	sd          *swiftdialog.SwiftDialog
	counter     int
	started     bool
}

func NewSetupExperiencer(client Client, rootDirPath string) *SetupExperiencer {
	return &SetupExperiencer{
		OrbitClient: client,
		openChan:    make(chan struct{}),
		closeChan:   make(chan struct{}, 1), // TODO(JVE): does this have to be buffered?
		rootDirPath: rootDirPath,
	}
}

func (s *SetupExperiencer) Run(oc *fleet.OrbitConfig) error {
	// We should only launch swiftDialog if we get the notification from Fleet.
	_, binaryPath, _ := update.LocalTargetPaths(
		s.rootDirPath,
		"swiftDialog",
		update.SwiftDialogMacOSTarget,
	)

	if _, err := os.Stat(binaryPath); err != nil {
		return nil
	}

	if !oc.Notifications.RunSetupExperience {
		log.Debug().Msg("skipping setup experience")
		return nil
	}

	s.StartSwiftDialog(binaryPath)

	// Poll the status endpoint. This also releases the device if we're done.
	payload, err := s.OrbitClient.GetSetupExperienceStatus()
	if err != nil {
		return err
	}

	// TODO: fill this in!

	// if profiles not verified: render spinner
	// if not account verification: render spinner
	// if not bootstrap package:  render spinner

	// render software + script status

	// if all steps terminal, render close button

	// If swiftDialog isn't up yet, then launch it
	select {
	case <-s.closeChan:
		log.Debug().Str("receiver", "setup_experiencer").Msg("closing swiftDialog")
	// case s.openChan <- struct{}{}:
	// 	log.Debug().Str("receiver", "setup_experiencer").Msg("swiftDialog is opened, proceeding")
	default:
		// ok
	}
	// ok
	if s.sd == nil {
		log.Debug().Msg("JVE_LOG: 84")
		return nil
	}

	// s.counter++
	// if err := s.sd.UpdateTitle(fmt.Sprintf("Config received %d times", s.counter)); err != nil {
	// 	log.Error().Err(err).Msg("updating swiftDialog title")
	// }

	if payload.BootstrapPackage != nil && payload.BootstrapPackage.Status == fleet.MDMBootstrapPackagePending {
		log.Debug().Msg("JVE_LOG: 97")
		return nil
	}

	if len(payload.ConfigurationProfiles) > 0 && func() bool {
		for _, p := range payload.ConfigurationProfiles {
			if p.Status == fleet.MDMDeliveryPending {
				log.Debug().Msg("JVE_LOG: 104")
				return true
			}
		}
		log.Debug().Msg("JVE_LOG: 108")
		return false
	}() {
		log.Debug().Msg("JVE_LOG: 111")
		return nil
	}

	if payload.AccountConfiguration != nil && payload.AccountConfiguration.Status == "pending" {
		log.Debug().Msg("JVE_LOG: 116")
		return nil
	}

	s.sd.EnableButton1(true)

	log.Debug().Msg("JVE_LOG: 120")
	return nil
}

func (s *SetupExperiencer) StartSwiftDialog(binaryPath string) {
	if s.started {
		return
	}
	s.started = true

	readyChan := make(chan struct{})
	s.sd, _ = swiftdialog.Create(context.Background(), binaryPath)
	go func() {
		initOpts := &swiftdialog.SwiftDialogOptions{
			Title:            "none",
			Message:          "### Setting up your Mac...\n\nYour Mac is being configured by your organization using Fleet. This process may take some time to complete. Please don't attempt to restart or shut down the computer unless prompted to do so.",
			Icon:             "https://upload.wikimedia.org/wikipedia/commons/0/08/Pinterest-logo.png",
			IconSize:         48,
			MessageAlignment: swiftdialog.AlignmentCenter,
			CentreIcon:       true,
			Height:           "625",
			Big:              true,
			Progress:         1,
			ProgressText:     "Configuring your device...",
			Button1Text:      "Close",
			Button1Disabled:  true,
		}

		s.sd.Start(context.Background(), initOpts)

		log.Debug().Msg("swiftDialog process started")
		readyChan <- struct{}{}
		s.sd.Wait()
		s.closeChan <- struct{}{}
	}()
	<-readyChan
}
