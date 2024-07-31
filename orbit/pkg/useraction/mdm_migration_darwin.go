//go:build darwin

package useraction

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/rs/zerolog/log"
)

type swiftDialogExitCode int

const (
	primaryBtnExitCode   = 0
	errorExitCode        = 1
	secondaryBtnExitCode = 2
	infoBtnExitCode      = 3
	timeoutExitCode      = 4
	userQuitExitCode     = 10
	unknownExitCode      = 99
)

// mdmEnrollmentFile is a file that we use as a sentinel value to detect MDM
// enrollment. The file must be present if the device is enrolled and absent
// otherwise. We have found that this file accomplishes this purpose for DEP
// enrollments, which is the only type of migration supported at the moment.
//
// Optionally we could use the output of `profiles show --type enrollment` to
// accomplish the same thing, but it's more resource intensive and harder for
// people that build integrations on top of the migration flow.
var mdmEnrollmentFile = "/private/var/db/ConfigurationProfiles/Settings/.cloudConfigProfileInstalled"

// mdmUnenrollmentTotalWaitTime defines how long the dialog is going to wait
// for the device to be unenrolled before bailing out and showing an error
// message.
const mdmUnenrollmentTotalWaitTime = 90 * time.Second

// defaultUnenrollmentRetryInterval defines how long we're going to wait
// between unenrollment checks.
const defaultUnenrollmentRetryInterval = 5 * time.Second

var mdmMigrationTemplatePreSonoma = template.Must(template.New("mdmMigrationTemplate").Parse(`
## Migrate to Fleet

Select **Start** and look for this notification in your notification center:` +
	"\n\n![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-migration-screenshot-notification-2048x480.png)\n\n" +
	"After you start, this window will popup every 15-20 minutes until you finish.",
))

var mdmMigrationTemplate = template.Must(template.New("mdmMigrationTemplate").Parse(`
## Migrate to Fleet

Select **Start** and Remote Management window will appear soon:` +
	"\n\n![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-migration-sonoma-1500x938.png)\n\n" +
	"After you start, this window will popup every 15-20 minutes until you finish.",
))

var errorTemplate = template.Must(template.New("").Parse(`
### Something's gone wrong.

Please contact your IT admin [here]({{ .ContactURL }}).
`))

// baseDialog implements the basic building blocks to render dialogs using
// swiftDialog.
type baseDialog struct {
	path        string
	interruptCh chan struct{}
}

func newBaseDialog(path string) *baseDialog {
	return &baseDialog{path: path, interruptCh: make(chan struct{})}
}

func (b *baseDialog) CanRun() bool {
	// check if swiftDialog has been downloaded
	if _, err := os.Stat(b.path); err != nil {
		return false
	}

	return true
}

// Exit sends the interrupt signal to try and stop the current swiftDialog
// instance.
func (b *baseDialog) Exit() {
	b.interruptCh <- struct{}{}
	log.Info().Msg("dialog exit message sent")
}

// render is a general-purpose render method that receives the flags used to
// display swiftDialog, and starts an asyncronous routine to display the dialog
// without blocking.
//
// The first returned channel sends the exit code returned by swiftDialog, and
// the second channel is used to send errors.
func (b *baseDialog) render(flags ...string) (chan swiftDialogExitCode, chan error) {
	exitCodeCh := make(chan swiftDialogExitCode, 1)
	errCh := make(chan error, 1)
	go func() {
		// all dialogs should always be blurred and on top
		flags = append(
			flags,
			"--blurscreen",
			"--ontop",
			"--messageposition", "center",
		)
		cmd := exec.Command(b.path, flags...) //nolint:gosec
		done := make(chan error)
		stopInterruptCh := make(chan struct{})
		defer close(stopInterruptCh)

		if err := cmd.Start(); err != nil {
			errCh <- err
			return
		}

		go func() { done <- cmd.Wait() }()
		go func() {
			select {
			case <-b.interruptCh:
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					log.Error().Err(err).Msg("sending interrupt signal to swiftDialog process")
					if err := cmd.Process.Kill(); err != nil {
						log.Error().Err(err).Msg("killing swiftDialog process")
						errCh <- errors.New("failed to stop/kill swiftDialog process")
					}
				}
			case <-stopInterruptCh:
				return
			}
		}()

		if err := <-done; err != nil {
			// non-zero exit codes
			if exitError, ok := err.(*exec.ExitError); ok {
				ec := exitError.ExitCode()
				switch ec {
				case errorExitCode:
					exitCodeCh <- errorExitCode
				case secondaryBtnExitCode, infoBtnExitCode, timeoutExitCode:
					exitCodeCh <- swiftDialogExitCode(ec)
				default:
					errCh <- fmt.Errorf("unknown exit code showing dialog: %w", exitError)
				}
			} else {
				errCh <- fmt.Errorf("running swiftDialog: %w", err)
			}
		} else {
			exitCodeCh <- 0
		}
	}()
	return exitCodeCh, errCh
}

// NewMDMMigrator creates a new  swiftDialogMDMMigrator with the right internal state.
func NewMDMMigrator(path string, frequency time.Duration, handler MDMMigratorHandler) MDMMigrator {
	return &swiftDialogMDMMigrator{
		handler:                   handler,
		baseDialog:                newBaseDialog(path),
		frequency:                 frequency,
		unenrollmentRetryInterval: defaultUnenrollmentRetryInterval,
		// set a buffer size of 1 to allow one Show without blocking
		showCh: make(chan struct{}, 1),
	}
}

// swiftDialogMDMMigrator implements MDMMigrator for macOS using swiftDialog as
// the underlying mechanism for user action.
type swiftDialogMDMMigrator struct {
	*baseDialog
	props     MDMMigratorProps
	frequency time.Duration
	handler   MDMMigratorHandler

	// ensures only one dialog is open at a time, protects access to
	// lastShown
	lastShown   time.Time
	lastShownMu sync.RWMutex
	showCh      chan struct{}

	// testEnrollmentCheckFileFn is used in tests to mock the call to verify
	// the enrollment status of the host
	testEnrollmentCheckFileFn func() (bool, error)
	// testEnrollmentCheckStatusFn is used in tests to mock the call to verify
	// the enrollment status of the host
	testEnrollmentCheckStatusFn func() (bool, string, error)
	unenrollmentRetryInterval   time.Duration
}

/**
 * Checks in macOS if the user is using dark mode. If we encounter an exit error this is because
 * out command returned a non-zero exit code. In this case we can assume the user is NOT using dark
 * mode as the "AppleInterfaceStyle" key is only set when dark mode has been set.
 *
 * More info can be found here:
 * https://gist.github.com/jerblack/869a303d1a604171bf8f00bbbefa59c2#file-2-dark-monitor-go-L33-L41
 */
func isDarkMode() bool {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		}
	}
	return true
}

func (m *swiftDialogMDMMigrator) render(message string, flags ...string) (chan swiftDialogExitCode, chan error) {
	icon := m.props.OrgInfo.OrgLogoURL

	// If the user is using light mode we will set the icon to use the light background logo
	if !isDarkMode() {
		icon = m.props.OrgInfo.OrgLogoURLLightBackground
	}

	// If the user has not set an org logo url, we will use the default fleet logo.
	if icon == "" {
		icon = "https://fleetdm.com/images/permanent/fleet-mark-color-40x40@4x.png"
	}

	flags = append([]string{
		// disable the built-in title so we have full control over the
		// content
		"--title", "none",
		// top icon
		"--icon", icon,
		"--iconsize", "80",
		"--centreicon",
		// modal content
		"--message", message,
		"--messagefont", "size=16",
		"--alignment", "center",
	}, flags...)

	return m.baseDialog.render(flags...)
}

func (m *swiftDialogMDMMigrator) renderLoadingSpinner() (chan swiftDialogExitCode, chan error) {
	return m.render("## Migrate to Fleet\nUnenrolling you from your old MDM. This could take 90 seconds...",
		"--button1text", "Start",
		"--button1disabled",
		"--quitkey", "x",
		"--height", "220",
	)
}

func (m *swiftDialogMDMMigrator) renderError() (chan swiftDialogExitCode, chan error) {
	var errorMessage bytes.Buffer
	if err := errorTemplate.Execute(
		&errorMessage,
		m.props.OrgInfo,
	); err != nil {
		codeChan := make(chan swiftDialogExitCode, 1)
		errChan := make(chan error, 1)
		errChan <- fmt.Errorf("execute error template: %w", err)
		return codeChan, errChan
	}

	return m.render(errorMessage.String(), "--button1text", "Close", "--height", "220")
}

// waitForUnenrollment waits 90 seconds (value determined by product) for the
// device to unenroll from the current MDM solution. If the device doesn't
// unenroll, an error is returned.
func (m *swiftDialogMDMMigrator) waitForUnenrollment() error {
	maxRetries := int(mdmUnenrollmentTotalWaitTime.Seconds() / m.unenrollmentRetryInterval.Seconds())
	checkFileFn := m.testEnrollmentCheckFileFn
	if checkFileFn == nil {
		checkFileFn = func() (bool, error) {
			return file.Exists(mdmEnrollmentFile)
		}
	}
	checkStatusFn := m.testEnrollmentCheckStatusFn
	if checkStatusFn == nil {
		checkStatusFn = func() (bool, string, error) {
			return profiles.IsEnrolledInMDM()
		}
	}
	return retry.Do(func() error {
		var unenrolled bool

		fileExists, fileErr := checkFileFn()
		if fileErr != nil {
			log.Error().Err(fileErr).Msg("checking for existence of cloudConfigProfileInstalled in migration modal")
		} else if fileExists {
			log.Info().Msg("checking for existence of cloudConfigProfileInstalled in migration modal: found")
		} else {
			log.Info().Msg("checking for existence of cloudConfigProfileInstalled in migration modal: not found")
			unenrolled = true
		}

		statusEnrolled, serverURL, statusErr := checkStatusFn()
		if statusErr != nil {
			log.Error().Err(statusErr).Msgf("checking profiles status in migration modal")
		} else if statusEnrolled {
			log.Info().Msgf("checking profiles status in migration modal: enrolled to %s", serverURL)
		} else {
			log.Info().Msg("checking profiles status in migration modal: not enrolled")
			unenrolled = true
		}

		if !unenrolled {
			log.Info().Msgf("device is still enrolled, waiting %s", m.unenrollmentRetryInterval)
			return errors.New("host didn't unenroll from MDM")
		}

		log.Info().Msg("device is unenrolled, closing migration modal")
		return nil
	},
		retry.WithMaxAttempts(maxRetries),
		retry.WithInterval(m.unenrollmentRetryInterval),
	)
}

func (m *swiftDialogMDMMigrator) renderMigration() error {
	message, flags, err := m.getMessageAndFlags()
	if err != nil {
		return fmt.Errorf("getting mdm migrator message: %w", err)
	}

	exitCodeCh, errCh := m.render(message.String(), flags...)

	select {
	case err := <-errCh:
		return fmt.Errorf("showing start migration dialog: %w", err)
	case exitCode := <-exitCodeCh:
		// we don't perform any action for all the other buttons
		if exitCode != primaryBtnExitCode {
			return nil
		}

		if !m.props.IsUnmanaged {
			// show the loading spinner
			m.renderLoadingSpinner()

			// send the API call
			if notifyErr := m.handler.NotifyRemote(); notifyErr != nil {
				m.baseDialog.Exit()
				errDialogExitChan, errDialogErrChan := m.renderError()
				select {
				case <-errDialogExitChan:
					// return the error after showing the
					// dialog so it can be caught upstream.
					return notifyErr
				case err := <-errDialogErrChan:
					return fmt.Errorf("rendering error dialog: %w", err)
				}
			}

			log.Info().Msg("webhook sent, checking for unenrollment")
			if err := m.waitForUnenrollment(); err != nil {
				m.baseDialog.Exit()
				errDialogExitChan, errDialogErrChan := m.renderError()
				select {
				case <-errDialogExitChan:
					// return the error after showing the
					// dialog so it can be caught upstream.
					return err
				case err := <-errDialogErrChan:
					return fmt.Errorf("rendering error dialog: %w", err)
				}
			}

			// close the spinner
			// TODO: maybe it's better to use
			// https://github.com/bartreardon/swiftDialog/wiki/Updating-Dialog-with-new-content
			// instead? it uses a file as IPC
			m.baseDialog.Exit()
		}

		// TODO(JVE): I think this is where we have to show the Remote Management modal instead
		log.Debug().Msg("checking manual enrollment status")
		manual, err := profiles.IsManuallyEnrolledInMDM()
		if err != nil {
			return err
		}

		if manual {
			log.Debug().Msg("host is manually enrolled")
			log.Info().Msg("showing instructions")
			if err := m.handler.ShowInstructions(); err != nil {
				return err
			}
		} else {
			log.Debug().Msg("host is DEP enrolled")
		}

	}

	return nil
}

// Show displays the dialog every time is called, as long as there isn't a
// dialog already open.
func (m *swiftDialogMDMMigrator) Show() error {
	select {
	case m.showCh <- struct{}{}:
		defer func() { <-m.showCh }()
	default:
		log.Info().Msg("there's a migration dialog already open, refusing to launch")
		return nil
	}

	if err := m.renderMigration(); err != nil {
		return fmt.Errorf("show: %w", err)
	}

	m.lastShownMu.Lock()
	m.lastShown = time.Now()
	m.lastShownMu.Unlock()

	return nil
}

// ShowInterval acts as a rate limiter for Show, it only calls the function IIF
// m.frequency has passed since the last time the dialog was successfully
// shown.
func (m *swiftDialogMDMMigrator) ShowInterval() error {
	m.lastShownMu.RLock()
	lastShown := m.lastShown
	m.lastShownMu.RUnlock()
	if time.Since(lastShown) <= m.frequency {
		log.Info().Msg("dialog was automatically launched too recently, skipping")
		return nil
	}

	if err := m.Show(); err != nil {
		return fmt.Errorf("show interval: %w", err)
	}

	return nil
}

func (m *swiftDialogMDMMigrator) SetProps(props MDMMigratorProps) {
	m.props = props
}

func (m *swiftDialogMDMMigrator) getMessageAndFlags() (*bytes.Buffer, []string, error) {
	vers, err := m.getMacOSMajorVersion()
	if err != nil {
		// log error for debugging and continue with default template
		log.Error().Err(err).Msg("getting macOS major version failed: using default migration template")
	}

	tmpl := mdmMigrationTemplate
	height := "669"
	if vers != 0 && vers < 14 {
		height = "440"
		tmpl = mdmMigrationTemplatePreSonoma
	}

	var message bytes.Buffer
	if err := tmpl.Execute(
		&message,
		m.props,
	); err != nil {
		return nil, nil, fmt.Errorf("executing migrqation template: %w", err)
	}

	flags := []string{
		// main button
		"--button1text", "Start",
		// secondary button
		"--button2text", "Later",
		"--height", height,
	}

	if m.props.OrgInfo.ContactURL != "" {
		flags = append(flags,
			// info button
			"--infobuttontext", "Unsure? Contact IT",
			"--infobuttonaction", m.props.OrgInfo.ContactURL,
			"--quitoninfo",
		)
	}

	return &message, flags, nil
}

// TODO: make this a variable for testing
func (m *swiftDialogMDMMigrator) getMacOSMajorVersion() (int, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("getting macOS version: %w", err)
	}
	parts := strings.SplitN(string(out), ".", 2)
	switch len(parts) {
	case 0:
		// this should never happen
		return 0, errors.New("getting macOS version: sw_vers command returned no output")
	case 1:
		// unexpected, so log for debugging
		log.Debug().Msgf("parsing macOS version: expected 2 parts, got 1: %s", out)
	default:
		// ok
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("parsing macOS major version: %w", err)
	}
	return major, nil
}
