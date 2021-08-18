// package packaging provides tools for buildin Orbit installation packages.
package packaging

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/secure"
	"github.com/pkg/errors"
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
	// OrbitChannel is the update channel to use for Orbit.
	OrbitChannel string
	// OsquerydChannel is the update channel to use for Osquery (osqueryd).
	OsquerydChannel string
	// UpdateURL is the base URL of the update server (TUF repository).
	UpdateURL string
	// UpdateRoots is the root JSON metadata for update server (TUF repository).
	UpdateRoots string
	// Debug determines whether to enable debug logging for the agent.
	Debug bool
}

func copyFile(srcPath, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrap(err, "open src for copy")
	}
	defer src.Close()

	if err := secure.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return errors.Wrap(err, "create dst dir for copy")
	}

	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return errors.Wrap(err, "open dst for copy")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copy src to dst")
	}
	if err := dst.Sync(); err != nil {
		return errors.Wrap(err, "sync dst after copy")
	}

	return nil
}

func initializeUpdates(updateOpt update.Options) error {
	localStore, err := filestore.New(filepath.Join(updateOpt.RootDirectory, "tuf-metadata.json"))
	if err != nil {
		return errors.Wrap(err, "failed to create local metadata store")
	}
	updateOpt.LocalStore = localStore

	updater, err := update.New(updateOpt)
	if err != nil {
		return errors.Wrap(err, "failed to init updater")
	}
	if err := updater.UpdateMetadata(); err != nil {
		return errors.Wrap(err, "failed to update metadata")
	}
	osquerydPath, err := updater.Get("osqueryd", updateOpt.OsquerydChannel)
	if err != nil {
		return errors.Wrap(err, "failed to get osqueryd")
	}
	log.Debug().Str("path", osquerydPath).Msg("got osqueryd")

	orbitPath, err := updater.Get("orbit", updateOpt.OrbitChannel)
	if err != nil {
		return errors.Wrap(err, "failed to get orbit")
	}
	log.Debug().Str("path", orbitPath).Msg("got orbit")

	return nil
}

func writeSecret(opt Options, orbitRoot string) error {
	// Enroll secret
	path := filepath.Join(orbitRoot, "secret.txt")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	if err := ioutil.WriteFile(path, []byte(opt.EnrollSecret), 0600); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func chmodRecursive(path string, perm os.FileMode) error {
	return filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "walk error")
		}

		if err := os.Chmod(path, perm); err != nil {
			return errors.Wrap(err, "chmod")
		}

		return nil
	})
}
