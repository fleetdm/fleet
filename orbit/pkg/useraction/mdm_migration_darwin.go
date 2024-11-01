//go:build darwin

package useraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/migration"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/service"
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
	"After you start, this window will popup every 3 minutes until you finish.",
))

var mdmManualMigrationTemplate = template.Must(template.New("").Parse(`
## Migrate to Fleet

Select **Start** and My device page will appear soon:` +
	"\n\n![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-manual-migration-1024x500.png)\n\n" +
	"After you start, this dialog will popup every 3 minutes until you finish.",
))

var mdmADEMigrationTemplate = template.Must(template.New("").Parse(`
## Migrate to Fleet

Select **Start** and Remote Management window will appear soon:` +
	"\n\n![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-ade-migration-1024x500.png)\n\n" +
	"After you start, **Remote Management** will popup every minute until you finish.",
))

var errorTemplate = template.Must(template.New("").Parse(`
### Something's gone wrong.

Please contact your IT admin [here]({{ .ContactURL }}).
`))

var unenrollBody = "## Migrate to Fleet\nUnenrolling you from your old MDM. This could take 90 seconds...\n\n%s"

var mdmMigrationTemplateOffline = template.Must(template.New("").Parse(`
## Migrate to Fleet

🛜🚫 No internet connection. Please connect to internet to continue.`,
))

// baseDialog implements the basic building blocks to render dialogs using
// swiftDialog.
type baseDialog struct {
	path        string
	interruptCh chan struct{}
}

func newBaseDialog(path string) *baseDialog {
	return &baseDialog{path: path, interruptCh: make(chan struct{}, 1)}
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
		// all dialogs should always be centered
		flags = append(
			flags,
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
func NewMDMMigrator(path string, frequency time.Duration, handler MDMMigratorHandler, mrw *migration.ReadWriter, fleetURL string, showCh chan struct{}) MDMMigrator {
	if cap(showCh) != 1 {
		log.Fatal().Msg("swift dialog channel must have a buffer size of 1")
	}
	return &swiftDialogMDMMigrator{
		handler:                   handler,
		baseDialog:                newBaseDialog(path),
		frequency:                 frequency,
		unenrollmentRetryInterval: defaultUnenrollmentRetryInterval,
		mrw:                       mrw,
		fleetURL:                  fleetURL,
		showCh:                    showCh,
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
	// showCh is shared with the offline watcher and used to ensure only one dialog is open at a time
	showCh chan struct{}

	// testEnrollmentCheckFileFn is used in tests to mock the call to verify
	// the enrollment status of the host
	testEnrollmentCheckFileFn func() (bool, error)
	// testEnrollmentCheckStatusFn is used in tests to mock the call to verify
	// the enrollment status of the host
	testEnrollmentCheckStatusFn func() (bool, string, error)
	unenrollmentRetryInterval   time.Duration
	mrw                         *migration.ReadWriter
	fleetURL                    string
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

func (m *swiftDialogMDMMigrator) renderLoadingSpinner(preSonoma, isManual bool) (chan swiftDialogExitCode, chan error) {
	var body string
	switch {
	case preSonoma:
		body = fmt.Sprintf(unenrollBody, "![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-migration-pre-sonoma-unenroll-1024x500.png)")
	case isManual:
		body = fmt.Sprintf(unenrollBody, "![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-manual-migration-1024x500.png)")
	default:
		// ADE migration, macOS > 14
		body = fmt.Sprintf(unenrollBody, "![Image showing MDM migration notification](https://fleetdm.com/images/permanent/mdm-ade-migration-1024x500.png)")
	}

	return m.render(body,
		"--button1text", "Start",
		"--button1disabled",
		"--quitkey", "x",
		"--height", "669",
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
func (m *swiftDialogMDMMigrator) waitForUnenrollment(isADEMigration bool) error {
	maxRetries := int(mdmUnenrollmentTotalWaitTime.Seconds() / m.unenrollmentRetryInterval.Seconds())
	checkFileFn := m.testEnrollmentCheckFileFn
	if checkFileFn == nil {
		checkFileFn = func() (bool, error) {
			return file.Exists(mdmEnrollmentFile)
		}
	}
	checkStatusFn := m.testEnrollmentCheckStatusFn
	if checkStatusFn == nil {
		checkStatusFn = profiles.IsEnrolledInMDM
	}
	return retry.Do(func() error {
		var unenrolled bool

		if isADEMigration {
			fileExists, fileErr := checkFileFn()
			switch {
			case fileErr != nil:
				log.Error().Err(fileErr).Msg("checking for existence of cloudConfigProfileInstalled in migration modal")
			case fileExists:
				log.Info().Msg("checking for existence of cloudConfigProfileInstalled in migration modal: found")
			default:
				log.Info().Msg("checking for existence of cloudConfigProfileInstalled in migration modal: not found")
				unenrolled = true
			}
		}

		statusEnrolled, serverURL, statusErr := checkStatusFn()
		if statusErr != nil { //nolint:gocritic // ignore ifElseChain
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
	log.Debug().Msg("checking current enrollment status")
	isCurrentlyManuallyEnrolled, err := profiles.IsManuallyEnrolledInMDM()
	if err != nil {
		return err
	}

	// Check what kind of migration was in progress, if any.
	previousMigrationType, err := m.mrw.GetMigrationType()
	if err != nil {
		log.Error().Err(err).Msg("getting migration type")
		return fmt.Errorf("getting migration type: %w", err)
	}

	isManualMigration := isCurrentlyManuallyEnrolled || previousMigrationType == constant.MDMMigrationTypeManual
	isADEMigration := previousMigrationType == constant.MDMMigrationTypeADE

	log.Debug().Bool("isManualMigration", isManualMigration).Bool("isADEMigration", isADEMigration).Bool("isCurrentlyManuallyEnrolled", isCurrentlyManuallyEnrolled).Str("previousMigrationType", previousMigrationType).Msg("props after assigning")

	vers, err := m.getMacOSMajorVersion()
	if err != nil {
		// log error for debugging and continue with default template
		log.Error().Err(err).Msg("getting macOS major version failed: using default migration template")
	}

	isPreSonoma := vers < constant.SonomaMajorVersion

	message, flags, err := m.getMessageAndFlags(vers, isManualMigration)
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

		if previousMigrationType == constant.MDMMigrationTypeADE {
			// Do nothing; the Remote Management modal will be launched by Orbit every minute.
			return nil
		}

		if previousMigrationType == constant.MDMMigrationTypeManual || previousMigrationType == constant.MDMMigrationTypePreSonoma {
			// Launch the "My device" page.
			log.Info().Msg("showing instructions")

			if err := m.handler.ShowInstructions(); err != nil {
				return err
			}
			return nil
		}

		if !m.props.IsUnmanaged {
			// show the loading spinner
			m.renderLoadingSpinner(isPreSonoma, isCurrentlyManuallyEnrolled)

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
			if err := m.waitForUnenrollment(isADEMigration); err != nil {
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

			switch {
			case isPreSonoma:
				if err := m.mrw.SetMigrationFile(constant.MDMMigrationTypePreSonoma); err != nil {
					log.Error().Str("migration_type", constant.MDMMigrationTypeADE).Err(err).Msg("set migration file")
				}

				log.Info().Msg("showing instructions after pre-sonoma unenrollment")
				if err := m.handler.ShowInstructions(); err != nil {
					return err
				}

			case isManualMigration:
				if err := m.mrw.SetMigrationFile(constant.MDMMigrationTypeManual); err != nil {
					log.Error().Str("migration_type", constant.MDMMigrationTypeManual).Err(err).Msg("set migration file")
				}

				log.Info().Msg("showing instructions after manual unenrollment")
				if err := m.handler.ShowInstructions(); err != nil {
					return err
				}

				m.frequency = 3 * time.Minute

			default:
				if err := m.mrw.SetMigrationFile(constant.MDMMigrationTypeADE); err != nil {
					log.Error().Str("migration_type", constant.MDMMigrationTypeADE).Err(err).Msg("set migration file")
				}
			}

			// close the spinner
			// TODO: maybe it's better to use
			// https://github.com/bartreardon/swiftDialog/wiki/Updating-Dialog-with-new-content
			// instead? it uses a file as IPC
			m.baseDialog.Exit()
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

func (m *swiftDialogMDMMigrator) getMessageAndFlags(version int, isManualMigration bool) (*bytes.Buffer, []string, error) {
	tmpl := mdmADEMigrationTemplate
	if isManualMigration {
		tmpl = mdmManualMigrationTemplate
	}

	height := "669"
	if version != 0 && version < constant.SonomaMajorVersion {
		height = "440"
		tmpl = mdmMigrationTemplatePreSonoma
	}

	var message bytes.Buffer
	if err := tmpl.Execute(
		&message,
		m.props,
	); err != nil {
		return nil, nil, fmt.Errorf("executing migration template: %w", err)
	}

	flags := []string{
		// main button
		"--button1text", "Start",
		// secondary button
		"--button2text", "Later",
		"--height", height,
	}

	if !m.props.DisableTakeover {
		flags = append(flags,
			"--blurscreen",
			"--ontop",
		)
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

func (m *swiftDialogMDMMigrator) MigrationInProgress() (string, error) {
	return m.mrw.GetMigrationType()
}

func (m *swiftDialogMDMMigrator) MarkMigrationCompleted() error {
	// Reset this to the original frequency.
	m.frequency = 15 * time.Minute
	return m.mrw.RemoveFile()
}

type offlineWatcher struct {
	client          *service.DeviceClient
	swiftDialogPath string
	// swiftDialogCh is shared with the migrator and used to ensure only one dialog is open at a time
	swiftDialogCh chan struct{}
	fileWatcher   migration.FileWatcher
}

// StartMDMMigrationOfflineWatcher starts a watcher running on a 3-minute loop that checks if the
// device goes offline in the process of migrating to Fleet's MDM and offline. If so, it shows a
// dialog to prompt the user to connect to the internet.
func StartMDMMigrationOfflineWatcher(ctx context.Context, client *service.DeviceClient, swiftDialogPath string, swiftDialogCh chan struct{}, fileWatcher migration.FileWatcher) MDMOfflineWatcher {
	if cap(swiftDialogCh) != 1 {
		log.Fatal().Msg("swift dialog channel must have a buffer size of 1")
	}

	watcher := &offlineWatcher{
		client:          client,
		swiftDialogPath: swiftDialogPath,
		swiftDialogCh:   swiftDialogCh,
		fileWatcher:     fileWatcher,
	}

	// start loop with 3-minute interval to ping server and show dialog if offline
	go func() {
		ticker := time.NewTicker(constant.MDMMigrationOfflineWatcherInterval)
		defer ticker.Stop()

		log.Info().Msg("starting watcher loop")
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("stopping offline dialog loop")
				return
			case <-ticker.C:
				log.Debug().Msg("offline dialog, got tick")
				go watcher.ShowIfOffline(ctx)
			}
		}
	}()

	return watcher
}

// ShowIfOffline shows the offline dialog if the host is offline.
// It returns true if the host is offline, and false otherwise.
func (o *offlineWatcher) ShowIfOffline(ctx context.Context) bool {
	// try the dialog channel
	select {
	case o.swiftDialogCh <- struct{}{}:
		log.Debug().Msg("occupying dialog channel")
	default:
		log.Debug().Msg("dialog channel already occupied")
		return false
	}

	defer func() {
		// non-blocking release of dialog channel
		select {
		case <-o.swiftDialogCh:
			log.Debug().Msg("releasing dialog channel")
		default:
			// this shouldn't happen so log for debugging
			log.Debug().Msg("dialog channel already released")
		}
	}()

	if !o.isUnmanaged() || !o.isOffline() {
		return false
	}

	log.Info().Msg("showing offline dialog")
	if err := o.showSwiftDialogMDMMigrationOffline(ctx); err != nil {
		log.Error().Err(err).Msg("error showing offline dialog")
	} else {
		log.Info().Msg("done showing offline dialog")
	}

	return true
}

func (o *offlineWatcher) isUnmanaged() bool {
	mt, err := o.fileWatcher.GetMigrationType()
	if err != nil {
		log.Error().Err(err).Msg("getting migration type")
	}

	if mt == "" {
		log.Debug().Msg("offline dialog, no migration type found, do nothing")
		return false
	}

	log.Debug().Msgf("offline dialog, device is unmanaged, migration type %s", mt)

	return true
}

func (o *offlineWatcher) isOffline() bool {
	err := o.client.Ping()
	if err == nil {
		log.Debug().Msg("offline dialog, ping ok, device is online")
		return false
	}
	if !isOfflineError(err) {
		log.Error().Err(err).Msg("offline dialog, error pinging server does not contain dial tcp or no such host, assuming device is online")
		return false
	}
	log.Debug().Err(err).Msg("offline dialog, error pinging server, assuming device is offline")

	return true
}

func isOfflineError(err error) bool {
	if err == nil {
		return false
	}
	offlineMsgs := []string{"no such host", "dial tcp", "no route to host"}
	for _, msg := range offlineMsgs {
		if strings.Contains(err.Error(), msg) {
			return true
		}
	}

	//  //  NOTE: We're starting with basic string matching and planning to improve error matching
	//  // in future iterations. Here's some ideas for stuff to add in addition to strings.Contains:
	// 	if urlErr, ok := err.(*url.Error); ok {
	// 		log.Info().Msg("is url error")
	// 		if urlErr.Timeout() {
	// 			log.Info().Msg("is timeout")
	// 			return true
	// 		}
	// 		// Check for no such host error
	// 		if opErr, ok := urlErr.Err.(*net.OpError); ok {
	// 			log.Info().Msg("is net op error")
	// 			if dnsErr, ok := opErr.Err.(*net.DNSError); ok {
	// 				log.Info().Msg("is dns error")
	// 				if dnsErr.Err == "no such host" {
	// 					log.Info().Msg("is dns no such host")
	// 					return true
	// 				}
	// 			}
	// 		}
	// 	}

	return false
}

// ShowDialogMDMMigrationOffline displays the dialog every time is called
func (o *offlineWatcher) showSwiftDialogMDMMigrationOffline(ctx context.Context) error {
	props := MDMMigratorProps{
		DisableTakeover: true,
	}
	m := swiftDialogMDMMigrationOffline{
		baseDialog: newBaseDialog(o.swiftDialogPath),
		props:      props,
	}

	flags, err := m.getFlags()
	if err != nil {
		return fmt.Errorf("getting flags for offline dialog: %w", err)
	}

	exitCodeCh, errCh := m.render(flags...)

	select {
	case <-ctx.Done():
		log.Debug().Msg("dialog context canceled")
		m.baseDialog.Exit()
		return nil
	case err := <-errCh:
		return fmt.Errorf("showing offline dialog: %w", err)
	case <-exitCodeCh:
		// there's only one button, so we don't need to check the exit code
		log.Info().Msg("closing offline dialog")
		return nil
	}
}

type swiftDialogMDMMigrationOffline struct {
	*baseDialog
	props MDMMigratorProps
}

func (m *swiftDialogMDMMigrationOffline) render(flags ...string) (chan swiftDialogExitCode, chan error) {
	return m.baseDialog.render(flags...)
}

func (m *swiftDialogMDMMigrationOffline) getFlags() ([]string, error) {
	tmpl := mdmMigrationTemplateOffline
	var message bytes.Buffer
	if err := tmpl.Execute(
		&message,
		nil,
	); err != nil {
		return nil, fmt.Errorf("executing migration template: %w", err)
	}

	// disable the built-in title and icon so we have full control over content
	title := "none"
	icon := "none"

	flags := []string{
		"--height", "124",
		"--alignment", "center",
		"--title", title,
		"--icon", icon,
		// modal content
		"--message", message.String(),
		"--messagefont", "size=16",
		// main button
		"--button1text", "Close",
	}

	if !m.props.DisableTakeover {
		flags = append(flags,
			"--blurscreen",
			"--ontop",
		)
	}

	return flags, nil
}
