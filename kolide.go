package main

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/kolide/kolide-ose/app"
	"github.com/kolide/kolide-ose/config"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	appName        = "kolide"
	appDescription = "osquery command and control"
	versionMajor   = 0
	versionMinor   = 1
	versionPatch   = 0
	commitHash     = ""
	version        = fmt.Sprintf("%d.%d.%d", versionMajor, versionMinor, versionPatch)
	fullVersion    = fmt.Sprintf("%d.%d.%d (commit: %v)", versionMajor, versionMinor, versionPatch, commitHash)
)

var (
	cli = kingpin.New(appName, appDescription)

	configPath = cli.Flag("config", "configuration file").
			Short('c').
			OverrideDefaultFromEnvar("KOLIDE_CONFIG_PATH").
			ExistingFile()

	debug = cli.Flag("debug", "Enable debug mode.").
		OverrideDefaultFromEnvar("KOLIDE_DEBUG").
		Bool()

	logJson = cli.Flag("log_format_json", "Log in JSON format.").
		OverrideDefaultFromEnvar("KOLIDE_LOG_FORMAT_JSON").
		Bool()

	prepareDB = cli.Command("prepare-db", "Create database tables")
	serve     = cli.Command("serve", "Run the Kolide server")
)

func init() {
	// set gin mode to release to silence some superfluous logging
	gin.SetMode(gin.ReleaseMode)

	// configure logging
	logrus.AddHook(logContextHook{})

	rand.Seed(time.Now().UnixNano())
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

func main() {
	// configure flag parsing and parse flags
	cli.Version(version)
	args, err := cli.Parse(os.Args[1:])

	// configure the application based on the flags that have been set
	if *debug {
		config.App.Debug = true
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	if *logJson {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	// if config hasn't been defined and the example config exists relative to
	// the binary, it's likely that the tool is being ran right after building
	// from source so we auto-populate the example config path.
	if *configPath == "" {
		if _, err = os.Stat("./tools/app/example_config.json"); err == nil {
			*configPath = "./tools/app/example_config.json"
		}
		logrus.Warn("Using example config. These settings should be used for development only!")
	}

	// if the user has defined a config path OR the example config is found
	// relative to the binary, load config content from the file. any content
	// in the config file will overwrite the default values
	if *configPath != "" {
		err = config.LoadConfig(*configPath)
		if err != nil {
			logrus.Fatalf("Error loading config: %s", err.Error())
		}
	}

	// route the executable based on the sub-command
	switch kingpin.MustParse(args, err) {
	case prepareDB.FullCommand():
		db, err := app.OpenDB(config.MySQL.Username, config.MySQL.Password, config.MySQL.Address, config.MySQL.Database)
		if err != nil {
			logrus.Fatalf("Error opening database: %s", err.Error())
		}
		app.DropTables(db)
		app.CreateTables(db)
	case serve.FullCommand():
		db, err := app.OpenDB(config.MySQL.Username, config.MySQL.Password, config.MySQL.Address, config.MySQL.Database)
		if err != nil {
			logrus.Fatalf("Error opening database: %s", err.Error())
		}

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
		fmt.Printf("=> %s %s application starting on https://%s\n", cli.Name, version, config.Server.Address)
		fmt.Println("=> Run `kolide help serve` for more startup options")
		fmt.Println("Use Ctrl-C to stop")
		fmt.Print("\n\n")

		err = app.CreateServer(db, os.Stderr).RunTLS(
			config.Server.Address,
			config.Server.Cert,
			config.Server.Key)
		if err != nil {
			logrus.WithError(err).Fatal("Error running server")
		}

	}
}
