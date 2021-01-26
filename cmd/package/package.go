package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/fleetdm/orbit/pkg/update"
	"github.com/fleetdm/orbit/pkg/update/filestore"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Logger = log.Output(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano},
	)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	app.Commands = []*cli.Command{}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	}
	app.Action = buildLinux

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("package failed")
	}
}

func buildLinux(c *cli.Context) error {
	log.Logger = log.Output(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano},
	)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if c.Bool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	tmpDir, err := ioutil.TempDir("", "orbit-package")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	log.Debug().Str("path", tmpDir).Msg("created temp dir")

	filesystemRoot := filepath.Join(tmpDir, "filesystem")
	if err := os.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create filesystem dir")
	}
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "fleet", "orbit")
	if err := os.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create orbit dir")
	}

	systemdRoot := filepath.Join(filesystemRoot, "usr", "lib", "systemd", "system")
	if err := os.MkdirAll(systemdRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create systemd dir")
	}
	if err := ioutil.WriteFile(
		filepath.Join(systemdRoot, "orbit.service"),
		[]byte(`
[Unit]
Description=The osquery Daemon
After=network.service syslog.service

[Service]
TimeoutStartSec=0
ExecStart=/usr/local/bin/orbit --insecure --fleet-url=https://10.0.0.115:8080 --enroll-secret=s96I5x42hhN6c/kqxnpnX2ODkVJBVrOP
Restart=on-failure
KillMode=control-group
KillSignal=SIGTERM
CPUQuota=20%

[Install]
WantedBy=multi-user.target
`),
		constant.DefaultFileMode,
	); err != nil {
		return errors.Wrap(err, "write systemd unit")
	}

	localStore, err := filestore.New(filepath.Join(orbitRoot, "tuf-metadata.json"))
	if err != nil {
		return errors.Wrap(err, "failed to create local metadata store")
	}
	opt := update.DefaultOptions
	opt.RootDirectory = orbitRoot
	opt.ServerURL = "https://tuf.fleetctl.com"
	opt.LocalStore = localStore
	opt.Platform = "linux"

	updater, err := update.New(opt)
	if err != nil {
		return errors.Wrap(err, "failed to init updater")
	}
	if err := updater.UpdateMetadata(); err != nil {
		return errors.Wrap(err, "failed to update metadata")
	}
	osquerydPath, err := updater.Get("osqueryd", "stable")
	if err != nil {
		return errors.Wrap(err, "failed to get osqueryd")
	}
	log.Debug().Str("path", osquerydPath).Msg("got osqueryd")

	contents := files.Contents{
		&files.Content{
			Source:      filepath.Join(filesystemRoot, "**"),
			Destination: "/",
		},
		&files.Content{
			Source:      "orbit",
			Destination: "/var/lib/fleet/orbit/orbit",
			FileInfo: &files.ContentFileInfo{
				Mode: constant.DefaultExecutableMode,
			},
		},
		&files.Content{
			Source:      "/var/lib/fleet/orbit/orbit",
			Destination: "/usr/local/bin/orbit",
			Type:        "symlink",
			FileInfo: &files.ContentFileInfo{
				// TODO follow up on nfpm not respecting this
				// https://github.com/goreleaser/nfpm/issues/286
				Mode: constant.DefaultExecutableMode | os.ModeSymlink,
			},
		},
	}
	contents, err = files.ExpandContentGlobs(contents, false)
	if err != nil {
		return errors.Wrap(err, "glob contents")
	}
	for _, c := range contents {
		log.Debug().Interface("file", c).Msg("added file")
	}

	postInstall := `#!/bin/sh

# If we have a systemd, daemon-reload away now
if [ -x /bin/systemctl ] && pidof systemd ; then
  /bin/systemctl daemon-reload 2>/dev/null 2>&1 && /bin/systemctl start orbit.service 2>&1
fi
`
	postInstallPath := filepath.Join(tmpDir, "postinstall.sh")
	if err := ioutil.WriteFile(
		postInstallPath,
		[]byte(postInstall),
		constant.DefaultFileMode,
	); err != nil {
		return errors.Wrap(err, "write postinstall")
	}

	info := &nfpm.Info{
		Name:        "orbit-osquery",
		Version:     "0.0.1",
		Description: "Osquery launcher and autoupdater",
		Arch:        "amd64",
		Maintainer:  "FleetDM Engineers <engineering@fleetdm.com>",
		Homepage:    "https://github.com/fleetdm/orbit",
		Overridables: nfpm.Overridables{
			Contents: contents,
			EmptyFolders: []string{
				"/var/log/osquery",
				"/var/log/fleet/orbit",
			},
			Scripts: nfpm.Scripts{
				PostInstall: postInstallPath,
			},
		},
	}

	pkger := deb.Default
	filename := pkger.ConventionalFileName(info)

	out, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, constant.DefaultFileMode)
	if err != nil {
		return errors.Wrap(err, "open output file")
	}
	defer out.Close()

	if err := deb.Default.Package(info, out); err != nil {
		return errors.Wrap(err, "write deb package")
	}
	log.Info().Str("path", filename).Msg("wrote deb package")

	return nil
}
