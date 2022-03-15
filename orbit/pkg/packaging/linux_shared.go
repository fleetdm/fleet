package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/rs/zerolog/log"
)

func buildNFPM(opt Options, pkger nfpm.Packager) (string, error) {
	// Initialize directories
	tmpDir, err := initializeTempDir()
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create root dir: %w", err)
	}
	orbitRoot := filepath.Join(filesystemRoot, "var", "lib", "orbit")
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create orbit dir: %w", err)
	}

	// Initialize autoupdate metadata

	updateOpt := update.DefaultOptions

	updateOpt.RootDirectory = orbitRoot
	updateOpt.Targets = update.LinuxTargets

	// Override default channels with the provided values.
	orbit := updateOpt.Targets["orbit"]
	orbit.Channel = opt.OrbitChannel
	updateOpt.Targets["orbit"] = orbit
	osqueryd := updateOpt.Targets["osqueryd"]
	osqueryd.Channel = opt.OsquerydChannel
	updateOpt.Targets["osqueryd"] = osqueryd

	updateOpt.ServerURL = opt.UpdateURL
	if opt.UpdateRoots != "" {
		updateOpt.RootKeys = opt.UpdateRoots
	}

	updatesData, err := InitializeUpdates(updateOpt)
	if err != nil {
		return "", fmt.Errorf("initialize updates: %w", err)
	}
	log.Debug().Stringer("data", updatesData).Msg("updates initialized")
	if opt.Version == "" {
		// We set the package version to orbit's latest version.
		opt.Version = updatesData.OrbitVersion
	}

	// Write files

	if err := writeSystemdUnit(opt, filesystemRoot); err != nil {
		return "", fmt.Errorf("write systemd unit: %w", err)
	}

	if err := writeEnvFile(opt, filesystemRoot); err != nil {
		return "", fmt.Errorf("write env file: %w", err)
	}

	if err := writeOsqueryFlagfile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write flagfile: %w", err)
	}

	if err := writeOsqueryCertPEM(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write certs.pem: %w", err)
	}

	postInstallPath := filepath.Join(tmpDir, "postinstall.sh")
	if err := writePostInstall(opt, postInstallPath); err != nil {
		return "", fmt.Errorf("write postinstall script: %w", err)
	}

	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet certificate: %w", err)
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
		return "", fmt.Errorf("glob contents: %w", err)
	}
	for _, c := range contents {
		log.Debug().Interface("file", c).Msg("added file")
	}

	// Build package

	info := &nfpm.Info{
		Name:        "fleet-osquery",
		Version:     opt.Version,
		Description: "Fleet osquery -- runtime and autoupdater",
		Arch:        "amd64",
		Maintainer:  "Fleet Engineers <engineering@fleetdm.com>",
		Homepage:    "https://fleetdm.com",
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
		return "", fmt.Errorf("open output file: %w", err)
	}
	defer out.Close()

	if err := pkger.Package(info, out); err != nil {
		return "", fmt.Errorf("write package: %w", err)
	}
	if err := out.Sync(); err != nil {
		return "", fmt.Errorf("sync output file: %w", err)
	}
	log.Info().Str("path", filename).Msg("wrote package")

	return filename, nil
}

func writeSystemdUnit(opt Options, rootPath string) error {
	systemdRoot := filepath.Join(rootPath, "usr", "lib", "systemd", "system")
	if err := secure.MkdirAll(systemdRoot, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("create systemd dir: %w", err)
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
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

var envTemplate = template.Must(template.New("env").Parse(`
ORBIT_UPDATE_URL={{ .UpdateURL }}
ORBIT_ORBIT_CHANNEL={{ .OrbitChannel }}
ORBIT_OSQUERYD_CHANNEL={{ .OsquerydChannel }}
{{ if .Insecure }}ORBIT_INSECURE=true{{ end }}
{{ if .DisableUpdates }}ORBIT_DISABLE_UPDATES=true{{ end }}
{{ if .FleetURL }}ORBIT_FLEET_URL={{.FleetURL}}{{ end }}
{{ if .FleetCertificate }}ORBIT_FLEET_CERTIFICATE=/var/lib/orbit/fleet.pem{{ end }}
{{ if .EnrollSecret }}ORBIT_ENROLL_SECRET={{.EnrollSecret}}{{ end }}
{{ if .Debug }}ORBIT_DEBUG=true{{ end }}
`))

func writeEnvFile(opt Options, rootPath string) error {
	envRoot := filepath.Join(rootPath, "etc", "default")
	if err := secure.MkdirAll(envRoot, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("create env dir: %w", err)
	}

	var contents bytes.Buffer
	if err := envTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := ioutil.WriteFile(
		filepath.Join(envRoot, "orbit"),
		contents.Bytes(),
		constant.DefaultFileMode,
	); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

var postInstallTemplate = template.Must(template.New("postinstall").Parse(`
#!/bin/sh

# Exit on error
set -e

# If we have a systemd, daemon-reload away now
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1
{{ if .StartService -}}
  systemctl restart orbit.service 2>&1
  systemctl enable orbit.service 2>&1
{{- end}}
fi
`))

func writePostInstall(opt Options, path string) error {
	var contents bytes.Buffer
	if err := postInstallTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
