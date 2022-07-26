package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func createPrepareCmd(configManager config.Manager) *cobra.Command {
	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Subcommands for initializing Fleet infrastructure",
		Long: `
Subcommands for initializing Fleet infrastructure

To setup Fleet infrastructure, use one of the available commands.
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	noPrompt := false
	// Whether to enable developer options
	dev := false

	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Given correct database configurations, prepare the databases for use",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if dev {
				applyDevFlags(&config)
				noPrompt = true
			}

			ds, err := mysql.New(config.Mysql, clock.C,
				mysql.WithMDMApple(config.MDMApple.Enable),
				mysql.WithMultiStatements(config.MDMApple.Enable),
			)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			prepareMigrationStatusCheck(status, noPrompt, dev, "fleet")

			if err := ds.MigrateTables(cmd.Context()); err != nil {
				initFatal(err, "migrating db schema")
			}

			if err := ds.MigrateData(cmd.Context()); err != nil {
				initFatal(err, "migrating builtin data")
			}

			if config.MDMApple.Enable {
				status, err := ds.MigrationMDMAppleStatus(cmd.Context())
				if err != nil {
					initFatal(err, "retrieving migration status")
				}
				prepareMigrationStatusCheck(status, noPrompt, dev, "mdm_apple")
				if err := ds.MigrateMDMAppleTables(cmd.Context()); err != nil {
					initFatal(err, "migrating mdm apple db schema")
				}
				if err := ds.MigrateMDMAppleData(cmd.Context()); err != nil {
					initFatal(err, "migrating mdmd apple builtin data")
				}
			}

			fmt.Println("Migrations completed.")
		},
	}

	dbCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "disable prompting before migrations (for use in scripts)")
	dbCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	prepareCmd.AddCommand(dbCmd)

	mdmCmd := &cobra.Command{
		Use:   "mdm-apple",
		Short: "Setup Apple's MDM in Fleet",
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if dev {
				applyDevFlags(&config)
			}

			err := verifyMDMAppleConfig(config)
			if err != nil {
				initFatal(err, "verifying MDM Apple config")
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

			migrationStatusCheck(status, false, dev, "mdm_apple")

			// (1) SCEP Setup

			scepCAKeyPassphrase := []byte(config.MDMApple.SCEP.CA.Passphrase)
			mdmAppleSCEPDepot, err := mds.NewMDMAppleSCEPDepot()
			if err != nil {
				initFatal(err, "initialize SCEP depot")
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
				initFatal(err, "create CA")
			}

			// (2) MDM core setup

			mdmStorage, err := mds.NewMDMAppleMDMStorage()
			if err != nil {
				initFatal(err, "initialize mdm apple MySQL storage")
			}
			err = mdmStorage.StorePushCert(cmd.Context(), config.MDMApple.MDM.PushCert.PEMCert, config.MDMApple.MDM.PushCert.PEMKey)
			if err != nil {
				initFatal(err, "store APNS push certificate")
			}
			topic, err := cryptoutil.TopicFromPEMCert(config.MDMApple.MDM.PushCert.PEMCert)
			if err != nil {
				initFatal(err, "extract topic from push PEM cert")
			}
			err = mdmStorage.SetCurrentTopic(cmd.Context(), topic)
			if err != nil {
				initFatal(err, "setting current push PEM topic")
			}

			fmt.Println("MDM setup completed.")
		},
	}

	mdmCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	prepareCmd.AddCommand(mdmCmd)

	return prepareCmd
}

func prepareMigrationStatusCheck(status *fleet.MigrationStatus, noPrompt, dev bool, dbName string) {
	switch status.StatusCode {
	case fleet.NoMigrationsCompleted:
		// OK
	case fleet.AllMigrationsCompleted:
		fmt.Printf("Migrations already completed for %q. Nothing to do.\n", dbName)
		return
	case fleet.SomeMigrationsCompleted:
		if !noPrompt {
			fmt.Printf("################################################################################\n"+
				"# WARNING:\n"+
				"#   This will perform %q database migrations. Please back up your data before\n"+
				"#   continuing.\n"+
				"#\n"+
				"#   Missing migrations: tables=%v, data=%v.\n"+
				"#\n"+
				"#   Press Enter to continue, or Control-c to exit.\n"+
				"################################################################################\n",
				dbName, status.MissingTable, status.MissingData)
			bufio.NewScanner(os.Stdin).Scan()
		}
	case fleet.UnknownMigrations:
		fmt.Printf("################################################################################\n"+
			"# WARNING:\n"+
			"#   Your %q database has unrecognized migrations. This could happen when\n"+
			"#   running an older version of Fleet on a newer migrated database.\n"+
			"#\n"+
			"#   Unknown migrations: tables=%v, data=%v.\n"+
			"################################################################################\n",
			dbName, status.UnknownTable, status.UnknownData)
		if dev {
			os.Exit(1)
		}
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
