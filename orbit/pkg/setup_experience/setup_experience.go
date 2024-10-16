package setupexperience

import (
	"context"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/swiftdialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

const doneMessage = `### Setup is complete\n\nPlease contact your IT Administrator if there were any errors.`

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

	// If swiftDialog isn't up yet, then launch it
	s.StartSwiftDialog(binaryPath)
	defer func() { s.started = true }()

	// Poll the status endpoint. This also releases the device if we're done.
	payload, err := s.OrbitClient.GetSetupExperienceStatus()
	if err != nil {
		return err
	}

	// TODO(JVE): do we need this now that we're blocking up in StartSwiftDialog?
	select {
	case <-s.closeChan:
		log.Debug().Str("receiver", "setup_experiencer").Msg("closing swiftDialog")
	default:
		// ok
	}

	// We're rendering the initial loading UI (shown while there are still profiles, bootstrap package,
	// and account configuration to verify) right off the bat, so we can just no-op if any of those
	// are not terminal
	if s.sd == nil {
		log.Debug().Msg("JVE_LOG: 84")
		return nil
	}
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

	// Now render the UI for the software and script.
	if len(payload.Software) > 0 || payload.Script != nil {
		var done int
		steps := append(payload.Software, payload.Script)
		for _, r := range steps {
			item := resultToListItem(r)
			if s.started {
				s.sd.UpdateListItemByTitle(item.Title, item.StatusText, item.Status)
			} else {
				s.sd.AddListItem(item)
			}
			if r.Status == fleet.SetupExperienceStatusFailure || r.Status == fleet.SetupExperienceStatusSuccess {
				done++
			}
		}
		s.sd.ShowList()
		s.sd.IncrementProgress()
		s.sd.UpdateProgressText(fmt.Sprintf("%d%%", done/len(steps)))

		if done == len(steps) {
			s.sd.SetMessage(doneMessage)
			s.sd.ShowList() // need to call this because SetMessage removes the list from the view for some reason :(
			s.sd.EnableButton1(true)
			return nil
		}
	}

	// If we get here, we can enable the button to allow the user to close the window.
	s.sd.EnableButton1(true)

	log.Debug().Msg("JVE_LOG: 120")
	return nil
}

func (s *SetupExperiencer) StartSwiftDialog(binaryPath string) {
	if s.started {
		return
	}

	readyChan := make(chan struct{})
	s.sd, _ = swiftdialog.Create(context.Background(), binaryPath)
	go func() {
		initOpts := &swiftdialog.SwiftDialogOptions{
			Title:            "none",
			Message:          "### Setting up your Mac...\n\nYour Mac is being configured by your organization using Fleet. This process may take some time to complete. Please don't attempt to restart or shut down the computer unless prompted to do so.",
			Icon:             "https://upload.wikimedia.org/wikipedia/commons/0/08/Pinterest-logo.png", // TODO(JVE): figure out how to get this
			IconSize:         48,
			MessageAlignment: swiftdialog.AlignmentCenter,
			CentreIcon:       true,
			Height:           "625",
			Big:              true,
			ProgressText:     "Configuring your device...",
			Button1Text:      "Close",
			Button1Disabled:  true,
		}

		s.sd.Start(context.Background(), initOpts)
		s.sd.ShowProgress()
		log.Debug().Msg("swiftDialog process started")
		readyChan <- struct{}{}
		s.sd.Wait()
		s.closeChan <- struct{}{}
	}()
	<-readyChan
}

func resultToListItem(result *fleet.SetupExperienceStatusResult) swiftdialog.ListItem {
	statusText := "Pending"
	status := swiftdialog.StatusWait

	switch result.Status {
	case fleet.SetupExperienceStatusFailure:
		status = swiftdialog.StatusFail
		statusText = "Failed"
	case fleet.SetupExperienceStatusSuccess:
		status = swiftdialog.StatusSuccess
		statusText = "Installed"
		if result.IsForScript() {
			statusText = "Ran"
		}
	}

	return swiftdialog.ListItem{
		Title:      result.Name,
		Status:     status,
		StatusText: statusText,
	}
}
