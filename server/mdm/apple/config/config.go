// Package config has utilities to verify Apple MDM related configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"

	configpkg "github.com/fleetdm/fleet/v4/server/config"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanomdm/cryptoutil"
	"golang.org/x/crypto/ssh"
)

// Verify verifies the Apple MDM configuration.
func Verify(config configpkg.MDMAppleConfig) error {
	if err := verifySCEPConfig(config); err != nil {
		return fmt.Errorf("scep: %w", err)
	}
	if err := verifyMDMConfig(config); err != nil {
		return fmt.Errorf("mdm: %w", err)
	}
	if err := verifyDEPConfig(config); err != nil {
		return fmt.Errorf("dep: %w", err)
	}
	return nil
}

func verifySCEPConfig(config configpkg.MDMAppleConfig) error {
	pemCert := []byte(config.SCEP.CA.PEMCert)
	if len(pemCert) == 0 {
		return errors.New("missing pem certificate")
	}
	if _, err := cryptoutil.DecodePEMCertificate(pemCert); err != nil {
		return fmt.Errorf("parse pem certificate: %w", err)
	}
	pemKey := []byte(config.SCEP.CA.PEMKey)
	if len(pemKey) == 0 {
		return errors.New("missing private key")
	}
	if _, err := ssh.ParseRawPrivateKey(pemKey); err != nil {
		return fmt.Errorf("parse MDM push PEM private key: %w", err)
	}
	return nil
}

func verifyMDMConfig(config configpkg.MDMAppleConfig) error {
	pushPEMCert := []byte(config.MDM.PushCert.PEMCert)
	if len(pushPEMCert) == 0 {
		return errors.New("missing PEM certificate")
	}
	if _, err := cryptoutil.DecodePEMCertificate(pushPEMCert); err != nil {
		return fmt.Errorf("parse PEM certificate: %w", err)
	}
	_, err := cryptoutil.TopicFromPEMCert(pushPEMCert)
	if err != nil {
		return fmt.Errorf("extract topic from push PEM cert: %w", err)
	}
	pemKey := []byte(config.MDM.PushCert.PEMKey)
	if len(pemKey) == 0 {
		return errors.New("missing PEM private key")
	}
	if _, err := ssh.ParseRawPrivateKey(pemKey); err != nil {
		return fmt.Errorf("parse PEM private key: %w", err)
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
