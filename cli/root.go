package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	ConfigFile string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&ConfigFile, "config", "", "Path to a configuration file")
}

func Launch() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

// RootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kolide",
	Short: "osquery management and orchestration",
	Long: `
osquery management and orchestration

Configurable Options:

Options may be supplied in a yaml configuration file or via environment
variables. You only need to define the configuration values for which you
wish to override the default value.

Available Configurations:

  mysql:
      address            (string)  (KOLIDE_MYSQL_ADDRESS)
      username           (string)  (KOLIDE_MYSQL_USERNAME)
      password           (string)  (KOLIDE_MYSQL_PASSWORD)
      database           (string)  (KOLIDE_MYSQL_DATABASE)
  server:
      address            (string)  (KOLIDE_SERVER_ADDRESS)
      cert               (string)  (KOLIDE_SERVER_CERT)
      key                (string)  (KOLIDE_SERVER_KEY)
  auth:
      jwt_key            (string)  (KOLIDE_AUTH_JWT_KEY)
      salt_key_size      (int)     (KOLIDE_AUTH_SALT_KEY_SIZE)
      bcrypt_cost        (int)     (KOLIDE_AUTH_BCRYPT_COST)
  app:
      web_address        (string)  (KOLIDE_APP_WEB_ADDRESS)
  smtp:
      server             (string)  (KOLIDE_SMTP_SERVER)
      username           (string)  (KOLIDE_SMTP_USERNAME)
      password           (string)  (KOLIDE_SMTP_PASSWORD)
      pool_connections   (int)     (KOLIDE_SMTP_POOL_CONNECTIONS)
      token_key_size     (int)     (KOLIDE_SMTP_TOKEN_KEY_SIZE)
  session:
      key_size           (int)     (KOLIDE_SESSION_KEY_SIZE)
      expiration_seconds (float64) (KOLIDE_SESSION_EXPIRATION_SECONDS)
      cookie_name	 (string)  (KOLIDE_SESSION_COOKIE_NAME)
  osquery:
      enroll_secret      (string)  (KOLIDE_OSQUERY_ENROLL_SECRET)
      node_key_size      (int)     (KOLIDE_OSQUERY_NODE_KEY_SIZE)
      status_log_file    (string)  (KOLIDE_OSQUERY_STATUS_LOG_FILE)
      result_log_file    (string)  (KOLIDE_OSQUERY_RESULT_LOG_FILE)
      label_up_interval  (int)     (KOLIDE_OSQUERY_LABEL_UP_INTERVAL)
  logging:
      debug              (bool)    (KOLIDE_LOGGING_DEBUG)
      disable_banner     (bool)    (KOLIDE_LOGGING_DISABLE_BANNER)
`,
}
