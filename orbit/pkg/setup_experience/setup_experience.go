package setupexperience

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/swiftdialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/rs/zerolog/log"
)

// OrbitClient is the minimal interface needed to communicate with the Fleet server.
type OrbitClient interface {
	GetSetupExperienceStatus() (*fleet.SetupExperienceStatusPayload, error)
}

// DeviceClient is the minimal interface needed to get the device's browser URL.
type DeviceClient interface {
	BrowserDeviceURL(token string) string
}

// SetupExperiencer is the type that manages the Fleet setup experience flow during macOS Setup
// Assistant. It uses swiftDialog as a UI for showing the status of software installations and
// script execution that are configured to run before the user has full access to the device.
// If the setup experience is supposed to run, it will launch a single swiftDialog instance and then
// update that instance based on the results from the /orbit/setup_experience/status endpoint.
type SetupExperiencer struct {
	OrbitClient  OrbitClient
	DeviceClient DeviceClient
	closeChan    chan struct{}
	rootDirPath  string
	// Note: this object is not safe for concurrent use. Since the SetupExperiencer is a singleton,
	// its Run method is called within a WaitGroup,
	// and no other parts of Orbit need access to this field (or any other parts of the
	// SetupExperiencer), it's OK to not protect this with a lock.
	sd                *swiftdialog.SwiftDialog
	uiSteps           map[string]swiftdialog.ListItem
	started           bool
	trw               *token.ReadWriter
	stopTokenRotation func()
}

func NewSetupExperiencer(orbitClient OrbitClient, deviceClient DeviceClient, rootDirPath string, trw *token.ReadWriter) *SetupExperiencer {
	return &SetupExperiencer{
		OrbitClient:  orbitClient,
		DeviceClient: deviceClient,
		closeChan:    make(chan struct{}),
		uiSteps:      make(map[string]swiftdialog.ListItem),
		rootDirPath:  rootDirPath,
		trw:          trw,
	}
}

func (s *SetupExperiencer) Run(oc *fleet.OrbitConfig) error {
	if !oc.Notifications.RunSetupExperience {
		log.Debug().Msg("skipping setup experience: notification flag is not set")
		return nil
	}

	// Ensure that the token rotation checker is started, so that we have a valid token
	// when we need to show or refresh the My Device URL in the webview.
	if s.stopTokenRotation == nil {
		s.stopTokenRotation = s.trw.StartRotation()
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
	// Marshall the payload for logging
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("marshalling setup experience payload for logging")
	} else {
		log.Info().Msgf("setup experience payload: %s", string(payloadBytes))
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

	// If we got this far, then we can hand the UI over to the webview.

	// Clear the dialog message.
	if err := s.sd.HideMessage(); err != nil {
		log.Info().Err(err).Msg("clearing message in setup experience UI")
	}
	// Remove the icon.
	if err := s.sd.HideIcon(); err != nil {
		log.Info().Err(err).Msg("clearing icon in setup experience UI")
	}
	// Hide the title.
	if err := s.sd.HideTitle(); err != nil {
		log.Info().Err(err).Msg("hiding title in setup experience UI")
	}
	// Hide the progress.
	if err := s.sd.HideProgress(); err != nil {
		log.Info().Err(err).Msg("hiding progress in setup experience UI")
	}
	// Get the device token.
	token, err := s.trw.Read()
	if err != nil {
		return fmt.Errorf("getting device token: %w", err)
	}
	// Get the My Device URL.
	browserURL := s.DeviceClient.BrowserDeviceURL(token)
	// log out the url
	log.Info().Msgf("setup experience: opening web content URL: %s", browserURL)
	// Set the web content URL.
	if err := s.sd.SetWebContent(browserURL + "?setup_only=1"); err != nil {
		log.Info().Err(err).Msg("setting web content URL in setup experience UI")
		return nil
	}

	// Note that we are setting this based on the current payload only just in case something
	// was removed from the payload that was there earlier(e.g. a deleted software title).
	allStepsDone := true

	// Now render the UI for the software and script.
	if len(payload.Software) > 0 || payload.Script != nil {
		log.Info().Msg("setup experience: rendering software and script UI")

		var steps []*fleet.SetupExperienceStatusResult
		if len(payload.Software) > 0 {
			steps = payload.Software
		}

		if payload.Script != nil {
			steps = append(steps, payload.Script)
		}

		for _, step := range steps {
			if step.Status != fleet.SetupExperienceStatusFailure && step.Status != fleet.SetupExperienceStatusSuccess {
				allStepsDone = false
			}
		}
	}

	// If we get here, we can close the webview.
	// It will likely already be displaying a "done" message.
	if allStepsDone {
		if err := s.sd.EnableButton1(true); err != nil {
			log.Info().Err(err).Msg("enabling close button in setup experience UI")
		}

		// Sleep for a few seconds to let the user see the done message before closing
		// the UI
		time.Sleep(3 * time.Second)

		if err := s.sd.Quit(); err != nil {
			log.Info().Err(err).Msg("quitting setup experience UI on completion")
		}

		// Stop the token rotation checker since we're done with the setup experience.
		s.stopTokenRotation()
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

// LinuxSetupExperiencer runs the setup experience on Linux hosts.
type LinuxSetupExperiencer struct {
	orbitClient OrbitClient
	rootDir     string
}

// NewLinuxSetupExperiencer creates a config receiver to run the setup experience on Linux hosts.
func NewLinuxSetupExperiencer(client OrbitClient, rootDir string) *LinuxSetupExperiencer {
	return &LinuxSetupExperiencer{
		orbitClient: client,
		rootDir:     rootDir,
	}
}

// Run implements fleet.OrbitConfigReceiver.
//
// Currently the fleet.OrbitConfig is ununsed but might be used in the future.
func (s *LinuxSetupExperiencer) Run(_ *fleet.OrbitConfig) error {
	info, err := ReadSetupExperienceStatusFile(s.rootDir)
	if err != nil {
		return fmt.Errorf("read setup experience file: %w", err)
	}
	if info == nil || info.TimeFinished != nil {
		// nothing to do.
		return nil
	}

	payload, err := s.orbitClient.GetSetupExperienceStatus()
	if err != nil {
		return err
	}

	if setupExperienceDone(payload) {
		info.TimeFinished = ptr.Time(time.Now())
		if err := WriteSetupExperienceStatusFile(s.rootDir, info); err != nil {
			log.Error().Err(err).Msg("write setup experience status file")
		}
	}
	return nil
}

func setupExperienceDone(payload *fleet.SetupExperienceStatusPayload) bool {
	for _, software := range payload.Software {
		if software != nil && (software.Status == fleet.SetupExperienceStatusPending || software.Status == fleet.SetupExperienceStatusRunning) {
			return false
		}
	}
	return true
}

// SetupExperienceInfo holds information of the state of the setup experience for a host.
type SetupExperienceInfo struct {
	// TimeInitiated is the time the setup experience was attempted during setup/installation.
	TimeInitiated time.Time `json:"time_initiated"`
	// Enabled is true if the setup experience was enabled during setup/installation.
	Enabled bool `json:"enabled"`
	// TimeFinished is the time the setup experience was finished by the host.
	TimeFinished *time.Time `json:"time_finished,omitempty"`
}

// ReadSetupExperienceStatusFile reads the setup experience state from a known file in the rootDir.
func ReadSetupExperienceStatusFile(rootDir string) (*SetupExperienceInfo, error) {
	infoPath := filepath.Join(rootDir, constant.SetupExperienceFilename)

	f, err := os.Open(infoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read setup experience file: %w", err)
	}
	defer f.Close()
	var exp SetupExperienceInfo
	if err := json.NewDecoder(f).Decode(&exp); err != nil {
		return nil, fmt.Errorf("decoding setup experience file: %w", err)
	}

	return &exp, nil
}

// WriteSetupExperienceStatusFile writes the setup experience state to a file under rootDir.
func WriteSetupExperienceStatusFile(rootDir string, exp *SetupExperienceInfo) error {
	infoPath := filepath.Join(rootDir, constant.SetupExperienceFilename)

	f, err := os.OpenFile(infoPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, constant.DefaultFileMode)
	if err != nil {
		return fmt.Errorf("create setup experience completed file: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(exp); err != nil {
		return fmt.Errorf("write setup experience completed file: %w", err)
	}
	return nil
}
