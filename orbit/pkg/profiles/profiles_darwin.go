//go:build darwin

package profiles

import (
	"bytes"
	"encoding/json"
	"fmt"
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
