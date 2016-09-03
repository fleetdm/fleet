package cli

import (
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/jordan-wright/email"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

// detect that docker --link to mysql container is up and use
// the default env vars
func dockerMySQLconnString() string {
	var (
		username = os.Getenv("MYSQL_ENV_MYSQL_USER")
		password = os.Getenv("MYSQL_ENV_MYSQL_PASSWORD")
		host     = os.Getenv("MYSQL_PORT_3306_TCP_ADDR")
		port     = os.Getenv("MYSQL_PORT_3306_TCP_PORT")
		dbName   = os.Getenv("MYSQL_ENV_MYSQL_DATABASE")
	)

	if host == "" {
		return "" // no docker conn detected
	}
	logrus.Infoln("detected docker mysql link, using link environment vars")

	connString := fmt.Sprintf(
		"%s:%s@(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		username, password, host, port, dbName,
	)
	return connString
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
		resultHandler := &server.OsqueryLogWriter{
			Writer: &lumberjack.Logger{
				Filename:   resultFile,
				MaxSize:    500, // megabytes
				MaxBackups: 3,
				MaxAge:     28, //days
			},
		}

		statusFile := viper.GetString("osquery.status_log_file")
		statusHandler := &server.OsqueryLogWriter{
			Writer: &lumberjack.Logger{
				Filename:   statusFile,
				MaxSize:    500, // megabytes
				MaxBackups: 3,
				MaxAge:     28, //days
			},
		}

		// get mysql connection from config or docker link
		// temporary until config is redone
		var connString string
		if os.Getenv("MYSQL_PORT_3306_TCP_ADDR") != "" {
			// try connection from docker link
			connString = dockerMySQLconnString()
		} else {
			connString = fmt.Sprintf(
				"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
				viper.GetString("mysql.username"),
				viper.GetString("mysql.password"),
				viper.GetString("mysql.address"),
				viper.GetString("mysql.database"),
			)
		}

		ds, err := datastore.New("gorm-mysql", connString)
		if err != nil {
			logrus.WithError(err).Fatal("error creating db connection")
		}

		handler := server.CreateServer(
			ds,
			smtpConnectionPool,
			os.Stderr,
			resultHandler,
			statusHandler,
		)
		err = http.ListenAndServeTLS(
			viper.GetString("server.address"),
			viper.GetString("server.cert"),
			viper.GetString("server.key"),
			handler,
		)
		if err != nil {
			logrus.WithError(err).Fatal("Error running server")
		}
	},
}
