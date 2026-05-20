package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/spf13/viper"
)

const (
	testUsername = "fleet"
	testPassword = "insecure"
	testAddress  = "localhost:3306"
	testDatabase = "fleet"
)

var cfg *viper.Viper

var (
	exportCmd       = flag.NewFlagSet("export", flag.ExitOnError)
	importCmd       = flag.NewFlagSet("import", flag.ExitOnError)
	flagKey         string
	flagDir         string
	flagDBUser      string
	flagDBPass      string
	flagDBAddress   string
	flagDBName      string
	flagImportName  string
	flagImportValue string
	flagExportName  string

	// TLS configuration flags
	flagTLSConfig     string
	flagTLSCA         string
	flagTLSCert       string
	flagTLSKey        string
	flagTLSServerName string

	validNames = map[fleet.MDMAssetName]struct{}{
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
)

// initConfig initializes the viper configuration. It loads defaults,
// reads environment variables (prefixed ASSETS_DB_), and optionally
// reads a .env file from the current directory.
func initConfig() {
	cfg = viper.New()
	cfg.SetConfigFile(".env")
	cfg.SetConfigType("env")

	// Set defaults (lowest priority)
	cfg.SetDefault("USER", testUsername)
	cfg.SetDefault("PASSWORD", testPassword)
	cfg.SetDefault("ADDRESS", testAddress)
	cfg.SetDefault("NAME", testDatabase)
	cfg.SetDefault("TLS_CONFIG", "skip-verify")
	cfg.SetDefault("TLS_CA", "")
	cfg.SetDefault("TLS_CERT", "")
	cfg.SetDefault("TLS_KEY", "")
	cfg.SetDefault("TLS_SERVER_NAME", "")

	// Read env vars (ASSETS_DB_*) — higher priority than .env file values
	cfg.AutomaticEnv()
	cfg.SetEnvPrefix("ASSETS_DB")

	// Load .env file if present (lower priority than shell env vars)
	_ = cfg.ReadInConfig()
}

// cfgOr returns the config value for the given key, falling back
// to the default if not set. Keys are looked up via viper with
// AutomaticEnv (ASSETS_DB_ prefix).
func cfgOr(key string, defaultValue string) string {
	if v := cfg.GetString(key); v != "" {
		return v
	}
	return defaultValue
}

func setupSharedFlags() {
	for _, fs := range []*flag.FlagSet{exportCmd, importCmd} {
		fs.StringVar(&flagKey, "key", "", "Key used to encrypt the assets")
		fs.StringVar(&flagDir, "dir", "", "Directory to put the exported assets")
		fs.StringVar(&flagDBUser, "db-user", cfgOr("USER", testUsername), "Username used to connect to the MySQL instance")
		fs.StringVar(&flagDBPass, "db-password", cfgOr("PASSWORD", testPassword), "Password used to connect to the MySQL instance")
		fs.StringVar(&flagDBAddress, "db-address", cfgOr("ADDRESS", testAddress), "Address used to connect to the MySQL instance")
		fs.StringVar(&flagDBName, "db-name", cfgOr("NAME", testDatabase), "Name of the database with the asset information in the MySQL instance")

		// TLS configuration flags
		fs.StringVar(&flagTLSConfig, "tls-config", cfgOr("TLS_CONFIG", "skip-verify"), "TLS configuration for MySQL connection (e.g., skip-verify, custom, or a registered TLS config name)")
		fs.StringVar(&flagTLSCA, "tls-ca", cfgOr("TLS_CA", ""), "Path to the CA certificate file for MySQL TLS")
		fs.StringVar(&flagTLSCert, "tls-cert", cfgOr("TLS_CERT", ""), "Path to the client certificate file for MySQL TLS")
		fs.StringVar(&flagTLSKey, "tls-key", cfgOr("TLS_KEY", ""), "Path to the client key file for MySQL TLS")
		fs.StringVar(&flagTLSServerName, "tls-server-name", cfgOr("TLS_SERVER_NAME", ""), "Server name to use for MySQL TLS certificate verification")
	}
}

func setupDS(privateKey, userName, password, address, name string, tlsConfig config.MysqlConfig) *mysql.Datastore {
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true&tls=%s", userName, password, address, name, tlsConfig.TLSConfig),
	)
	if err != nil {
		log.Fatal("opening MySQL connection:", err)
	}
	defer db.Close()

	mysqlCfg := config.MysqlConfig{
		Username:      userName,
		Password:      password,
		Address:       address,
		Database:      name,
		TLSConfig:     tlsConfig.TLSConfig,
		TLSCA:         tlsConfig.TLSCA,
		TLSCert:       tlsConfig.TLSCert,
		TLSKey:        tlsConfig.TLSKey,
		TLSServerName: tlsConfig.TLSServerName,
	}
	ds, err := mysql.New(
		mysqlCfg,
		clock.NewMockClock(),
		mysql.LimitAttempts(1),
		mysql.WithFleetConfig(&config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: privateKey,
			},
		}),
	)
	if err != nil {
		log.Fatal("creating datastore instance:", err) //nolint:gocritic // ignore exitAfterDefer
	}

	return ds
}

func buildTLSConfig() config.MysqlConfig {
	tlsConfig := config.MysqlConfig{
		TLSConfig:     flagTLSConfig,
		TLSCA:         flagTLSCA,
		TLSCert:       flagTLSCert,
		TLSKey:        flagTLSKey,
		TLSServerName: flagTLSServerName,
	}
	return tlsConfig
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("invalid subcommand, expected import or export")
	}

	// Load config: defaults → .env file → ASSETS_DB_* env vars
	initConfig()

	ctx := context.Background()

	// Flag setup
	setupSharedFlags()
	importCmd.StringVar(&flagImportName, "name", "", "Name of the asset to import. Valid names are: apns_cert, apns_key, ca_cert, ca_key, abm_key, abm_cert, abm_token, scep_challenge, vpp_token")
	importCmd.StringVar(&flagImportValue, "value", "", "Value of the asset to import")
	exportCmd.StringVar(&flagExportName, "name", "", "Name of the asset to export. Valid names are: apns_cert, apns_key, ca_cert, ca_key, abm_key, abm_cert, abm_token, scep_challenge, vpp_token")

	// Execute subcommands
	switch os.Args[1] {
	case "import":
		if err := importCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal("parsing import flags", err)
		}

		if len(flagKey) > 32 {
			// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
			// infra setups generate keys that are longer than 32 bytes.
			flagKey = flagKey[:32]
		}

		ds := setupDS(flagKey, flagDBUser, flagDBPass, flagDBAddress, flagDBName, buildTLSConfig())
		defer ds.Close()

		// Check required flags
		if flagDir != "" {
			if err := os.MkdirAll(flagDir, os.ModePerm); err != nil {
				log.Fatal("ensuring directory: ", err) //nolint:gocritic // ignore exitAfterDefer
			}
		}

		if flagImportName == "" {
			log.Fatal("-name flag is required")
		}

		if flagImportValue == "" {
			log.Fatal("-value flag is required")
		}

		if _, ok := validNames[fleet.MDMAssetName(flagImportName)]; !ok {
			log.Fatalf("invalid asset name %s", flagImportName)
		}

		err := ds.ReplaceMDMConfigAssets(ctx,
			[]fleet.MDMConfigAsset{{Name: fleet.MDMAssetName(flagImportName), Value: []byte(flagImportValue)}}, nil)
		if err != nil {
			log.Fatal("writing asset to db: ", err)
		}
		return
	case "export":
		if err := exportCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal("parsing export flags", err)
		}

		// Check required flags
		if flagKey == "" {
			log.Fatal("-key flag is required")
		}

		if len(flagKey) > 32 {
			// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
			// infra setups generate keys that are longer than 32 bytes.
			flagKey = flagKey[:32]
		}

		ds := setupDS(flagKey, flagDBUser, flagDBPass, flagDBAddress, flagDBName, buildTLSConfig())
		defer ds.Close()

		if flagDir != "" {
			if err := os.MkdirAll(flagDir, os.ModePerm); err != nil {
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

		if flagExportName != "" {
			if _, ok := validNames[fleet.MDMAssetName(flagExportName)]; !ok {
				log.Fatalf("invalid asset name %s", flagExportName)
			}

			names = []fleet.MDMAssetName{fleet.MDMAssetName(flagExportName)}
		}

		assets, err := ds.GetAllMDMConfigAssetsByName(ctx, names, nil)
		if err != nil && !errors.Is(err, mysql.ErrPartialResult) {
			log.Fatal("retrieving assets from db:", err)
		}

		for _, asset := range assets {
			path := filepath.Join(flagDir, string(asset.Name))
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

		flagDir, err = filepath.Abs(flagDir)
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
`, flagDir)
	default:
		log.Fatalf("invalid subcommand %s, valid subcommands: import, export", os.Args[1])
	}
}
