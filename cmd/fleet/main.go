package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/shellquote"
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

	// See if the program is being piped data on stdin.
	fi, err := os.Stdin.Stat()
	if err != nil {
		initFatal(err, "getting stdin stats")
	}
	if fi.Mode()&os.ModeNamedPipe != 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Reading additional arguments from stdin...")
		// See charsets at https://godoc.org/github.com/briandowns/spinner#pkg-variables
		s := spinner.New(spinner.CharSets[24], 200*time.Millisecond)
		s.Writer = os.Stderr
		s.Start()

		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			initFatal(err, "reading from stdin")
		}

		// Split the string into arguments like a shell would.
		extraArgs, err := shellquote.Split(string(data))
		if err != nil {
			initFatal(err, "splitting arguments from stdin")
		}

		// Add the new args to the existing args
		os.Args = append(os.Args, extraArgs...)

		s.Stop()
	}

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

	cfg.S3.CarvesBucket = "carves-dev"
	cfg.S3.CarvesRegion = "minio"
	cfg.S3.CarvesPrefix = "dev-prefix"
	cfg.S3.CarvesEndpointURL = "localhost:9000"
	cfg.S3.CarvesAccessKeyID = "minio"
	cfg.S3.CarvesSecretAccessKey = "minio123!"
	cfg.S3.CarvesDisableSSL = true
	cfg.S3.CarvesForceS3PathStyle = true

	// Allow the software installers bucket to be overridden in dev mode
	if cfg.S3.SoftwareInstallersBucket == "" {
		cfg.S3.SoftwareInstallersBucket = "software-installers-dev"
		cfg.S3.SoftwareInstallersRegion = "minio"
		cfg.S3.SoftwareInstallersPrefix = "dev-prefix"
		cfg.S3.SoftwareInstallersEndpointURL = "localhost:9000"
		cfg.S3.SoftwareInstallersAccessKeyID = "minio"
		cfg.S3.SoftwareInstallersSecretAccessKey = "minio123!"
		cfg.S3.SoftwareInstallersDisableSSL = true
		cfg.S3.SoftwareInstallersForceS3PathStyle = true
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
