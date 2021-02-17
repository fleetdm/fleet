package packaging

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/fleetdm/orbit/pkg/update"
	"github.com/fleetdm/orbit/pkg/update/filestore"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func BuildDeb(opt Options) error {
	// Initialize directories

	tmpDir, err := ioutil.TempDir("", "orbit-package")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	log.Debug().Str("path", tmpDir).Msg("created temp dir")

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := os.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create root dir")
	}
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "fleet", "orbit")
	if err := os.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create orbit dir")
	}

	// Initialize autoupdate metadata

	localStore, err := filestore.New(filepath.Join(orbitRoot, "tuf-metadata.json"))
	if err != nil {
		return errors.Wrap(err, "failed to create local metadata store")
	}
	updateOpt := update.DefaultOptions
	updateOpt.RootDirectory = orbitRoot
	updateOpt.ServerURL = "https://tuf.fleetctl.com"
	updateOpt.LocalStore = localStore
	updateOpt.Platform = "linux"

	updater, err := update.New(updateOpt)
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

	// Write files

	if err := writeSystemdUnit(opt, filesystemRoot); err != nil {
		return errors.Wrap(err, "write systemd unit")
	}

	if err := writeEnvFile(opt, filesystemRoot); err != nil {
		return errors.Wrap(err, "write env file")
	}

	postInstallPath := filepath.Join(tmpDir, "postinstall.sh")
	if err := writePostInstall(opt, postInstallPath); err != nil {
		return errors.Wrap(err, "write postinstall script")
	}

	// Pick up all file contents

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

	// Build package

	info := &nfpm.Info{
		Name:        "orbit-osquery",
		Version:     opt.Version,
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

func writeSystemdUnit(opt Options, rootPath string) error {
	systemdRoot := filepath.Join(rootPath, "usr", "lib", "systemd", "system")
	if err := os.MkdirAll(systemdRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create systemd dir")
	}
	if err := ioutil.WriteFile(
		filepath.Join(systemdRoot, "orbit.service"),
		[]byte(`
[Unit]
Description=Fleet Orbit osquery
After=network.service syslog.service

[Service]
TimeoutStartSec=0
EnvironmentFile=/etc/default/orbit
ExecStart=/usr/local/bin/orbit
Restart=on-failure
KillMode=control-group
KillSignal=SIGTERM
CPUQuota=20%

[Install]
WantedBy=multi-user.target
`),
		constant.DefaultFileMode,
	); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

var envTemplate = template.Must(template.New("env").Parse(`
{{- if .Insecure }}ORBIT_INSECURE=true{{ end }}
{{ if .FleetURL }}ORBIT_FLEET_URL={{.FleetURL}}{{ end }}
{{ if .EnrollSecret }}ORBIT_ENROLL_SECRET={{.EnrollSecret}}{{ end }}
`))

func writeEnvFile(opt Options, rootPath string) error {
	envRoot := filepath.Join(rootPath, "etc", "default")
	if err := os.MkdirAll(envRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create env dir")
	}

	var contents bytes.Buffer
	if err := envTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(
		filepath.Join(envRoot, "orbit"),
		contents.Bytes(),
		constant.DefaultFileMode,
	); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

var postInstallTemplate = template.Must(template.New("postinstall").Parse(`
#!/bin/sh

# Exit on error
set -e

# If we have a systemd, daemon-reload away now
if [ -x /bin/systemctl ] && pidof systemd ; then
  /bin/systemctl daemon-reload 2>/dev/null 2>&1
{{ if .StartService -}}
  /bin/systemctl start orbit.service 2>&1
{{- end}}
fi
`))

func writePostInstall(opt Options, path string) error {
	var contents bytes.Buffer
	if err := postInstallTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}
