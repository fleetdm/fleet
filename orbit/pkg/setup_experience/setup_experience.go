package setupexperience

import (
	"context"
	"errors"
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
// If the setup experience is supposed to run, it will launch a single swiftDialog instance and then
// update that instance based on the results from the /orbit/setup_experience/status endpoint.
type SetupExperiencer struct {
	OrbitClient Client
	closeChan   chan struct{}
	rootDirPath string
	// Note: this object is not safe for concurrent use. Since the SetupExperiencer is a singleton,
	// its Run method is called within a WaitGroup,
	// and no other parts of Orbit need access to this field (or any other parts of the
	// SetupExperiencer), it's OK to not protect this with a lock.
	sd *swiftdialog.SwiftDialog
	// Name of each step -> is that step done
	steps   map[string]bool
	started bool
}

func NewSetupExperiencer(client Client, rootDirPath string) *SetupExperiencer {
	return &SetupExperiencer{
		OrbitClient: client,
		closeChan:   make(chan struct{}),
		steps:       make(map[string]bool),
		rootDirPath: rootDirPath,
	}
}

func (s *SetupExperiencer) Run(oc *fleet.OrbitConfig) error {
	if !oc.Notifications.RunSetupExperience {
		log.Debug().Msg("skipping setup experience: notification flag is not set")
		return nil
	}

	_, binaryPath, _ := update.LocalTargetPaths(
		s.rootDirPath,
		"swiftDialog",
		update.SwiftDialogMacOSTarget,
	)

	if _, err := os.Stat(binaryPath); err != nil {
		log.Debug().Msg("skipping setup experience: swiftDialog is not installed")
		return nil
	}

	// Poll the status endpoint. This also releases the device if we're done.
	payload, err := s.OrbitClient.GetSetupExperienceStatus()
	if err != nil {
		return err
	}

	// If swiftDialog isn't up yet, then launch it
	orgLogo := payload.OrgLogoURL
	if orgLogo == "" {
		orgLogo = "https://fleetdm.com/images/permanent/fleet-mark-color-40x40@4x.png"
	}

	if err := s.startSwiftDialog(binaryPath, orgLogo); err != nil {
		return err
	}

	// Defer this so that s.started is only false the first time this function runs.
	defer func() { s.started = true }()

	select {
	case <-s.closeChan:
		log.Debug().Str("receiver", "setup_experiencer").Msg("swiftDialog closed")
		return nil
	default:
		// ok
	}

	// We're rendering the initial loading UI (shown while there are still profiles, bootstrap package,
	// and account configuration to verify) right off the bat, so we can just no-op if any of those
	// are not terminal

	if payload.BootstrapPackage != nil {
		if payload.BootstrapPackage.Status != fleet.MDMBootstrapPackageFailed && payload.BootstrapPackage.Status != fleet.MDMBootstrapPackageInstalled {
			return nil
		}
	}

	s.steps["bootstrap"] = true

	if anyProfilePending(payload.ConfigurationProfiles) {
		return nil
	}

	s.steps["config_profiles"] = true

	if payload.AccountConfiguration != nil {
		if payload.AccountConfiguration.Status != fleet.MDMAppleStatusAcknowledged &&
			payload.AccountConfiguration.Status != fleet.MDMAppleStatusError &&
			payload.AccountConfiguration.Status != fleet.MDMAppleStatusCommandFormatError {
			return nil
		}
	}

	s.steps["account_config"] = true

	// Now render the UI for the software and script.
	if len(payload.Software) > 0 || payload.Script != nil {
		var stepsDone int
		var prog uint
		var steps []*fleet.SetupExperienceStatusResult
		if len(payload.Software) > 0 {
			steps = payload.Software
		}

		if payload.Script != nil {
			steps = append(steps, payload.Script)
		}

		for _, step := range steps {
			item := resultToListItem(step)
			if _, ok := s.steps[step.Name]; ok {
				err = s.sd.UpdateListItemByTitle(item.Title, item.StatusText, item.Status)
				if err != nil {
					log.Info().Err(err).Msg("updating list item in setup experience UI")
				}
			} else {
				err = s.sd.AddListItem(item)
				if err != nil {
					log.Info().Err(err).Msg("adding list item in setup experience UI")
				}
				s.steps[step.Name] = false
			}

			if step.Status == fleet.SetupExperienceStatusFailure || step.Status == fleet.SetupExperienceStatusSuccess {
				stepsDone++
				s.steps[step.Name] = true
				// The swiftDialog progress bar is out of 100
				for range int(float32(1) / float32(len(steps)) * 100) {
					prog++
				}
			}
		}

		if err = s.sd.UpdateProgress(prog); err != nil {
			log.Info().Err(err).Msg("updating progress bar in setup experience UI")
		}

		if err := s.sd.ShowList(); err != nil {
			log.Info().Err(err).Msg("showing progress bar in setup experience UI")
		}

		if err := s.sd.UpdateProgressText(fmt.Sprintf("%.0f%%", float32(stepsDone)/float32(len(steps))*100)); err != nil {
			log.Info().Err(err).Msg("updating progress text in setup experience UI")
		}

	}

	// If we get here, we can render the "done" UI.

	if s.allStepsDone() {
		if err := s.sd.SetMessage(doneMessage); err != nil {
			log.Info().Err(err).Msg("setting message in setup experience UI")
		}

		if err := s.sd.CompleteProgress(); err != nil {
			log.Info().Err(err).Msg("completing progress bar in setup experience UI")
		}

		if len(payload.Software) > 0 || payload.Script != nil {
			// need to call this because SetMessage removes the list from the view for some reason :(
			if err := s.sd.ShowList(); err != nil {
				log.Info().Err(err).Msg("showing list in setup experience UI")
			}
		}

		if err := s.sd.UpdateProgressText("100%"); err != nil {
			log.Info().Err(err).Msg("updating progress text in setup experience UI")
		}

		if err := s.sd.EnableButton1(true); err != nil {
			log.Info().Err(err).Msg("enabling close button in setup experience UI")
		}
	}

	return nil
}

func (s *SetupExperiencer) allStepsDone() bool {
	for _, done := range s.steps {
		if !done {
			return false
		}
	}

	return true
}

func anyProfilePending(profiles []*fleet.SetupExperienceConfigurationProfileResult) bool {
	for _, p := range profiles {
		if p.Status == fleet.MDMDeliveryPending {
			return true
		}
	}

	return false
}

func (s *SetupExperiencer) startSwiftDialog(binaryPath, orgLogo string) error {
	if s.started {
		return nil
	}

	created := make(chan struct{})
	swiftDialog, err := swiftdialog.Create(context.Background(), binaryPath)
	if err != nil {
		return errors.New("creating swiftDialog instance: %w")
	}
	s.sd = swiftDialog
	go func() {
		initOpts := &swiftdialog.SwiftDialogOptions{
			Title:            "none",
			Message:          "### Setting up your Mac...\n\nYour Mac is being configured by your organization using Fleet. This process may take some time to complete. Please don't attempt to restart or shut down the computer unless prompted to do so.",
			Icon:             orgLogo,
			MessageAlignment: swiftdialog.AlignmentCenter,
			CentreIcon:       true,
			Height:           "625",
			Big:              true,
			ProgressText:     "Configuring your device...",
			Button1Text:      "Close",
			Button1Disabled:  true,
		}

		if err := s.sd.Start(context.Background(), initOpts); err != nil {
			log.Error().Err(err).Msg("starting swiftDialog instance")
		}

		if err = s.sd.ShowProgress(); err != nil {
			log.Error().Err(err).Msg("setting initial setup experience progress")
		}

		if err := s.sd.SetIconSize(80); err != nil {
			log.Error().Err(err).Msg("setting initial setup experience icon size")
		}

		log.Debug().Msg("swiftDialog process started")
		created <- struct{}{}

		if _, err = s.sd.Wait(); err != nil {
			log.Error().Err(err).Msg("swiftdialog.Wait failed")
		}

		s.closeChan <- struct{}{}
	}()
	<-created
	return nil
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
