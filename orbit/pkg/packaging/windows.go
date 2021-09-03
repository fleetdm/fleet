package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/wix"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// BuildMSI builds a Windows .msi.
func BuildMSI(opt Options) error {
	// Initialize directories
	dir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "user home directory")
	}

	packageDir := filepath.Join(dir, ".fleet", "orbit-package")
	defer os.RemoveAll(packageDir)

	filesystemRoot := filepath.Join(packageDir, "root")
	if err := secure.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create root dir")
	}
	log.Debug().Str("path", packageDir).Msg("created temp dir")
	orbitRoot := filesystemRoot
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "create orbit dir")
	}

	// Initialize autoupdate metadata

	updateOpt := update.DefaultOptions
	updateOpt.Platform = "windows"
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

	if err := writeSecret(opt, orbitRoot); err != nil {
		return errors.Wrap(err, "write enroll secret")
	}

	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return errors.Wrap(err, "write fleet certificate")
		}
	}

	if err := writeWixFile(opt, packageDir); err != nil {
		return errors.Wrap(err, "write wix file")
	}

	// Make sure permissions are permissive so that the `wine` user in the Wix Docker container can access files.
	if err := chmodRecursive(packageDir, os.ModePerm); err != nil {
		return err
	}

	if err := wix.Heat(packageDir); err != nil {
		return errors.Wrap(err, "package root files")
	}

	if err := wix.TransformHeat(filepath.Join(packageDir, "heat.wxs")); err != nil {
		return errors.Wrap(err, "transform heat")
	}

	if err := wix.Candle(packageDir); err != nil {
		return errors.Wrap(err, "build package")
	}

	if err := wix.Light(packageDir); err != nil {
		return errors.Wrap(err, "build package")
	}

	filename := fmt.Sprintf("orbit-osquery_%s.msi", opt.Version)
	if err := os.Rename(filepath.Join(packageDir, "orbit.msi"), filename); err != nil {
		return errors.Wrap(err, "rename msi")
	}
	log.Info().Str("path", filename).Msg("wrote msi package")

	return nil
}

func writeWixFile(opt Options, rootPath string) error {
	// PackageInfo is metadata for the pkg
	path := filepath.Join(rootPath, "main.wxs")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	var contents bytes.Buffer
	if err := windowsWixTemplate.Execute(&contents, opt); err != nil {
		return errors.Wrap(err, "execute template")
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), 0o666); err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}
