package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/wix"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// BuildMSI builds a Windows .msi.
func BuildMSI(opt Options) (string, error) {
	tmpDir, err := initializeTempDir()
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	filesystemRoot := filepath.Join(tmpDir, "root")
	if err := secure.MkdirAll(filesystemRoot, constant.DefaultDirMode); err != nil {
		return "", errors.Wrap(err, "create root dir")
	}
	orbitRoot := filesystemRoot
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return "", errors.Wrap(err, "create orbit dir")
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

	if err := InitializeUpdates(updateOpt); err != nil {
		return "", errors.Wrap(err, "initialize updates")
	}

	// Write files

	if err := writeSecret(opt, orbitRoot); err != nil {
		return "", errors.Wrap(err, "write enroll secret")
	}

	if err := writeOsqueryFlagfile(opt, orbitRoot); err != nil {
		return "", errors.Wrap(err, "write flagfile")
	}

	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return "", errors.Wrap(err, "write fleet certificate")
		}
	}

	if err := writeWixFile(opt, tmpDir); err != nil {
		return "", errors.Wrap(err, "write wix file")
	}

	if runtime.GOOS == "windows" {
		// Explicitly grant read access, otherwise within the Docker container there are permissions
		// errors.
		out, err := exec.Command("icacls", tmpDir, "/grant", "everyone:R", "/t").CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return "", errors.Wrap(err, "icacls")
		}
	}

	if err := wix.Heat(tmpDir); err != nil {
		return "", errors.Wrap(err, "package root files")
	}

	if err := wix.TransformHeat(filepath.Join(tmpDir, "heat.wxs")); err != nil {
		return "", errors.Wrap(err, "transform heat")
	}

	if err := wix.Candle(tmpDir); err != nil {
		return "", errors.Wrap(err, "build package")
	}

	if err := wix.Light(tmpDir); err != nil {
		return "", errors.Wrap(err, "build package")
	}

	filename := "fleet-osquery.msi"
	if err := file.Copy(filepath.Join(tmpDir, "orbit.msi"), filename, constant.DefaultFileMode); err != nil {
		return "", errors.Wrap(err, "rename msi")
	}
	log.Info().Str("path", filename).Msg("wrote msi package")

	return filename, nil
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
