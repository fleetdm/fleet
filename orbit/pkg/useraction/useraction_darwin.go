//go:build darwin

package useraction

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/ptr"
)

const defaultDialogTimeout = 30 // in seconds

var (
	errInvalidPassword = errors.New("invalid password")
	errTimeout         = errors.New("timeout without user response")
	errCanceled        = errors.New("canceled by user")
)

type appleScriptDialogOutput struct {
	ButtonReturned string `json:"buttonReturned"`
	TextReturned   string `json:"textReturned"`
	GaveUp         bool   `json:"gaveUp"`
}

type appleScriptDialogOptions struct {
	Message string   `json:"message,omitempty"`
	Answer  *string  `json:"defaultAnswer,omitempty"`
	Hidden  bool     `json:"hiddenAnswer,omitempty"`
	Title   *string  `json:"withTitle,omitempty"`
	Icon    string   `json:"withIcon,omitempty"`
	Buttons []string `json:"buttons,omitempty"`
	Default int      `json:"defaultButton,omitempty"`
	Timeout int      `json:"givingUpAfter,omitempty"`
}

type dialogOptions struct {
	Options appleScriptDialogOptions
	Text    string
}

func RotateDiskEncryptionKey(maxRetries int) error {
	var u, p string
	var err error

	u, p, err = askUserCredentials()
	if err != nil {
		return fmt.Errorf("asking user credentials: %w", err)
	}

	for i := 1; i <= maxRetries; i++ {
		if err = rotateFileVaultKey(u, p); err == nil {
			break
		}

		if errors.Is(err, errInvalidPassword) {
			if p, err = wrongPasswordMessage(u); err != nil {
				if errors.Is(err, errTimeout) || errors.Is(err, errCanceled) {
					return err
				}

				if err := errorMessage(fmt.Sprintf("error: %s", err)); err != nil {
					return fmt.Errorf("showing error message: %w", err)
				}

				return fmt.Errorf("re-asking for password: %w", err)
			}
		}

		if i >= maxRetries {
			if err := errorMessage(fmt.Sprintf("Failed to generate a new disk encryption key after %d tries.", maxRetries)); err != nil {
				return fmt.Errorf("showing error message after max retries exceeded: %w", err)
			}
			return fmt.Errorf("failed after %d retries", maxRetries)
		}
	}

	if err := showSuccessMsg(); err != nil {
		return fmt.Errorf("showing success message: %w", err)
	}

	return nil
}

var dialogScript = template.Must(template.New("").Funcs(template.FuncMap{"json": func(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}}).Parse(`
var app = Application.currentApplication()
app.includeStandardAdditions = true
var opts = {{json .Options}}
if (opts.withIcon) {
  opts.withIcon = Path(opts.withIcon)
}
app.displayDialog({{json .Text}}, opts)`))

// displayDialog builds a new dialog from the template with the provided
// options and runs `osascript` to display it to the user.
//
// It makes sure the dialog doesn't stay open forever by enforcing a timeout.
func displayDialog(opts dialogOptions) (string, error) {
	// ensure all dialogs have a timeout
	if opts.Options.Timeout <= 0 {
		opts.Options.Timeout = defaultDialogTimeout
	}

	var script bytes.Buffer
	if err := dialogScript.Execute(&script, opts); err != nil {
		return "", err
	}

	// -l to choose the language
	// -s to make the output machine-readable (JSON)
	cmd := exec.Command("osascript", "-l", "JavaScript", "-s", "s")
	cmd.Stdin = &script
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			// TODO: the script will fail with this string and an error
			// number (-128) if the user presses on "cancel". I would
			// have expected to get a normal return value and the
			// "cancel" button in dialogOutput.ButtonReturned. Is there
			// a better way to capture this?
			if bytes.Contains(ee.Stderr, []byte("User canceled")) {
				return "", errCanceled
			}
			return "", fmt.Errorf("osascript failed: %w: %s", err, string(ee.Stderr))
		}
		return "", err
	}

	var d appleScriptDialogOutput
	if err := json.Unmarshal(out, &d); err != nil {
		return "", fmt.Errorf("unmarshal osascript output: %w", err)
	}

	if d.GaveUp {
		return "", errTimeout
	}

	return d.TextReturned, nil
}

// askUserCredentials fetches the username/password combination for the current logged in user.
func askUserCredentials() (string, string, error) {
	rawUser, err := exec.Command("/usr/bin/stat", "-f", "%Su", "/dev/console").Output()
	if err != nil {
		return "", "", err
	}
	u := strings.TrimSpace(string(rawUser))
	p, err := displayDialog(dialogOptions{
		Text: fmt.Sprintf("To generate a new disk encryption key, enter login password for '%s'", u),
		Options: appleScriptDialogOptions{
			Title:   ptr.String("Reset disk encryption key"),
			Icon:    "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/FileVaultIcon.icns",
			Buttons: []string{"Cancel", "Ok"},
			Default: 2,
			Answer:  ptr.String(""),
			Hidden:  true,
		},
	})
	return u, p, err
}

// wrongPasswordMessage shows a message when the entered password is wrong, and
// allows the user to enter the password again.
func wrongPasswordMessage(u string) (string, error) {
	p, err := displayDialog(dialogOptions{
		Text: fmt.Sprintf("The password entered for user '%s' was invalid, please try again.", u),
		Options: appleScriptDialogOptions{
			Title:   ptr.String("Reset disk encryption key"),
			Icon:    "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/AlertStopIcon.icns",
			Buttons: []string{"Cancel", "Ok"},
			Default: 2,
			Answer:  ptr.String(""),
			Hidden:  true,
		},
	})
	return p, err
}

// errorMessage displays an error string to the user.
func errorMessage(msg string) error {
	_, err := displayDialog(dialogOptions{
		Text: msg,
		Options: appleScriptDialogOptions{
			Title:   ptr.String("Reset disk encryption key"),
			Icon:    "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/AlertStopIcon.icns",
			Buttons: []string{"Ok"},
			Default: 1,
		},
	})
	return err
}

// successMessage displays success string to the user
func showSuccessMsg() error {
	_, err := displayDialog(dialogOptions{
		Text: "Success! Your disk encryption key was reset.",
		Options: appleScriptDialogOptions{
			Title:   ptr.String("Reset disk encryption key"),
			Buttons: []string{"Ok"},
			Default: 1,
		},
	})
	return err
}

// rotateFileVaultKey runs the necessary commands to rotate/generate the user's
// FileVault disk encryption key.
func rotateFileVaultKey(u, p string) error {
	if p == "" {
		return errInvalidPassword
	}

	script := fmt.Sprintf(`
		log_user 0
		spawn fdesetup changerecovery -personal
		expect "Enter the user name:"
		send {%s}   
		send \r
		expect "Enter a password for '/', or the recovery key:"
		send {%s}   
		send \r
		log_user 1
		expect eof`, u, p)
	out, err := exec.Command("expect", "-c", script).Output()
	if err != nil {
		return fmt.Errorf("osascript failed: %w", err)
	}

	// wrong user/pass returns a 0 exit code but with an error message
	if bytes.Contains(out, []byte("User could not be authenticated")) {
		return errInvalidPassword
	}
	return nil
}
