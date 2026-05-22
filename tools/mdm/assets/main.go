package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var validNames = map[fleet.MDMAssetName]struct{}{
	fleet.MDMAssetABMCert:                  {},
	fleet.MDMAssetABMTokenDeprecated:       {},
	fleet.MDMAssetABMKey:                   {},
	fleet.MDMAssetAPNSCert:                 {},
	fleet.MDMAssetAPNSKey:                  {},
	fleet.MDMAssetCACert:                   {},
	fleet.MDMAssetCAKey:                    {},
	fleet.MDMAssetSCEPChallenge:            {},
	fleet.MDMAssetVPPTokenDeprecated:       {},
	fleet.MDMAssetAndroidFleetServerSecret: {},
}

var legacyFlagAliases = map[string]string{
	"-key":             "--key",
	"-dir":             "--dir",
	"-name":            "--name",
	"-value":           "--value",
	"-db-user":         "--mysql_username",
	"-db-password":     "--mysql_password",
	"-db-address":      "--mysql_address",
	"-db-name":         "--mysql_database",
	"-tls-config":      "--mysql_tls_config",
	"-tls-ca":          "--mysql_tls_ca",
	"-tls-cert":        "--mysql_tls_cert",
	"-tls-key":         "--mysql_tls_key",
	"-tls-server-name": "--mysql_tls_server_name",
}

type commandOptions struct {
	key string
	dir string
}

type importOptions struct {
	commandOptions
	name  string
	value string
}

type exportOptions struct {
	commandOptions
	name string
}

func setupDS(fleetConfig config.FleetConfig) *mysql.Datastore {
	opts := []mysql.DBOption{
		mysql.LimitAttempts(1),
		mysql.WithFleetConfig(&fleetConfig),
	}

	ds, err := mysql.New(
		fleetConfig.Mysql,
		clock.NewMockClock(),
		opts...,
	)
	if err != nil {
		log.Fatal("creating datastore instance:", err) //nolint:gocritic // ignore exitAfterDefer
	}

	return ds
}

func privateKeyFromOptions(opts commandOptions, fleetConfig config.FleetConfig) string {
	privateKey := opts.key
	if privateKey == "" {
		privateKey = fleetConfig.Server.PrivateKey
	}
	if len(privateKey) > 32 {
		// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
		// infra setups generate keys that are longer than 32 bytes.
		privateKey = privateKey[:32]
	}
	return privateKey
}

func validatePrivateKey(privateKey string) error {
	if privateKey == "" {
		return errors.New("-key flag or server.private_key config is required")
	}
	if len(privateKey) < 32 {
		return errors.New("private key must be at least 32 bytes long")
	}
	return nil
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage MDM assets in a Fleet database",
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to a Fleet configuration file")

	configManager := config.NewManager(rootCmd)
	hideUnrelatedFleetConfigFlags(rootCmd)
	rootCmd.AddCommand(newImportCmd(configManager))
	rootCmd.AddCommand(newExportCmd(configManager))
	return rootCmd
}

func hideUnrelatedFleetConfigFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "config" ||
			(strings.HasPrefix(flag.Name, "mysql_") && !strings.HasPrefix(flag.Name, "mysql_read_replica_")) ||
			flag.Name == "server_private_key" {
			return
		}
		flag.Hidden = true
	})
}

func normalizeLegacyArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		if replacement, ok := legacyFlagAliases[arg]; ok {
			normalized = append(normalized, replacement)
			continue
		}
		if flagName, flagValue, ok := strings.Cut(arg, "="); ok {
			if replacement, ok := legacyFlagAliases[flagName]; ok {
				normalized = append(normalized, replacement+"="+flagValue)
				continue
			}
		}
		normalized = append(normalized, arg)
	}
	return normalized
}

func addCommonFlags(cmd *cobra.Command, opts *commandOptions) {
	cmd.Flags().StringVar(&opts.key, "key", "", "Key used to encrypt the assets")
	cmd.Flags().StringVar(&opts.dir, "dir", "", "Directory to put the exported assets")
}

func newImportCmd(configManager config.Manager) *cobra.Command {
	opts := importOptions{}
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an MDM asset into Fleet",
		Run: func(cmd *cobra.Command, args []string) {
			runImport(cmd.Context(), configManager, opts)
		},
	}
	addCommonFlags(cmd, &opts.commandOptions)
	cmd.Flags().StringVar(&opts.name, "name", "", "Name of the asset to import. Valid names are: apns_cert, apns_key, ca_cert, ca_key, abm_key, abm_cert, abm_token, scep_challenge, vpp_token")
	cmd.Flags().StringVar(&opts.value, "value", "", "Value of the asset to import")
	return cmd
}

func runImport(ctx context.Context, configManager config.Manager, opts importOptions) {
	fleetConfig := configManager.LoadConfig()
	fleetConfig.Server.PrivateKey = privateKeyFromOptions(opts.commandOptions, fleetConfig)
	if err := validatePrivateKey(fleetConfig.Server.PrivateKey); err != nil {
		log.Fatal(err)
	}

	ds := setupDS(fleetConfig)
	defer ds.Close()

	if opts.dir != "" {
		if err := os.MkdirAll(opts.dir, os.ModePerm); err != nil {
			log.Fatal("ensuring directory: ", err) //nolint:gocritic // ignore exitAfterDefer
		}
	}

	if opts.name == "" {
		log.Fatal("-name flag is required")
	}

	if opts.value == "" {
		log.Fatal("-value flag is required")
	}

	if _, ok := validNames[fleet.MDMAssetName(opts.name)]; !ok {
		log.Fatalf("invalid asset name %s", opts.name)
	}

	err := ds.ReplaceMDMConfigAssets(ctx,
		[]fleet.MDMConfigAsset{{Name: fleet.MDMAssetName(opts.name), Value: []byte(opts.value)}}, nil)
	if err != nil {
		log.Fatal("writing asset to db: ", err)
	}
}

func newExportCmd(configManager config.Manager) *cobra.Command {
	opts := exportOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export MDM assets from Fleet",
		Run: func(cmd *cobra.Command, args []string) {
			runExport(cmd.Context(), configManager, opts)
		},
	}
	addCommonFlags(cmd, &opts.commandOptions)
	cmd.Flags().StringVar(&opts.name, "name", "", "Name of the asset to export. Valid names are: apns_cert, apns_key, ca_cert, ca_key, abm_key, abm_cert, abm_token, scep_challenge, vpp_token")
	return cmd
}

func runExport(ctx context.Context, configManager config.Manager, opts exportOptions) {
	fleetConfig := configManager.LoadConfig()
	fleetConfig.Server.PrivateKey = privateKeyFromOptions(opts.commandOptions, fleetConfig)
	if err := validatePrivateKey(fleetConfig.Server.PrivateKey); err != nil {
		log.Fatal(err)
	}

	ds := setupDS(fleetConfig)
	defer ds.Close()

	if opts.dir != "" {
		if err := os.MkdirAll(opts.dir, os.ModePerm); err != nil {
			log.Fatal("ensuring directory: ", err)
		}
	}

	names := []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
		fleet.MDMAssetCAKey,
		fleet.MDMAssetAPNSKey,
		fleet.MDMAssetAPNSCert,
		fleet.MDMAssetABMCert,
		fleet.MDMAssetABMKey,
		fleet.MDMAssetABMTokenDeprecated,
		fleet.MDMAssetSCEPChallenge,
		fleet.MDMAssetVPPTokenDeprecated,
	}

	if opts.name != "" {
		if _, ok := validNames[fleet.MDMAssetName(opts.name)]; !ok {
			log.Fatalf("invalid asset name %s", opts.name)
		}

		names = []fleet.MDMAssetName{fleet.MDMAssetName(opts.name)}
	}

	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, names, nil)
	if err != nil && !errors.Is(err, mysql.ErrPartialResult) {
		log.Fatal("retrieving assets from db:", err)
	}

	for _, asset := range assets {
		path := filepath.Join(opts.dir, string(asset.Name))
		switch {
		case strings.Contains(path, "_key"):
			path += ".key"
		case strings.Contains(path, "_cert"):
			path += ".crt"
		}
		if err := os.WriteFile(path, asset.Value, 0o600); err != nil {
			log.Fatal("writing asset:", err)
		}

		log.Printf("wrote %s in %s", asset.Name, path)
	}

	absDir, err := filepath.Abs(opts.dir)
	if err != nil {
		log.Fatalf("abs path: %s", err)
	}

	fmt.Printf(`You can set the following on your Fleet configuration:
export FLEET_MDM_APPLE_APNS_CERT=%[1]s/apns_cert.crt
export FLEET_MDM_APPLE_APNS_KEY=%[1]s/apns_key.key
export FLEET_MDM_APPLE_SCEP_CERT=%[1]s/ca_cert.crt
export FLEET_MDM_APPLE_SCEP_KEY=%[1]s/ca_key.key
export FLEET_MDM_APPLE_SCEP_CHALLENGE=$(cat %[1]s/scep_challenge)
export FLEET_MDM_APPLE_BM_SERVER_TOKEN=%[1]s/abm_token
export FLEET_MDM_APPLE_BM_CERT=%[1]s/abm_cert.crt
export FLEET_MDM_APPLE_BM_KEY=%[1]s/abm_key.key
`, absDir)
}

func main() {
	rootCmd := newRootCmd()
	rootCmd.SetArgs(normalizeLegacyArgs(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
