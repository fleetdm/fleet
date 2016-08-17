package main

//go:generate make generate

import (
	"fmt"
	"math/rand"
	"net"
	"net/smtp"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jordan-wright/email"
	"github.com/kolide/kolide-ose/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	appName      = "kolide"
	versionMajor = 0
	versionMinor = 1
	versionPatch = 0
	version      = fmt.Sprintf("%d.%d.%d", versionMajor, versionMinor, versionPatch)
)

var (
	configFile string
)

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
  osquery:
      enroll_secret      (string)  (KOLIDE_OSQUERY_ENROLL_SECRET)
      node_key_size      (int)     (KOLIDE_OSQUERY_NODE_KEY_SIZE)
  logging:
      debug              (bool)    (KOLIDE_LOGGING_DEBUG)
      disable_banner     (bool)    (KOLIDE_LOGGING_DISABLE_BANNER)
`,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Launch the kolide server",
	Long: `
Launch the kolide server

Use kolide serve to run the main HTTPS server. The Kolide server bundles
together all static assets and dependent libraries into a statically linked go
binary (which you're executing right now). Use the options below to customize
the way that the kolide server works.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.Get("server.cert") == nil || viper.Get("server.key") == nil {
			logrus.Fatal("TLS certificate and key were not found.")
		}

		db, err := app.OpenDB(
			viper.GetString("mysql.username"),
			viper.GetString("mysql.password"),
			viper.GetString("mysql.address"),
			viper.GetString("mysql.database"),
		)
		if err != nil {
			logrus.Fatalf("Error opening database: %s", err.Error())
		}

		smtpHost, _, err := net.SplitHostPort(viper.GetString("smtp.address"))
		if err != nil {
			logrus.WithError(err).Fatal("Could not parse mail address string")
		}
		smtpConnectionPool := email.NewPool(
			viper.GetString("smtp.address"),
			viper.GetInt("smtp.pool_connections"),
			smtp.PlainAuth("", viper.GetString("smtp.username"), viper.GetString("smtp.password"), smtpHost))

		if !viper.GetBool("logging.disable_banner") {
			fmt.Println(`

 .........77777$7$....................... .   .  .  .. .... .. . .. . ..
........$7777777777................. . .... .. .. . . .. . .. .  ..  . .. ....
......?7777777777777........................... . . . . . . ..... ..   .........
.....777777777777777................  ....    .. ... .... .. . . .. .....    .
...$77......77$....7$............... .. . .. ......... .. . .... ..  ....... .
..$777$.....7$....$77......+DI....DD .DD8DDDN...D8... . $D:..8DDDDDD~...DD88DDDD
$7777777....$....$777$.....+DI..DDD..DDI...8D...D8......$D:..8D....8D...8D......
77777777........777777 ....+DD,DDO...DD... DD...D8......$D:..8D....D8. .D8.. ...
77777777........777777.....+DDDDD....DD....DD...D8......$D:..8D....D8...DDDD....
77777777....7....77777$....+DI..DDD..DD....DD...D8......$D:..8D....D8...DD......
.7777777....7$....77777....+DI...OD8.~DD8DDDD...DDDDDD..$D:..8DDDDDD8...DDDDDD88
.$777777....777....777$....................... ....................... .........
.....=$77777777777777............................ ...... ....... ........ ......
...........=7777777I................  ..  . . ..  ... .   .....   .  ....
..... ...........I.................. .  .   . ..   .   .    .   . .. . .  . .

`)
			fmt.Printf("=> Server starting on https://%s\n", viper.GetString("server.address"))
			fmt.Println("=> Run `kolide serve --help` for more startup options")
			fmt.Println("Use Ctrl-C to stop")
			fmt.Print("\n\n")
		}

		resultFile := viper.GetString("osquery.result_log_file")
		resultHandler := &app.OsqueryLogWriter{
			Writer: &lumberjack.Logger{
				Filename:   resultFile,
				MaxSize:    500, // megabytes
				MaxBackups: 3,
				MaxAge:     28, //days
			},
		}

		statusFile := viper.GetString("osquery.status_log_file")
		statusHandler := &app.OsqueryLogWriter{
			Writer: &lumberjack.Logger{
				Filename:   statusFile,
				MaxSize:    500, // megabytes
				MaxBackups: 3,
				MaxAge:     28, //days
			},
		}

		err = app.CreateServer(
			db,
			smtpConnectionPool,
			os.Stderr,
			resultHandler,
			statusHandler,
		).RunTLS(
			viper.GetString("server.address"),
			viper.GetString("server.cert"),
			viper.GetString("server.key"),
		)
		if err != nil {
			logrus.WithError(err).Fatal("Error running server")
		}
	},
}

var prepareCmd = &cobra.Command{
	Use:   "prepare",
	Short: "Subcommands for initializing kolide infrastructure",
	Long: `
Subcommands for initializing kolide infrastructure

To setup kolide infrastructure, use one of the available commands.
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Given correct database configurations, prepare the databases for use",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := app.OpenDB(
			viper.GetString("mysql.username"),
			viper.GetString("mysql.password"),
			viper.GetString("mysql.address"),
			viper.GetString("mysql.database"),
		)
		if err != nil {
			logrus.Fatalf("Error opening database: %s", err.Error())
		}
		app.DropTables(db)
		app.CreateTables(db)
	},
}

// Due to a deficiency in viper (https://github.com/spf13/viper/issues/71), one
// can not set the default values of nested config elements. For example, if the
// "mysql" section of the config allows a user to define "username", "password",
// and "database", but the only wants to override the default for "username".
// they should be able to create a config which looks like:
//
//   mysql:
//     username: foobar
//
// In viper, that would nullify the default values of all other config keys in
// the mysql section ("mysql.*"). To get around this, instead of using the
// provided API for setting default values, after we've read the config and env,
// we manually check to see if the value has been set and, if it hasn't, we set
// it manually.
func setDefaultConfigValue(key string, value interface{}) {
	if viper.Get(key) == nil {
		viper.Set(key, value)
	}
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	viper.SetConfigName("kolide")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath("./tools/app")
	viper.AddConfigPath("/etc/kolide")

	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("KOLIDE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Infoln("Not reading config file. Relying on environment variables and default values.")
	}

	setDefaultConfigValue("mysql.address", "localhost:3306")
	setDefaultConfigValue("mysql.username", "kolide")
	setDefaultConfigValue("mysql.password", "kolide")
	setDefaultConfigValue("mysql.database", "kolide")

	setDefaultConfigValue("server.address", "0.0.0.0:8080")

	setDefaultConfigValue("app.web_address", "0.0.0.0:8080")

	setDefaultConfigValue("auth.bcrypt_cost", 12)
	setDefaultConfigValue("auth.salt_key_size", 24)

	setDefaultConfigValue("smtp.token_key_size", 24)
	setDefaultConfigValue("smtp.address", "localhost:1025")
	setDefaultConfigValue("smtp.pool_connections", 4)

	setDefaultConfigValue("session.key_size", 64)
	setDefaultConfigValue("session.expiration_seconds", 60*60*24*90)

	setDefaultConfigValue("osquery.node_key_size", 24)
	setDefaultConfigValue("osquery.status_log_file", "/tmp/osquery_status")
	setDefaultConfigValue("osquery.result_log_file", "/tmp/osquery_result")

	setDefaultConfigValue("logging.debug", false)
	setDefaultConfigValue("logging.disable_banner", false)

	if viper.GetBool("logging.debug") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	if viper.GetBool("logs.json") {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}

// logContextHook is a logrus hook which is used to contextualize application
// logs to include data stuch as line numbers, file names, etc.
type logContextHook struct{}

// Levels defines which levels the logContextHook logrus hook should apply to
func (hook logContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire defines what the logContextHook should actually do when it is triggered
func (hook logContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(8); ok {
		funcName := runtime.FuncForPC(pc).Name()

		entry.Data["func"] = path.Base(funcName)
		entry.Data["location"] = fmt.Sprintf("%s:%d", path.Base(file), line)
	}

	return nil
}

func init() {
	gin.SetMode(gin.ReleaseMode)

	logrus.AddHook(logContextHook{})

	rand.Seed(time.Now().UnixNano())

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to a configuration file")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(prepareCmd)
	prepareCmd.AddCommand(dbCmd)
}

func main() {
	rootCmd.Execute()
}
