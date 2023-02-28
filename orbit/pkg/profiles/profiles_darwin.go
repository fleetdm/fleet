//go:build darwin

package profiles

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/groob/plist"
)

type profileItem struct {
	PayloadContent fleet.MDMAppleFleetdConfig
	PayloadType    string
}

type profilePayload struct {
	ProfileIdentifier string
	ProfileItems      []profileItem
}

type profilesOutput struct {
	ComputerLevel []profilePayload `plist:"_computerlevel"`
}

// GetFleetdConfig searches and parses a device level configuration profile
// with Fleet's payload identifier.
func GetFleetdConfig() (*fleet.MDMAppleFleetdConfig, error) {
	p, err := getProfile(apple_mdm.FleetdConfigPayloadIdentifier)
	if err != nil {
		return nil, err
	}

	return &p.ProfileItems[0].PayloadContent, nil
}

func getProfile(identifier string) (*profilePayload, error) {
	outBuf, err := execProfileCmd()
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	var profiles profilesOutput
	if err := plist.Unmarshal(outBuf.Bytes(), &profiles); err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	for _, profile := range profiles.ComputerLevel {
		if profile.ProfileIdentifier == identifier {
			return &profile, nil
		}
	}

	return nil, ErrNotFound
}

// execProfileCmd is declared as a variable so it can be overwritten by tests.
var execProfileCmd = func() (*bytes.Buffer, error) {
	var outBuf bytes.Buffer
	cmd := exec.Command("/usr/bin/profiles", "list", "-o", "stdout-xml")
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &outBuf, nil
}
