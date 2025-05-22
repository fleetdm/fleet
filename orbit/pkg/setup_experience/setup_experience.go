package setupexperience

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

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
	sd      *swiftdialog.SwiftDialog
	uiSteps map[string]swiftdialog.ListItem
	started bool
}

func NewSetupExperiencer(client Client, rootDirPath string) *SetupExperiencer {
	return &SetupExperiencer{
		OrbitClient: client,
		closeChan:   make(chan struct{}),
		uiSteps:     make(map[string]swiftdialog.ListItem),
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
		log.Info().Msg("skipping setup experience: swiftDialog is not installed")
		return nil
	}

	log.Info().Msg("checking setup experience status")

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
		log.Info().Str("receiver", "setup_experiencer").Msg("swiftDialog closed")
		return nil
	default:
		// ok
	}

	// We're rendering the initial loading UI (shown while there are still profiles, bootstrap package,
	// and account configuration to verify) right off the bat, so we can just no-op if any of those
	// are not terminal

	log.Info().Msg("setup experience: checking for pending statuses")

	if payload.BootstrapPackage != nil {
		if payload.BootstrapPackage.Status != fleet.MDMBootstrapPackageFailed && payload.BootstrapPackage.Status != fleet.MDMBootstrapPackageInstalled {
			log.Info().Msg("setup experience: bootstrap package pending")
			return nil
		}
	}

	if isPending, name := anyProfilePending(payload.ConfigurationProfiles); isPending {
		log.Info().Msg(fmt.Sprintf("setup experience: profile pending: %s", name))
		return nil
	}

	if payload.AccountConfiguration != nil {
		if payload.AccountConfiguration.Status != fleet.MDMAppleStatusAcknowledged &&
			payload.AccountConfiguration.Status != fleet.MDMAppleStatusError &&
			payload.AccountConfiguration.Status != fleet.MDMAppleStatusCommandFormatError {

			log.Info().Msg("setup experience: account config pending")
			return nil
		}
	}

	// Note that we are setting this based on the current payload only just in case something
	// was removed from the payload that was there earlier(e.g. a deleted software title).
	allStepsDone := true

	// Now render the UI for the software and script.
	if len(payload.Software) > 0 || payload.Script != nil {
		log.Info().Msg("setup experience: rendering software and script UI")

		var stepsDone int
		var prog uint
		var steps []*fleet.SetupExperienceStatusResult
		if len(payload.Software) > 0 {
			steps = payload.Software
		}

		if payload.Script != nil {
			steps = append(steps, payload.Script)
		}

		// Check for any items that were in the payload that are no longer there. This can happen
		// if a software title was deleted, for instance
		for uiStepName, uiStep := range s.uiSteps {
			uiStepExistsInPayload := false
			for _, step := range steps {
				if uiStep.Title == step.Name {
					uiStepExistsInPayload = true
					break
				}
			}
			if !uiStepExistsInPayload {
				log.Info().Msgf("Setup Experience: list item %s removed from payload", uiStep.Title)
				err = s.sd.DeleteListItemByTitle(uiStep.Title)
				if err != nil {
					log.Info().Err(err).Msg("deleting list item removed from payload from setup experience UI")
				}
				delete(s.uiSteps, uiStepName)
			}
		}

		for _, step := range steps {
			currentStepState := resultToListItem(step)
			if priorStepState, ok := s.uiSteps[step.Name]; ok {
				if currentStepState != priorStepState {
					// We only want to resend on change so we're not unnecessarily scrolling the UI
					err = s.sd.UpdateListItemByTitle(currentStepState.Title, currentStepState.StatusText, currentStepState.Status)
					if err != nil {
						log.Info().Err(err).Msg("updating list item in setup experience UI")
					}
				} else {
					log.Info().Msgf("setup experience: no change in status for %s", step.Name)
				}
			} else {
				err = s.sd.AddListItem(currentStepState)
				if err != nil {
					log.Info().Err(err).Msg("adding list item in setup experience UI")
				}
				s.uiSteps[step.Name] = currentStepState
			}

			if step.Status == fleet.SetupExperienceStatusFailure || step.Status == fleet.SetupExperienceStatusSuccess {
				stepsDone++
				// The swiftDialog progress bar is out of 100
				for range int(float32(1) / float32(len(steps)) * 100) {
					prog++
				}
			} else {
				allStepsDone = false
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

	if allStepsDone {
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

		// Sleep for a few seconds to let the user see the done message before closing
		// the UI
		time.Sleep(3 * time.Second)

		if err := s.sd.Quit(); err != nil {
			log.Info().Err(err).Msg("quitting setup experience UI on completion")
		}
	}

	return nil
}

func anyProfilePending(profiles []*fleet.SetupExperienceConfigurationProfileResult) (bool, string) {
	for _, p := range profiles {
		if p.Status == fleet.MDMDeliveryPending {
			return true, p.Name
		}
	}

	return false, ""
}

func (s *SetupExperiencer) startSwiftDialog(binaryPath, orgLogo string) error {
	if s.started {
		log.Info().Msg("swiftDialog started")
		return nil
	}

	log.Info().Msg("creating swiftDialog instance")

	created := make(chan struct{})
	swiftDialog, err := swiftdialog.Create(context.Background(), binaryPath)
	if err != nil {
		return errors.New("creating swiftDialog instance: %w")
	}
	s.sd = swiftDialog

	iconSize, err := swiftdialog.GetIconSize(orgLogo)
	if err != nil {
		log.Error().Err(err).Msg("setup experience: getting icon size")
		iconSize = swiftdialog.DefaultIconSize
	}

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
			BlurScreen:       true,
			OnTop:            true,
			QuitKey:          "X", // Capital X to require command+shift+x
		}

		if err := s.sd.Start(context.Background(), initOpts, true); err != nil {
			log.Error().Err(err).Msg("starting swiftDialog instance")
		}

		if err = s.sd.ShowProgress(); err != nil {
			log.Error().Err(err).Msg("setting initial setup experience progress")
		}

		if err := s.sd.SetIconSize(iconSize); err != nil {
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
