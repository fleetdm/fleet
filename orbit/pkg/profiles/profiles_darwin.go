//go:build darwin

package profiles

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/groob/plist"
)

type profileItem[T any] struct {
	PayloadContent    T
	PayloadType       string
	PayloadIdentifier string
}

type profilePayload[T any] struct {
	ProfileItems []profileItem[T]
}

type profilesOutput[T any] struct {
	ComputerLevel []profilePayload[T] `plist:"_computerlevel"`
}

// GetFleetdConfig searches and parses a device level configuration profile
// with Fleet's payload identifier.
func GetFleetdConfig() (*fleet.MDMAppleFleetdConfig, error) {
	pc, err := getProfilePayloadContent[fleet.MDMAppleFleetdConfig](mobileconfig.FleetdConfigPayloadIdentifier)
	if err != nil {
		if err == ErrNotFound {
			return &fleet.MDMAppleFleetdConfig{}, nil
		}

		return nil, err
	}

	return pc, nil
}

func GetCustomEnrollmentProfileEndUserEmail() (string, error) {
	pc, err := getProfilePayloadContent[fleet.MDMCustomEnrollmentProfileItem](mobileconfig.FleetEnrollmentPayloadIdentifier)
	if err != nil {
		return "", err
	}
	if pc == nil || pc.EndUserEmail == "" {
		return "", ErrNotFound
	}
	return pc.EndUserEmail, nil
}

func getProfilePayloadContent[T any](identifier string) (*T, error) {
	outBuf, err := execProfileCmd()
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	var profiles profilesOutput[T]
	if err := plist.Unmarshal(outBuf.Bytes(), &profiles); err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	for _, profile := range profiles.ComputerLevel {
		for _, item := range profile.ProfileItems {
			if item.PayloadIdentifier == identifier {
				return &item.PayloadContent, nil
			}
		}
	}

	return nil, ErrNotFound
}

// execProfileCmd is declared as a variable so it can be overwritten by tests.
var execProfileCmd = func() (*bytes.Buffer, error) {
	var outBuf bytes.Buffer
	// TODO: check if there is a reason to prefer -L over -C in some cases
	cmd := exec.Command("/usr/bin/profiles", "-C", "-o", "stdout-xml")
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &outBuf, nil
}

// IsEnrolledInMDM runs the `profiles` command to get the current MDM
// enrollment information and reports if the host is enrolled, and the URL of
// the MDM server (if enrolled)
func IsEnrolledInMDM() (bool, string, error) {
	out, err := getMDMInfoFromProfilesCmd()
	if err != nil {
		return false, "", fmt.Errorf("calling /usr/bin/profiles: %w", err)
	}

	// The output of the command is in the form:
	//
	// ```
	// Enrolled via DEP: No
	// MDM enrollment: Yes (User Approved)
	// MDM server: https://test.example.com/mdm/apple/mdm
	// ```
	//
	// If the host is not enrolled into an MDM, the last line is ommitted,
	// so we need to check that:
	//
	// 1. We've got three rows
	// 2. The last row matches our server URL
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(lines) < 3 {
		return false, "", nil
	}

	parts := bytes.SplitN(lines[2], []byte(":"), 2)
	if len(parts) < 2 {
		return false, "", fmt.Errorf("splitting profiles output to get MDM server URL: %w", err)
	}

	enrollmentURL := string(bytes.TrimSpace(parts[1]))

	return true, enrollmentURL, nil
}

func IsManuallyEnrolledInMDM() (bool, error) {
	out, err := getMDMInfoFromProfilesCmd()
	if err != nil {
		return false, fmt.Errorf("calling /usr/bin/profiles: %w", err)
	}

	// The output of the command is in the form:
	//
	// ```
	// Enrolled via DEP: No
	// MDM enrollment: Yes (User Approved)
	// MDM server: https://test.example.com/mdm/apple/mdm
	// ```
	//
	// If the host is not enrolled into an MDM, the last line is ommitted,
	// so we need to check that:
	//
	// 1. We've got three rows
	// 2. Whether the first line contains "Yes" or "No"
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(lines) < 3 {
		return false, nil
	}

	if strings.Contains(string(lines[0]), "Yes") {
		return false, nil
	}

	return true, nil
}

// getMDMInfoFromProfilesCmd is declared as a variable so it can be overwritten by tests.
var getMDMInfoFromProfilesCmd = func() ([]byte, error) {
	cmd := exec.Command("/usr/bin/profiles", "status", "-type", "enrollment")
	return cmd.Output()
}

// CheckAssignedEnrollmentProfile runs the `profiles show -type enrollment` command to get the assigned
// MDM enrollment profile and reports if the hostname of the MDM server
// in the assigned profile the device matches the hostname of the provided URL.
func CheckAssignedEnrollmentProfile(expectedURL string) error {
	expected, err := url.Parse(expectedURL)
	if err != nil {
		return fmt.Errorf("parsing expected URL: %w", err)
	}

	out, err := showEnrollmentProfileCmd()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("show enrollment profile command: %w: %s", err, exitErr.Stderr)
		}
		return fmt.Errorf("show enrollment profile command: %w", err)
	}

	// If an enrollment profile is assigned, the output of the command is in the form:
	//
	// ```
	// Device Enrollment configuration:
	// {
	//     AllowPairing = 1;
	//     AutoAdvanceSetup = 0;
	//     AwaitDeviceConfigured = 0;
	//     ConfigurationURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	//     ConfigurationWebURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	//     ...
	// }
	// ```
	//
	// If the host is not enrolled into an MDM, the output of the command is in the form:
	//
	// ```
	// Device Enrollment configuration:
	// (null)
	// ```
	//
	// We will check that the output is at least 2 lines and contains the expected URL

	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(lines) < 2 {
		return fmt.Errorf("parsing profiles output: expected at least 2 lines but got %d", len(lines))
	}
	if !bytes.Equal(lines[0], []byte("Device Enrollment configuration:")) {
		return errors.New("parsing profiles output: does not match expected device enrollment configuration format")
	}
	if bytes.Equal(lines[1], []byte("(null)")) {
		return errors.New("parsing profiles output: received null device enrollment configuration")
	}

	var assignedURL string
	for _, line := range lines {
		// Note the output may contain both ConfigurationURL and ConfigurationWebURL but we check only
		// the latter for backwards compatibility.
		// See https://github.com/fleetdm/fleet/blob/963b2438537de14e7e16f1f18857ed8a66d51bfc/server/mdm/apple/apple_mdm.go#L195
		v, ok := parseEnrollmentProfileValue(line, "ConfigurationWebURL")
		if ok {
			assignedURL = v
			break
		}
	}

	if assignedURL == "" {
		return errors.New("parsing profiles output: missing or empty configuration web url")
	}

	assigned, err := url.Parse(assignedURL)
	if err != nil {
		return fmt.Errorf("parsing profiles output: unable to parse configuration web url: %w", err)
	}

	if !strings.EqualFold(assigned.Hostname(), expected.Hostname()) {
		return fmt.Errorf(`matching configuration web url: expected '%s' but found '%s'`, expected.Hostname(), assigned.Hostname())
	}

	return nil
}

func parseEnrollmentProfileValue(line []byte, key string) (string, bool) {
	// Output lines of `profiles show -type enrollment` take the form below:
	// ```
	// Device Enrollment configuration:
	// {
	//     AllowPairing = 1;
	//     AutoAdvanceSetup = 0;
	//     AwaitDeviceConfigured = 0;
	//     ConfigurationURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	//     ConfigurationWebURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	//     ...
	// }

	// We are interested in the key-value pairs, which feature the separator " = ".
	// Note that we want to include the spaces around the equals sign to avoid further splitting
	// values, e.g., the url value may also contain an equals sign in the query string.
	parts := bytes.SplitN(line, []byte(" = "), 3)
	if len(parts) != 2 {
		return "", false
	}

	k := strings.TrimSpace(string(parts[0]))
	if k == key {
		// The value may be quoted and may contain a trailing semicolon. Remove both.
		v := strings.TrimSpace(string(parts[1]))
		v = strings.TrimSuffix(v, `;`)
		v = strings.Trim(v, `"`)
		return v, true
	}

	return "", false
}

// showEnrollmentProfileCmd is declared as a variable so it can be overwritten by tests.
var showEnrollmentProfileCmd = func() ([]byte, error) {
	cmd := exec.Command("sh", "-c", `launchctl asuser $(id -u $(stat -f "%u" /dev/console)) profiles show -type enrollment`)
	return cmd.Output()
}
