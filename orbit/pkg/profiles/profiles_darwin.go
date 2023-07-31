//go:build darwin

package profiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
)

// GetFleetdConfig reads a system level setting set with Fleet's payload identifier.
func GetFleetdConfig() (*fleet.MDMAppleFleetdConfig, error) {
	readFleetdConfigAppleScript := fmt.Sprintf(`
           const config = $.NSUserDefaults.alloc.initWithSuiteName("%s");
           const enrollSecret = config.objectForKey("EnrollSecret");
           const fleetURL = config.objectForKey("FleetURL");
           JSON.stringify({
             EnrollSecret: ObjC.deepUnwrap(enrollSecret),
             FleetURL: ObjC.deepUnwrap(fleetURL),
           });
         `, mobileconfig.FleetdConfigPayloadIdentifier)

	outBuf, err := execScript(readFleetdConfigAppleScript)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	var cfg fleet.MDMAppleFleetdConfig
	if err = json.Unmarshal(outBuf.Bytes(), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling configuration: %w", err)
	}

	if cfg.EnrollSecret == "" || cfg.FleetURL == "" {
		return nil, ErrNotFound
	}

	return &cfg, err
}

// execScript is declared as a variable so it can be overwritten by tests.
var execScript = func(script string) (*bytes.Buffer, error) {
	var outBuf bytes.Buffer
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", script)
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &outBuf, nil
}

// IsEnrolledIntoMatchingURL runs the `profiles` command to get the current MDM
// enrollment information and reports if the hostname of the MDM server
// supervising the device matches the hostname of the provided URL.
func IsEnrolledIntoMatchingURL(serverURL string) (bool, error) {
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
	// 2. The last row matches our server URL
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(lines) < 3 {
		return false, nil
	}

	parts := bytes.SplitN(lines[2], []byte(":"), 2)
	if len(parts) < 2 {
		return false, fmt.Errorf("splitting profiles output to get MDM server URL: %w", err)
	}

	u, err := url.Parse(string(bytes.TrimSpace(parts[1])))
	if err != nil {
		return false, fmt.Errorf("parsing URL from profiles command: %w", err)
	}

	fu, err := url.Parse(serverURL)
	if err != nil {
		return false, fmt.Errorf("parsing provided Fleet URL: %w", err)
	}

	return u.Hostname() == fu.Hostname(), nil
}

// getMDMInfoFromProfilesCmd is declared as a variable so it can be overwritten by tests.
var getMDMInfoFromProfilesCmd = func() ([]byte, error) {
	cmd := exec.Command("/usr/bin/profiles", "status", "-type", "enrollment")
	return cmd.Output()
}
