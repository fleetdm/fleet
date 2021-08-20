package packaging

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/secure"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func buildNFPM(opt Options, pkger nfpm.Packager) error {
	// Initialize directories

	tmpDir, err := ioutil.TempDir("", "orbit-package")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	log.Debug().Str("path", tmpDir).Msg("created temp dir")

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create root dir")
	}
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "orbit")
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create orbit dir")
	}

	// Initialize autoupdate metadata

	updateOpt := update.DefaultOptions
	updateOpt.Platform = "linux"
	updateOpt.RootDirectory = orbitRoot
	updateOpt.OrbitChannel = opt.OrbitChannel
	updateOpt.OsquerydChannel = opt.OsquerydChannel
	updateOpt.ServerURL = opt.UpdateURL
	if opt.UpdateRoots != "" {
		updateOpt.RootKeys = opt.UpdateRoots
	}

	if err := initializeUpdates(updateOpt); err != nil {
		return errors.Wrap(err, "initialize updates")
	}

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

	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return errors.Wrap(err, "write fleet certificate")
		}
	}

	// Pick up all file contents

	contents := files.Contents{
		&files.Content{
			Source:      filepath.Join(filesystemRoot, "**"),
			Destination: "/",
		},
		// Symlink current into /var/lib/orbit/bin/orbit/orbit
		&files.Content{
			Source:      "/var/lib/orbit/bin/orbit/linux/" + opt.OrbitChannel + "/orbit",
			Destination: "/var/lib/orbit/bin/orbit/orbit",
			Type:        "symlink",
			FileInfo: &files.ContentFileInfo{
				Mode: constant.DefaultExecutableMode | os.ModeSymlink,
			},
		},
		// Symlink current into /usr/local/bin
		&files.Content{
			Source:      "/var/lib/orbit/bin/orbit/orbit",
			Destination: "/usr/local/bin/orbit",
			Type:        "symlink",
			FileInfo: &files.ContentFileInfo{
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
		Description: "Orbit osquery -- runtime and autoupdater by Fleet",
		Arch:        "amd64",
		Maintainer:  "Fleet Engineers <engineering@fleetdm.com>",
		Homepage:    "https://github.com/fleetdm/orbit",
		Overridables: nfpm.Overridables{
			Contents: contents,
			EmptyFolders: []string{
				"/var/log/osquery",
				"/var/log/orbit",
			},
			Scripts: nfpm.Scripts{
				PostInstall: postInstallPath,
			},
		},
	}
	filename := pkger.ConventionalFileName(info)

	out, err := secure.OpenFile(filename, os.O_CREATE|os.O_RDWR, constant.DefaultFileMode)
	if err != nil {
		return errors.Wrap(err, "open output file")
	}
	defer out.Close()

	if err := pkger.Package(info, out); err != nil {
		return errors.Wrap(err, "write package")
	}
	if err := out.Sync(); err != nil {
		return errors.Wrap(err, "sync output file")
	}
	log.Info().Str("path", filename).Msg("wrote package")

	return nil
}

func writeSystemdUnit(opt Options, rootPath string) error {
	systemdRoot := filepath.Join(rootPath, "usr", "lib", "systemd", "system")
	if err := secure.MkdirAll(systemdRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create systemd dir")
	}
	if err := ioutil.WriteFile(
		filepath.Join(systemdRoot, "orbit.service"),
		[]byte(`
[Unit]
Description=Orbit osquery
After=network.service syslog.service
StartLimitIntervalSec=0

[Service]
TimeoutStartSec=0
EnvironmentFile=/etc/default/orbit
ExecStart=/var/lib/orbit/bin/orbit/orbit
Restart=always
RestartSec=1
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
ORBIT_UPDATE_URL={{ .UpdateURL }}
ORBIT_ORBIT_CHANNEL={{ .OrbitChannel }}
ORBIT_OSQUERYD_CHANNEL={{ .OsquerydChannel }}
{{ if .Insecure }}ORBIT_INSECURE=true{{ end }}
{{ if .FleetURL }}ORBIT_FLEET_URL={{.FleetURL}}{{ end }}
{{ if .FleetCertificate }}ORBIT_FLEET_CERTIFICATE=/var/lib/orbit/fleet.pem{{ end }}
{{ if .EnrollSecret }}ORBIT_ENROLL_SECRET={{.EnrollSecret}}{{ end }}
{{ if .Debug }}ORBIT_DEBUG=true{{ end }}
`))

func writeEnvFile(opt Options, rootPath string) error {
	envRoot := filepath.Join(rootPath, "etc", "default")
	if err := secure.MkdirAll(envRoot, constant.DefaultDirMode); err != nil {
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
  /bin/systemctl restart orbit.service 2>&1
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
