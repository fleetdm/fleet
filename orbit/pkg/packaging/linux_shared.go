package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/semver"
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

	rootDir := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(rootDir, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create root dir: %w", err)
	}
	orbitRoot := filepath.Join(rootDir, "opt", "orbit")
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create orbit dir: %w", err)
	}

	// Initialize autoupdate metadata
	updateOpt := update.DefaultOptions

	updateOpt.RootDirectory = orbitRoot
	updateOpt.Targets = update.LinuxTargets

	if opt.Desktop {
		updateOpt.Targets["desktop"] = update.DesktopLinuxTarget
		// Override default channel with the provided value.
		updateOpt.Targets.SetTargetChannel("desktop", opt.DesktopChannel)
	}

	// Override default channels with the provided values.
	updateOpt.Targets.SetTargetChannel("orbit", opt.OrbitChannel)
	updateOpt.Targets.SetTargetChannel("osqueryd", opt.OsquerydChannel)

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

	varLibSymlink := false
	if orbitSemVer, err := semver.NewVersion(updatesData.OrbitVersion); err == nil {
		if orbitSemVer.LessThan(semver.MustParse("0.0.11")) {
			varLibSymlink = true
		}
	}
	// If err != nil we assume non-legacy Orbit.

	// Write files

	if err := writeSystemdUnit(opt, rootDir); err != nil {
		return "", fmt.Errorf("write systemd unit: %w", err)
	}

	if err := writeEnvFile(opt, rootDir); err != nil {
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
	preRemovePath := filepath.Join(tmpDir, "preremove.sh")
	if err := writePreRemove(opt, preRemovePath); err != nil {
		return "", fmt.Errorf("write preremove script: %w", err)
	}
	postRemovePath := filepath.Join(tmpDir, "postremove.sh")
	if err := writePostRemove(opt, postRemovePath); err != nil {
		return "", fmt.Errorf("write postremove script: %w", err)
	}

	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet certificate: %w", err)
		}
	}

	// Pick up all file contents

	contents := files.Contents{
		&files.Content{
			Source:      filepath.Join(rootDir, "**"),
			Destination: "/",
		},
		// Symlink current into /opt/orbit/bin/orbit/orbit
		&files.Content{
			Source:      "/opt/orbit/bin/orbit/linux/" + opt.OrbitChannel + "/orbit",
			Destination: "/opt/orbit/bin/orbit/orbit",
			Type:        "symlink",
			FileInfo: &files.ContentFileInfo{
				Mode: constant.DefaultExecutableMode | os.ModeSymlink,
			},
		},
		// Symlink current into /usr/local/bin
		&files.Content{
			Source:      "/opt/orbit/bin/orbit/orbit",
			Destination: "/usr/local/bin/orbit",
			Type:        "symlink",
			FileInfo: &files.ContentFileInfo{
				Mode: constant.DefaultExecutableMode | os.ModeSymlink,
			},
		},
	}

	// Add empty folders to be created.
	for _, emptyFolder := range []string{"/var/log/osquery", "/var/log/orbit"} {
		contents = append(contents, (&files.Content{
			Destination: emptyFolder,
			Type:        "dir",
		}).WithFileInfoDefaults())
	}

	if varLibSymlink {
		contents = append(contents,
			// Symlink needed to support old versions of orbit.
			&files.Content{
				Source:      "/opt/orbit",
				Destination: "/var/lib/orbit",
				Type:        "symlink",
				FileInfo: &files.ContentFileInfo{
					Mode: os.ModeSymlink,
				},
			})
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
			Scripts: nfpm.Scripts{
				PostInstall: postInstallPath,
				PreRemove:   preRemovePath,
				PostRemove:  postRemovePath,
			},
		},
	}
	filename := pkger.ConventionalFileName(info)
	if opt.NativeTooling {
		filename = filepath.Join("build", filename)
	}

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
ExecStart=/opt/orbit/bin/orbit/orbit
Restart=always
RestartSec=1
KillMode=control-group
KillSignal=SIGTERM
CPUQuota=20%

[Install]
WantedBy=multi-user.target
`),
		constant.DefaultSystemdUnitMode,
	); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

var envTemplate = template.Must(template.New("env").Parse(`
ORBIT_UPDATE_URL={{ .UpdateURL }}
ORBIT_ORBIT_CHANNEL={{ .OrbitChannel }}
ORBIT_OSQUERYD_CHANNEL={{ .OsquerydChannel }}
ORBIT_UPDATE_INTERVAL={{ .OrbitUpdateInterval }}
{{ if .Desktop }}
ORBIT_FLEET_DESKTOP=true
ORBIT_DESKTOP_CHANNEL={{ .DesktopChannel }}
{{ end }}
{{ if .Insecure }}ORBIT_INSECURE=true{{ end }}
{{ if .DisableUpdates }}ORBIT_DISABLE_UPDATES=true{{ end }}
{{ if .FleetURL }}ORBIT_FLEET_URL={{.FleetURL}}{{ end }}
{{ if .FleetCertificate }}ORBIT_FLEET_CERTIFICATE=/opt/orbit/fleet.pem{{ end }}
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

var postInstallTemplate = template.Must(template.New("postinstall").Parse(`#!/bin/sh

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

func writePreRemove(opt Options, path string) error {
	// We add `|| true` in case the service is not running
	// or has been manually disabled already. Otherwise,
	// uninstallation fails.
	//
	// "pkill fleet-desktop" is required because the application
	// runs as user (separate from sudo command that launched it),
	// so on some systems it's not killed properly.
	if err := ioutil.WriteFile(path, []byte(`#!/bin/sh

systemctl stop orbit.service || true
systemctl disable orbit.service || true
pkill fleet-desktop || true
`), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writePostRemove(opt Options, path string) error {
	if err := ioutil.WriteFile(path, []byte(`#!/bin/sh

rm -rf /var/lib/orbit /var/log/orbit /usr/local/bin/orbit /etc/default/orbit /usr/lib/systemd/system/orbit.service /opt/orbit
`), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
