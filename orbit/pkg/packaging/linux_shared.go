package packaging

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/goreleaser/nfpm/v2/rpm"
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

	if opt.Architecture != ArchAmd64 && opt.Architecture != ArchArm64 {
		return "", fmt.Errorf("Invalid architecture: %s", opt.Architecture)
	}

	// Initialize autoupdate metadata
	updateOpt := update.DefaultOptions

	updateOpt.RootDirectory = orbitRoot
	if opt.Architecture == ArchArm64 {
		updateOpt.Targets = update.LinuxArm64Targets
	} else {
		updateOpt.Targets = update.LinuxTargets
	}
	updateOpt.ServerCertificatePath = opt.UpdateTLSServerCertificate

	if opt.UpdateTLSClientCertificate != "" {
		updateClientCrt, err := tls.LoadX509KeyPair(opt.UpdateTLSClientCertificate, opt.UpdateTLSClientKey)
		if err != nil {
			return "", fmt.Errorf("error loading update client certificate and key: %w", err)
		}
		updateOpt.ClientCertificate = &updateClientCrt
	}

	if opt.Desktop {
		if opt.Architecture == ArchArm64 {
			updateOpt.Targets[constant.DesktopTUFTargetName] = update.DesktopLinuxArm64Target
		} else {
			updateOpt.Targets[constant.DesktopTUFTargetName] = update.DesktopLinuxTarget
		}
		// Override default channel with the provided value.
		updateOpt.Targets.SetTargetChannel(constant.DesktopTUFTargetName, opt.DesktopChannel)
	}

	// Override default channels with the provided values.
	updateOpt.Targets.SetTargetChannel(constant.OrbitTUFTargetName, opt.OrbitChannel)
	updateOpt.Targets.SetTargetChannel(constant.OsqueryTUFTargetName, opt.OsquerydChannel)

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

	_, isRPM := pkger.(*rpm.RPM)

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
	if err := writePostRemove(postRemovePath); err != nil {
		return "", fmt.Errorf("write postremove script: %w", err)
	}
	var postTransPath string
	if isRPM {
		postTransPath = filepath.Join(tmpDir, "posttrans.sh")
		if err := writeRPMPostTrans(opt, postTransPath); err != nil {
			return "", fmt.Errorf("write RPM posttrans script: %w", err)
		}
	}

	if opt.FleetCertificate != "" {
		if err := writeFleetServerCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet server certificate: %w", err)
		}
	}

	if opt.FleetTLSClientCertificate != "" {
		if err := writeFleetClientCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet client certificate: %w", err)
		}
	}

	if opt.UpdateTLSServerCertificate != "" {
		if err := writeUpdateServerCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write update server certificate: %w", err)
		}
	}

	if opt.UpdateTLSClientCertificate != "" {
		if err := writeUpdateClientCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write update client certificate: %w", err)
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
			Source:      "/opt/orbit/bin/orbit/" + updateOpt.Targets[constant.OrbitTUFTargetName].Platform + "/" + opt.OrbitChannel + "/orbit",
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

	rpmInfo := nfpm.RPM{}
	if _, ok := pkger.(*rpm.RPM); ok {
		rpmInfo.Scripts.PostTrans = postTransPath
	}

	// Build package
	info := &nfpm.Info{
		Name:        "fleet-osquery",
		Version:     opt.Version,
		Description: "Fleet osquery -- runtime and autoupdater",
		Arch:        opt.Architecture,
		Maintainer:  "Fleet Device Management",
		Vendor:      "Fleet Device Management",
		License:     "https://github.com/fleetdm/fleet/blob/main/LICENSE",
		Homepage:    "https://fleetdm.com",
		Overridables: nfpm.Overridables{
			Contents: contents,
			Scripts: nfpm.Scripts{
				PostInstall: postInstallPath,
				PreRemove:   preRemovePath,
				PostRemove:  postRemovePath,
			},
			RPM: rpmInfo,
		},
	}
	filename := pkger.ConventionalFileName(info)
	if opt.NativeTooling {
		filename = filepath.Join("build", filename)
	}

	if err := os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("removing existing file: %w", err)
	}

	if opt.NativeTooling {
		if err := secure.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			return "", fmt.Errorf("cannot create build dir: %w", err)
		}
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
	if err := os.WriteFile(
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
{{ if .FleetDesktopAlternativeBrowserHost }}
ORBIT_FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST={{ .FleetDesktopAlternativeBrowserHost }}
{{ end }}
{{ end }}
{{ if .Insecure }}ORBIT_INSECURE=true{{ end }}
{{ if .DisableUpdates }}ORBIT_DISABLE_UPDATES=true{{ end }}
{{ if .FleetURL }}ORBIT_FLEET_URL={{.FleetURL}}{{ end }}
{{ if .FleetCertificate }}ORBIT_FLEET_CERTIFICATE=/opt/orbit/fleet.pem{{ end }}
{{ if .UpdateTLSServerCertificate }}ORBIT_UPDATE_TLS_CERTIFICATE=/opt/orbit/update.pem{{ end }}
{{ if .EnrollSecret }}ORBIT_ENROLL_SECRET={{.EnrollSecret}}{{ end }}
{{ if .Debug }}ORBIT_DEBUG=true{{ end }}
{{ if .EnableScripts }}ORBIT_ENABLE_SCRIPTS=true{{ end }}
{{ if and (ne .HostIdentifier "") (ne .HostIdentifier "uuid") }}ORBIT_HOST_IDENTIFIER={{.HostIdentifier}}{{ end }}
{{ if .OsqueryDB }}ORBIT_OSQUERY_DB={{.OsqueryDB}}{{ end }}
{{ if .EndUserEmail }}ORBIT_END_USER_EMAIL={{.EndUserEmail}}{{ end }}
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

	if err := os.WriteFile(
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

	if err := os.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
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
	if err := os.WriteFile(path, []byte(`#!/bin/sh

systemctl stop orbit.service || true
systemctl disable orbit.service || true
pkill fleet-desktop || true
`), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writePostRemove(path string) error {
	if err := os.WriteFile(path, []byte(`#!/bin/sh

# For RPM during uninstall, $1 is 0
# For Debian during remove, $1 is "remove"
if [ "$1" = 0 ] || [ "$1" = "remove" ]; then
	rm -rf /var/lib/orbit /var/log/orbit /usr/local/bin/orbit /etc/default/orbit /usr/lib/systemd/system/orbit.service /opt/orbit
fi
`), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// postTransTemplate contains the template for RPM posttrans scriptlet (used when upgrading).
// See https://docs.fedoraproject.org/en-US/packaging-guidelines/Scriptlets/.
//
// We cannot rely on "$1" because it's always "0" for RPM < 4.12
// (see https://github.com/rpm-software-management/rpm/commit/ab069ec876639d46d12dd76dad54fd8fb762e43d)
// thus we check if orbit service is enabled, and if not we enable it (because posttrans
// will run both on "install" and "upgrade").
var postTransTemplate = template.Must(template.New("posttrans").Parse(`#!/bin/sh

# Exit on error
set -e

if ! systemctl is-enabled orbit >/dev/null 2>&1; then
	# If we have a systemd, daemon-reload away now
	if command -v systemctl >/dev/null 2>&1; then
		systemctl daemon-reload >/dev/null 2>&1
{{ if .StartService -}}
		systemctl restart orbit.service 2>&1
		systemctl enable orbit.service 2>&1
{{- end}}
	fi
fi
`))

// writeRPMPostTrans sets the posttrans scriptlets necessary to support RPM upgrades.
func writeRPMPostTrans(opt Options, path string) error {
	var contents bytes.Buffer
	if err := postTransTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := os.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
