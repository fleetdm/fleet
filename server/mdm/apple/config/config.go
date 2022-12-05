// Package config has utilities to verify Apple MDM related configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"

	configpkg "github.com/fleetdm/fleet/v4/server/config"
	nanodep_client "github.com/micromdm/nanodep/client"
)

// VerifyDEP verifies the Apple MDM configuration.
func VerifyDEP(config configpkg.MDMAppleConfig) error {
	if err := verifyDEPConfig(config); err != nil {
		return fmt.Errorf("dep: %w", err)
	}
	return nil
}

func verifyDEPConfig(config configpkg.MDMAppleConfig) error {
	token := []byte(config.DEP.Token)
	if len(token) == 0 {
		return errors.New("missing MDM DEP token")
	}
	if err := json.Unmarshal(token, &nanodep_client.OAuth1Tokens{}); err != nil {
		return fmt.Errorf("parse DEP token: %w", err)
	}
	return nil
}
