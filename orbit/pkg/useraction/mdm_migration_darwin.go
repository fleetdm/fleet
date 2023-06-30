//go:build darwin

package useraction

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"text/template"
	"time"

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

var mdmMigrationTemplate = template.Must(template.New("mdmMigrationTemplate").Parse(`
## Migrate to Fleet

To begin, click "Start." Your default browser will open your My Device page.

{{ if .IsUnmanaged }}You {{ else }} Once you start, you {{ end -}} will see this dialog every 15 minutes until you click "Turn on MDM" and complete the instructions.` +
	"\n\n![Image showing the Fleet UI](https://fleetdm.com/images/permanent/mdm-migration-screenshot-768x180-2x.png)\n\n",
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

func NewMDMMigrator(path string, frequency time.Duration, handler MDMMigratorHandler) MDMMigrator {
	return &swiftDialogMDMMigrator{
		handler:    handler,
		baseDialog: newBaseDialog(path),
		frequency:  frequency,
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
	showMu    sync.Mutex
	lastShown time.Time

	// ensures only one dialog is open at a given interval
	intervalMu sync.Mutex
}

func (m *swiftDialogMDMMigrator) render(message string, flags ...string) (chan swiftDialogExitCode, chan error) {
	icon := m.props.OrgInfo.OrgLogoURL
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
		"--ontop",
	}, flags...)

	return m.baseDialog.render(flags...)
}

func (m *swiftDialogMDMMigrator) renderLoadingSpinner() (chan swiftDialogExitCode, chan error) {
	return m.render("## Migrate to Fleet\n\nCommunicating with MDM server...",
		"--button1text", "Start",
		"--button1disabled",
		"--quitkey", "x",
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

	return m.render(errorMessage.String(), "--button1text", "Close")
}

func (m *swiftDialogMDMMigrator) renderMigration() error {

	var message bytes.Buffer
	if err := mdmMigrationTemplate.Execute(
		&message,
		m.props,
	); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	flags := []string{
		// main button
		"--button1text", "Start",
		// secondary button
		"--button2text", "Later",
		"--blurscreen", "--ontop", "--height", "500",
	}

	if m.props.OrgInfo.ContactURL != "" {
		flags = append(flags,
			// info button
			"--infobuttontext", "Unsure? Contact IT",
			"--infobuttonaction", m.props.OrgInfo.ContactURL,
			"--quitoninfo",
		)
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
					return nil
				case err := <-errDialogErrChan:
					return fmt.Errorf("rendering errror dialog: %w", err)
				}
			}

			log.Info().Msg("webhook sent, closing spinner")

			// close the spinner
			// TODO: maybe it's better to use
			// https://github.com/bartreardon/swiftDialog/wiki/Updating-Dialog-with-new-content
			// instead? it uses a file as IPC
			m.baseDialog.Exit()
		}

		log.Info().Msg("showing instructions")
		m.handler.ShowInstructions()
	}

	return nil
}

func (m *swiftDialogMDMMigrator) Show() error {
	if m.showMu.TryLock() {
		defer m.showMu.Unlock()

		if err := m.renderMigration(); err != nil {
			return fmt.Errorf("show: %w", err)
		}
	}

	return nil
}

func (m *swiftDialogMDMMigrator) ShowInterval() error {
	if m.intervalMu.TryLock() {
		defer m.intervalMu.Unlock()
		if time.Since(m.lastShown) > m.frequency {
			if err := m.Show(); err != nil {
				return fmt.Errorf("show interval: %w", err)
			}
			m.lastShown = time.Now()
		}
	}

	return nil
}

func (m *swiftDialogMDMMigrator) SetProps(props MDMMigratorProps) {
	m.props = props
}
