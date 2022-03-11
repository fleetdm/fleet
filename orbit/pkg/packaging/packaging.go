// Package packaging provides tools for building Orbit installation packages.
package packaging

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
)

// Options are the configurable options provided for the package.
type Options struct {
	// FleetURL is the URL to the Fleet server.
	FleetURL string
	// EnrollSecret is the enroll secret used to authenticate to the Fleet
	// server.
	EnrollSecret string
	// Version is the version number for this package.
	Version string
	// Identifier is the identifier (eg. com.fleetdm.orbit) for the package product.
	Identifier string
	// StartService is a boolean indicating whether to start a system-specific
	// background service.
	StartService bool
	// Insecure enables insecure TLS connections for the generated package.
	Insecure bool
	// SignIdentity is the codesigning identity to use (only macOS at this time)
	SignIdentity string
	// Notarize sets whether macOS packages should be Notarized.
	Notarize bool
	// FleetCertificate is a path to a server certificate to include in the package.
	FleetCertificate string
	// DisableUpdates disables auto updates on the generated package.
	DisableUpdates bool
	// OrbitChannel is the update channel to use for Orbit.
	OrbitChannel string
	// OsquerydChannel is the update channel to use for Osquery (osqueryd).
	OsquerydChannel string
	// DesktopChannel is the update channel to use for the Fleet Desktop application.
	DesktopChannel string
	// UpdateURL is the base URL of the update server (TUF repository).
	UpdateURL string
	// UpdateRoots is the root JSON metadata for update server (TUF repository).
	UpdateRoots string
	// OsqueryFlagfile is the (optional) path to a flagfile to provide to osquery.
	OsqueryFlagfile string
	// Debug determines whether to enable debug logging for the agent.
	Debug bool
	// Desktop determines whether to package the Fleet Desktop application.
	Desktop bool
}

func initializeTempDir() (string, error) {
	// Initialize directories
	tmpDir, err := ioutil.TempDir("", "orbit-package")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	if err := os.Chmod(tmpDir, 0o755); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("change temp directory permissions: %w", err)
	}
	log.Debug().Str("path", tmpDir).Msg("created temp directory")

	return tmpDir, nil
}

type UpdatesData struct {
	OrbitPath    string
	OrbitVersion string

	OsquerydPath    string
	OsquerydVersion string

	DesktopPath    string
	DesktopVersion string
}

func (u UpdatesData) String() string {
	return fmt.Sprintf(
		"orbit={%s,%s}, osqueryd={%s,%s}",
		u.OrbitPath, u.OrbitVersion,
		u.OsquerydPath, u.OsquerydVersion,
	)
}

func InitializeUpdates(updateOpt update.Options) (*UpdatesData, error) {
	localStore, err := filestore.New(filepath.Join(updateOpt.RootDirectory, "tuf-metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create local metadata store: %w", err)
	}
	updateOpt.LocalStore = localStore

	updater, err := update.New(updateOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to init updater: %w", err)
	}
	if err := updater.UpdateMetadata(); err != nil {
		return nil, fmt.Errorf("failed to update metadata: %w", err)
	}

	osquerydPath, err := updater.Get("osqueryd")
	if err != nil {
		return nil, fmt.Errorf("failed to get osqueryd: %w", err)
	}
	osquerydMeta, err := updater.Lookup("osqueryd")
	if err != nil {
		return nil, fmt.Errorf("failed to get osqueryd metadata: %w", err)
	}
	type custom struct {
		Version string `json:"version"`
	}
	var osquerydCustom custom
	if err := json.Unmarshal(*osquerydMeta.Custom, &osquerydCustom); err != nil {
		return nil, fmt.Errorf("failed to get osqueryd version: %w", err)
	}

	orbitPath, err := updater.Get("orbit")
	if err != nil {
		return nil, fmt.Errorf("failed to get orbit: %w", err)
	}
	orbitMeta, err := updater.Lookup("orbit")
	if err != nil {
		return nil, fmt.Errorf("failed to get orbit metadata: %w", err)
	}
	var orbitCustom custom
	if err := json.Unmarshal(*orbitMeta.Custom, &orbitCustom); err != nil {
		return nil, fmt.Errorf("failed to get orbit version: %w", err)
	}

	var (
		desktopPath   string
		desktopCustom custom
	)
	if _, ok := updateOpt.Targets["desktop"]; ok {
		desktopPath, err = updater.Get("desktop")
		if err != nil {
			return nil, fmt.Errorf("failed to get desktop: %w", err)
		}
		desktopMeta, err := updater.Lookup("desktop")
		if err != nil {
			return nil, fmt.Errorf("failed to get orbit metadata: %w", err)
		}
		if err := json.Unmarshal(*desktopMeta.Custom, &desktopCustom); err != nil {
			return nil, fmt.Errorf("failed to get orbit version: %w", err)
		}
	}

	if devBuildPath := os.Getenv("FLEETCTL_ORBIT_DEV_BUILD_PATH"); devBuildPath != "" {
		updater.CopyDevBuild("orbit", devBuildPath)
	}

	return &UpdatesData{
		OrbitPath:    orbitPath,
		OrbitVersion: orbitCustom.Version,

		OsquerydPath:    osquerydPath,
		OsquerydVersion: osquerydCustom.Version,

		DesktopPath:    desktopPath,
		DesktopVersion: desktopCustom.Version,
	}, nil
}

func writeSecret(opt Options, orbitRoot string) error {
	// Enroll secret
	path := filepath.Join(orbitRoot, "secret.txt")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := ioutil.WriteFile(path, []byte(opt.EnrollSecret), 0o600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func writeOsqueryFlagfile(opt Options, orbitRoot string) error {
	dstPath := filepath.Join(orbitRoot, "osquery.flags")

	if opt.OsqueryFlagfile == "" {
		// Write empty flagfile
		if err := os.WriteFile(dstPath, []byte(""), constant.DefaultFileMode); err != nil {
			return fmt.Errorf("write empty flagfile: %w", err)
		}

		return nil
	}

	if err := file.Copy(opt.OsqueryFlagfile, dstPath, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("copy flagfile: %w", err)
	}

	return nil
}

// Embed the certs file that osquery uses so that we can drop it into our installation packages.
// This file copied from https://raw.githubusercontent.com/osquery/osquery/master/tools/deployment/certs.pem
//go:embed certs.pem
var osqueryCerts []byte

func writeOsqueryCertPEM(opt Options, orbitRoot string) error {
	dstPath := filepath.Join(orbitRoot, "certs.pem")

	if err := ioutil.WriteFile(dstPath, osqueryCerts, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
