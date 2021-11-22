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
		return "", fmt.Errorf("create root dir: %w", err)
	}
	orbitRoot := filesystemRoot
	if err := secure.MkdirAll(orbitRoot, constant.DefaultDirMode); err != nil {
		return "", fmt.Errorf("create orbit dir: %w", err)
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
		return "", fmt.Errorf("initialize updates: %w", err)
	}

	// Write files

	if err := writeSecret(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write enroll secret: %w", err)
	}

	if err := writeOsqueryFlagfile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write flagfile: %w", err)
	}

	if err := writeOsqueryCertPEM(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write certs.pem: %w", err)
	}

	if opt.FleetCertificate != "" {
		if err := writeCertificate(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write fleet certificate: %w", err)
		}
	}

	if err := writeWixFile(opt, tmpDir); err != nil {
		return "", fmt.Errorf("write wix file: %w", err)
	}

	if runtime.GOOS == "windows" {
		// Explicitly grant read access, otherwise within the Docker container there are permissions
		// errors.
		out, err := exec.Command("icacls", tmpDir, "/grant", "everyone:R", "/t").CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return "", fmt.Errorf("icacls: %w", err)
		}
	}

	if err := wix.Heat(tmpDir); err != nil {
		return "", fmt.Errorf("package root files: %w", err)
	}

	if err := wix.TransformHeat(filepath.Join(tmpDir, "heat.wxs")); err != nil {
		return "", fmt.Errorf("transform heat: %w", err)
	}

	if err := wix.Candle(tmpDir); err != nil {
		return "", fmt.Errorf("build package: %w", err)
	}

	if err := wix.Light(tmpDir); err != nil {
		return "", fmt.Errorf("build package: %w", err)
	}

	filename := "fleet-osquery.msi"
	if err := file.Copy(filepath.Join(tmpDir, "orbit.msi"), filename, constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("rename msi: %w", err)
	}
	log.Info().Str("path", filename).Msg("wrote msi package")

	return filename, nil
}

func writeWixFile(opt Options, rootPath string) error {
	// PackageInfo is metadata for the pkg
	path := filepath.Join(rootPath, "main.wxs")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var contents bytes.Buffer
	if err := windowsWixTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), 0o666); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
