package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/spf13/cobra"
	"go.mozilla.org/pkcs7"
	"golang.org/x/crypto/ssh"
)

func createMDMAppleSetupCmd(configManager config.Manager, dev *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Setup Apple's MDM in Fleet",
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if *dev {
				applyDevFlags(&config)
			}

			err := verifyMDMAppleConfig(config)
			if err != nil {
				initFatal(err, "verifying MDM Apple config")
			}

			mds, err := mysql.New(config.Mysql, clock.C,
				mysql.WithMDMApple(config.MDMApple.Enable),
			)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := mds.MigrationMDMAppleStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			migrationStatusCheck(status, false, *dev, config.Mysql.DatabaseMDMApple)

			// (1) SCEP Setup

			scepCAKeyPassphrase := []byte(config.MDMApple.SCEP.CA.Passphrase)
			mdmAppleSCEPDepot, err := mds.NewMDMAppleSCEPDepot()
			if err != nil {
				initFatal(err, "initializing SCEP depot")
			}
			_, _, err = mdmAppleSCEPDepot.CreateCA(
				scepCAKeyPassphrase,
				int(config.MDMApple.SCEP.CA.ValidityYears),
				config.MDMApple.SCEP.CA.CN,
				config.MDMApple.SCEP.CA.Organization,
				config.MDMApple.SCEP.CA.OrganizationalUnit,
				config.MDMApple.SCEP.CA.Country,
			)
			if err != nil {
				initFatal(err, "creating SCEP CA")
			}

			// (2) MDM core setup

			mdmStorage, err := mds.NewMDMAppleMDMStorage()
			if err != nil {
				initFatal(err, "initializing mdm apple MySQL storage")
			}
			err = mdmStorage.StorePushCert(cmd.Context(), config.MDMApple.MDM.PushCert.PEMCert, config.MDMApple.MDM.PushCert.PEMKey)
			if err != nil {
				initFatal(err, "storing APNS push certificate")
			}
			topic, err := cryptoutil.TopicFromPEMCert(config.MDMApple.MDM.PushCert.PEMCert)
			if err != nil {
				initFatal(err, "extracting topic from push PEM cert")
			}
			err = mdmStorage.SetCurrentTopic(cmd.Context(), topic)
			if err != nil {
				initFatal(err, "setting current push PEM topic")
			}

			// (3) MDM DEP setup (stage 1 - keypair generation)

			mdmAppleDEPStorage, err := mds.NewMDMAppleDEPStorage()
			if err != nil {
				initFatal(err, "initializing DEP storage")
			}

			// TODO(lucas): Check validity days default value.
			const (
				cn           = "fleet"
				validityDays = 1
			)
			key, cert, err := tokenpki.SelfSignedRSAKeypair(cn, validityDays)
			if err != nil {
				initFatal(err, "generating DEP keypair")
			}
			pemCert := tokenpki.PEMCertificate(cert.Raw)
			pemKey := tokenpki.PEMRSAPrivateKey(key)

			err = mdmAppleDEPStorage.StoreTokenPKI(cmd.Context(), apple.DEPName, pemCert, pemKey)
			if err != nil {
				initFatal(err, "storing DEP keypair")
			}
			pemCertFile := "dep_public_key.pem"
			if err := os.WriteFile(pemCertFile, pemCert, 0o600); err != nil {
				initFatal(err, "writing deb_public_key.pem file")
			}
			fmt.Printf("Upload the public key %q file to the MDM server on Apple Business Manager.\n", pemCertFile)
			fmt.Println("Apple MDM setup completed.")
		},
	}
}

func createMDMAppleDEPPushTokenCmd(configManager config.Manager, dev *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "dep-auth-token",
		Short: "Set Apple's MDM DEP auth token in Fleet",
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if *dev {
				applyDevFlags(&config)
			}

			err := verifyMDMAppleDEPConfig(config)
			if err != nil {
				initFatal(err, "verifying MDM Apple DEP config")
			}

			mds, err := mysql.New(config.Mysql, clock.C,
				mysql.WithMDMApple(config.MDMApple.Enable),
				mysql.WithMultiStatements(config.MDMApple.Enable),
			)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := mds.MigrationMDMAppleStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			migrationStatusCheck(status, false, *dev, config.Mysql.DatabaseMDMApple)

			depStorage, err := mds.NewMDMAppleDEPStorage()
			if err != nil {
				initFatal(err, "initializing Apple MDM DEP storage")
			}
			depPEMCert, depPEMKey, err := depStorage.RetrieveTokenPKI(cmd.Context(), apple.DEPName)
			if err != nil {
				initFatal(err, "retrieving Apple MDM DEP keypair")
			}
			depCert, err := tokenpki.CertificateFromPEM(depPEMCert)
			if err != nil {
				initFatal(err, "parsing Apple MDM DEP PEM certificate")
			}
			depKey, err := tokenpki.RSAKeyFromPEM(depPEMKey)
			if err != nil {
				initFatal(err, "parsing Apple MDM DEP PEM certificate")
			}
			tokenJSON, err := tokenpki.DecryptTokenJSON(config.MDMApple.DEP.EncryptedAuthToken, depCert, depKey)
			if err != nil {
				initFatal(err, "decrypting Apple MDM DEP auth token")
			}
			tokens := new(client.OAuth1Tokens)
			if err = json.Unmarshal(tokenJSON, tokens); err != nil {
				initFatal(err, "parsing Apple MDM DEP auth token")
			}
			if !tokens.Valid() {
				initFatal(err, "invalid Apple MDM DEP auth token")
			}
			if err := depStorage.StoreAuthTokens(cmd.Context(), apple.DEPName, tokens); err != nil {
				initFatal(err, "storing Apple MDM DEP auth token")
			}

			// TODO(lucas): Check whether we want this here or not.
			//
			// Define a default DEP profile on Apple and set the returned ID for the assigner+syncer.
			httpClient := fleethttp.NewClient()
			depTransport := client.NewTransport(httpClient.Transport, httpClient, depStorage, nil)
			depClient := client.NewClient(fleethttp.NewClient(), depTransport)

			// TODO(lucas): Define this in a common location to be used by fleetctl in the future.
			type depProfileRequestFields struct {
				ProfileName           string   `json:"profile_name"`
				URL                   string   `json:"url"`
				AllowPairing          bool     `json:"allow_pairing"`
				AutoAdvanceSetup      bool     `json:"auto_advance_setup"`
				AwaitDeviceConfigured bool     `json:"await_device_configured"`
				ConfigurationWebURL   string   `json:"configuration_web_url"`
				IsSupervised          bool     `json:"is_supervised"`
				IsMultiUser           bool     `json:"is_multi_user"`
				IsMandatory           bool     `json:"is_mandatory"`
				IsMDMRemovable        bool     `json:"is_mdm_removable"`
				AnchorCerts           []string `json:"anchor_certs"`
				SupervisingHostCerts  []string `json:"supervising_host_certs"`
				SkipSetupItems        []string `json:"skip_setup_items"`
				Devices               []string `json:"devices"`
			}
			// TODO(lucas): Using the following default values. Pause and ponder.
			depProfile := depProfileRequestFields{
				ProfileName:           "Fleet Device Management Inc.",
				URL:                   "https://" + config.MDMApple.DEP.ServerURL + "/mdm/apple/api/enroll",
				AllowPairing:          true,
				AutoAdvanceSetup:      false,
				AwaitDeviceConfigured: false,
				ConfigurationWebURL:   "https://" + config.MDMApple.DEP.ServerURL + "/mdm/apple/api/enroll",
				IsSupervised:          false,
				IsMultiUser:           false,
				IsMandatory:           false,
				IsMDMRemovable:        true,
				AnchorCerts:           []string{},
				SupervisingHostCerts:  []string{},
				SkipSetupItems:        []string{"AppleID", "Android"},
				Devices:               []string{},
			}
			depProfileBody, err := json.Marshal(depProfile)
			if err != nil {
				initFatal(err, "serializing dep profile request JSON")
			}
			defineProfileRequest, err := client.NewRequestWithContext(
				cmd.Context(), apple.DEPName, depStorage,
				"POST", "/profile", bytes.NewReader(depProfileBody),
			)
			if err != nil {
				initFatal(err, "creating request to define default DEP profile")
			}
			defineProfileHTTPResponse, err := depClient.Do(defineProfileRequest)
			if err != nil {
				initFatal(err, "defining default DEP profile")
			}
			defer defineProfileHTTPResponse.Body.Close()
			if defineProfileHTTPResponse.StatusCode != http.StatusOK {
				initFatal(
					fmt.Errorf("defining default DEP profile: %s", defineProfileHTTPResponse.Status),
					"defining default DEP profile",
				)
			}
			defineProfileResponseBody, err := io.ReadAll(defineProfileHTTPResponse.Body)
			if err != nil {
				initFatal(err, "reading DEP profile response")
			}
			// TODO(lucas): Define this in a common location to be used by fleetctl in the future.
			type depProfileResponseFields struct {
				ProfileUUID string `json:"profile_uuid"`
			}
			defineProfileResponse := depProfileResponseFields{}
			if err := json.Unmarshal(defineProfileResponseBody, &defineProfileResponse); err != nil {
				initFatal(err, "parsing DEP profile response")
			}
			if err := depStorage.StoreAssignerProfile(
				cmd.Context(), apple.DEPName, defineProfileResponse.ProfileUUID,
			); err != nil {
				initFatal(err, "setting profile_uuid to assigner")
			}

			fmt.Println("Apple MDM DEP auth token successfully stored.")
		},
	}
}

func verifyMDMAppleConfig(config config.FleetConfig) error {
	if !config.MDMApple.Enable {
		return errors.New("MDM disabled")
	}
	if scepCAKeyPassphrase := []byte(config.MDMApple.SCEP.CA.Passphrase); len(scepCAKeyPassphrase) == 0 {
		return errors.New("missing passphrase for SCEP CA private key")
	}
	pushPEMCert := config.MDMApple.MDM.PushCert.PEMCert
	if len(pushPEMCert) == 0 {
		return errors.New("missing MDM push PEM certificate")
	}
	if _, err := cryptoutil.DecodePEMCertificate(pushPEMCert); err != nil {
		return fmt.Errorf("parse MDM push PEM certificate: %w", err)
	}
	_, err := cryptoutil.TopicFromPEMCert(pushPEMCert)
	if err != nil {
		return fmt.Errorf("extract topic from push PEM cert: %w", err)
	}
	pemKey := config.MDMApple.MDM.PushCert.PEMKey
	if len(pemKey) == 0 {
		return errors.New("missing MDM push PEM private key")
	}
	_, err = ssh.ParseRawPrivateKey(pemKey)
	if err != nil {
		return fmt.Errorf("parse MDM push PEM private key: %w", err)
	}
	return nil
}

func verifyMDMAppleDEPConfig(config config.FleetConfig) error {
	if !config.MDMApple.Enable {
		return errors.New("MDM disabled")
	}
	if len(config.MDMApple.DEP.EncryptedAuthToken) == 0 {
		return errors.New("missing MDM DEP encrypted token")
	}
	p7Bytes, err := tokenpki.UnwrapSMIME(config.MDMApple.DEP.EncryptedAuthToken)
	if err != nil {
		return fmt.Errorf("unwrap SMIME encrypted token: %w", err)
	}
	if _, err := pkcs7.Parse(p7Bytes); err != nil {
		return fmt.Errorf("parse PKCS.7 encrypted token: %w", err)
	}
	if config.MDMApple.DEP.ServerURL == "" {
		return errors.New("missing Fleet server address")
	}
	return nil
}
