package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Launch is the entrypoint that sets up and runs the Fleet commands.
func main() {
	rootCmd := createRootCmd()

	configManager := config.NewManager(rootCmd)

	rootCmd.AddCommand(createVulnProcessingCmd(configManager))
	rootCmd.AddCommand(createPrepareCmd(configManager))
	rootCmd.AddCommand(createServeCmd(configManager))
	rootCmd.AddCommand(createConfigDumpCmd(configManager))
	rootCmd.AddCommand(createVersionCmd(configManager))

	if err := rootCmd.Execute(); err != nil {
		initFatal(err, "running root command")
	}
}

// initFatal prints an error message and exits with a non-zero status.
func initFatal(err error, message string) {
	fmt.Printf("Failed to start: %s: %v\n", message, err)
	os.Exit(1)
}

func createRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:   "fleet",
		Short: "osquery management and orchestration",
		Long: `Fleet server (https://fleetdm.com)

Configurable Options:

Options may be supplied in a yaml configuration file or via environment
variables. You only need to define the configuration values for which you
wish to override the default value.
`,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")

	return rootCmd
}

func applyDevFlags(cfg *config.FleetConfig) {
	cfg.Mysql.Username = "fleet"
	cfg.Mysql.Database = "fleet"
	cfg.Mysql.Password = "insecure"

	if cfg.Prometheus.BasicAuth.Username == "" {
		cfg.Prometheus.BasicAuth.Username = "fleet"
	}
	if cfg.Prometheus.BasicAuth.Password == "" {
		cfg.Prometheus.BasicAuth.Password = "insecure"
	}

	cfg.S3 = config.S3Config{
		CarvesBucket:           "carves-dev",
		CarvesRegion:           "minio",
		CarvesPrefix:           "dev-prefix",
		CarvesEndpointURL:      "localhost:9000",
		CarvesAccessKeyID:      "minio",
		CarvesSecretAccessKey:  "minio123!",
		CarvesDisableSSL:       true,
		CarvesForceS3PathStyle: true,

		SoftwareInstallersBucket:           "software-installers-dev",
		SoftwareInstallersRegion:           "minio",
		SoftwareInstallersPrefix:           "dev-prefix",
		SoftwareInstallersEndpointURL:      "localhost:9000",
		SoftwareInstallersAccessKeyID:      "minio",
		SoftwareInstallersSecretAccessKey:  "minio123!",
		SoftwareInstallersDisableSSL:       true,
		SoftwareInstallersForceS3PathStyle: true,
	}

	cfg.Packaging.S3 = config.S3Config{
		Bucket:           "installers-dev",
		Region:           "minio",
		Prefix:           "dev-prefix",
		EndpointURL:      "localhost:9000",
		AccessKeyID:      "minio",
		SecretAccessKey:  "minio123!",
		DisableSSL:       true,
		ForceS3PathStyle: true,
	}
}

func initLogger(cfg config.FleetConfig) kitlog.Logger {
	var logger kitlog.Logger
	{
		output := os.Stderr
		if cfg.Logging.JSON {
			logger = kitlog.NewJSONLogger(output)
		} else {
			logger = kitlog.NewLogfmtLogger(output)
		}
		if cfg.Logging.Debug {
			logger = level.NewFilter(logger, level.AllowDebug())
		} else {
			logger = level.NewFilter(logger, level.AllowInfo())
		}
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
	}
	return logger
}
