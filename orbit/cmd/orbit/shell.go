package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osquery"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/google/uuid"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var shellCommand = &cli.Command{
	Name:    "shell",
	Aliases: []string{"osqueryi"},
	Usage:   "Run the osqueryi shell",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "osqueryd-channel",
			Usage:   "Channel of osqueryd version to use",
			Value:   "stable",
			EnvVars: []string{"ORBIT_OSQUERYD_CHANNEL"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enable debug logging",
			EnvVars: []string{"ORBIT_DEBUG"},
		},
	},
	Action: func(c *cli.Context) error {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		if err := secure.MkdirAll(c.String("root-dir"), constant.DefaultDirMode); err != nil {
			return fmt.Errorf("initialize root dir: %w", err)
		}

		localStore, err := filestore.New(filepath.Join(c.String("root-dir"), update.MetadataFileName))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create local metadata store")
		}

		// Initialize updater and get expected version
		opt := update.DefaultOptions

		// Override default channel with the provided value.
		opt.Targets.SetTargetChannel(constant.OsqueryTUFTargetName, c.String("osqueryd-channel"))

		opt.RootDirectory = c.String("root-dir")
		opt.ServerURL = c.String("update-url")
		opt.LocalStore = localStore
		opt.InsecureTransport = c.Bool("insecure")

		updater, err := update.NewUpdater(opt)
		if err != nil {
			return err
		}
		if err := updater.UpdateMetadata(); err != nil {
			log.Info().Err(err).Msg("failed to update metadata. using saved metadata.")
		}
		osquerydLocalTarget, err := updater.Get(constant.OsqueryTUFTargetName)
		if err != nil {
			return err
		}
		osquerydPath := osquerydLocalTarget.ExecPath

		var g run.Group

		// We use a suffix on the extension path on Windows
		// because there was an issue when the osqueryd instance ran through
		// `orbit shell` attempted to register the same named pipe name used by
		// the osqueryd instance launched by orbit service.
		extensionPathPostfix := ""
		if runtime.GOOS == "windows" {
			extensionPathPostfix = "-" + uuid.New().String()
		}

		// We use a different path from the orbit daemon
		// to avoid issues with concurrent accesses to files/databases
		// (RocksDB lock and/or Windows file locking).
		dataPath := filepath.Join(c.String("root-dir"), "shell")
		osqueryDB := filepath.Join(dataPath, "osquery.db")
		opts := []osquery.Option{
			osquery.WithShell(),
			osquery.WithDataPath(dataPath, extensionPathPostfix),
			osquery.WithFlags([]string{"--database_path", osqueryDB}),
		}

		// Detect if the additional arguments have a positional argument.
		//
		// osqueryi/osqueryd has the following usage:
		// Usage: osqueryi [OPTION]... [SQL STATEMENT]
		additionalArgs := c.Args().Slice()
		singleQueryArg := false
		if len(additionalArgs) > 0 {
			if !strings.HasPrefix(additionalArgs[len(additionalArgs)-1], "--") {
				singleQueryArg = true
				opts = append(opts, osquery.SingleQuery())
			}
		}

		// Handle additional args after --
		opts = append(opts, osquery.WithFlags(additionalArgs))

		r, err := osquery.NewRunner(osquerydPath, opts...)
		if err != nil {
			return fmt.Errorf("create osquery runner: %w", err)
		}
		g.Add(r.Execute, r.Interrupt)

		if !singleQueryArg {
			// We currently start the extension runner when !singleQueryArg
			// because otherwise osquery exits and leaves too quickly,
			// leaving the extension runner waiting for the socket.
			// NOTE(lucas): `--extensions_require` doesn't seem to work with
			// thrift extensions?
			registerExtensionRunner(&g, r.ExtensionSocketPath()+extensionPathPostfix)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		if err := g.Run(); err != nil {
			log.Error().Err(err).Msg("unexpected exit")
		}

		return nil
	},
}
