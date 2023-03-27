package packaging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/wix"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/josephspurrier/goversioninfo"
	"github.com/rs/zerolog/log"
)

// BuildMSI builds a Windows .msi.
// Note: this function is not safe for concurrent use
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

	updateOpt.RootDirectory = orbitRoot
	updateOpt.Targets = update.WindowsTargets

	if opt.Desktop {
		updateOpt.Targets["desktop"] = update.DesktopWindowsTarget
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

	if err := writeEventLogFile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write eventlog file: %w", err)
	}

	if err := writePowershellInstallerUtilsFile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write powershell installer utils file: %w", err)
	}

	if err := writeResourceSyso(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write VERSIONINFO: %w", err)
	}

	if err := writeWixFile(opt, tmpDir); err != nil {
		return "", fmt.Errorf("write wix file: %w", err)
	}

	if runtime.GOOS == "windows" {
		// Explicitly grant read access, otherwise within the Docker
		// container there are permissions errors.
		// "S-1-1-0" is the SID for the World/Everyone group
		// (a group that includes all users).
		out, err := exec.Command(
			"icacls", tmpDir, "/grant", "*S-1-1-0:R", "/t",
		).CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return "", fmt.Errorf("icacls: %w", err)
		}
	}

	if err := wix.Heat(tmpDir, opt.NativeTooling); err != nil {
		return "", fmt.Errorf("package root files: %w", err)
	}

	if err := wix.TransformHeat(filepath.Join(tmpDir, "heat.wxs")); err != nil {
		return "", fmt.Errorf("transform heat: %w", err)
	}

	if err := wix.Candle(tmpDir, opt.NativeTooling); err != nil {
		return "", fmt.Errorf("build package: %w", err)
	}

	if err := wix.Light(tmpDir, opt.NativeTooling); err != nil {
		return "", fmt.Errorf("build package: %w", err)
	}

	filename := "fleet-osquery.msi"
	if opt.NativeTooling {
		filename = filepath.Join("build", filename)
	}
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

func writeEventLogFile(opt Options, rootPath string) error {
	// Eventlog manifest is going to be built and dumped into working directory
	path := filepath.Join(rootPath, "osquery.man")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("event log manifest creation: %w", err)
	}

	var contents bytes.Buffer
	if err := windowsOsqueryEventLogTemplate.Execute(&contents, opt); err != nil {
		return fmt.Errorf("event log manifest creation: %w", err)
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("event log manifest creation: %w", err)
	}

	return nil
}

func writePowershellInstallerUtilsFile(opt Options, rootPath string) error {
	// Powershell installer utils file is going to be built and dumped into working directory
	path := filepath.Join(rootPath, "installer_utils.ps1")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("powershell installer utils location creation: %w", err)
	}

	var contents bytes.Buffer
	if err := windowsPSInstallerUtils.Execute(&contents, opt); err != nil {
		return fmt.Errorf("powershell installer utils transform: %w", err)
	}

	if err := ioutil.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("powershell installer utils file write: %w", err)
	}

	return nil
}

// writeManifestXML creates the manifest.xml file used when generating the 'resource.syso' metadata
// (see writeResourceSyso). Returns the path of the newly created file.
func writeManifestXML(opt Options, orbitPath string) (string, error) {
	filePath := filepath.Join(orbitPath, "manifest.xml")
	var contents bytes.Buffer
	if err := manifestXMLTemplate.Execute(&contents, opt); err != nil {
		return "", fmt.Errorf("parsing manifest XML: %w", err)
	}
	if err := ioutil.WriteFile(filePath, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return filePath, nil
}

// writeVersionInfoJSON creates the versioninfo.json used when generating the 'resource.syso'
// metadata (see writeResourceSyso). Returns a pointer to a VersionInfo struct with the proper info.
func writeVersionInfoJSON(opt Options, orbitPath string, manifestPath string) (*goversioninfo.VersionInfo, error) {
	vParts := strings.Split(opt.Version, ".")
	if len(vParts) < 3 {
		return nil, fmt.Errorf("invalid version: %s", opt.Version)
	}

	// Append a default 'build' number if the version str contains none.
	if len(vParts) <= 4 {
		vParts = append(vParts, "0")
	}

	tmplOpts := struct {
		Version      string
		VersionParts []string
		Copyright    string
		ManifestPath string
	}{
		Version:      opt.Version,
		VersionParts: vParts,
		Copyright:    fmt.Sprintf("%d Fleet Device Management Inc.", time.Now().Year()),
		ManifestPath: manifestPath,
	}

	var contents bytes.Buffer
	if err := versionInfoJSONTemplate.Execute(&contents, tmplOpts); err != nil {
		return nil, fmt.Errorf("parsing versioninfo.json template: %w", err)
	}

	result := &goversioninfo.VersionInfo{}
	if err := result.ParseJSON(contents.Bytes()); err != nil {
		return nil, fmt.Errorf("parsing versioninfo.json: %w", err)
	}

	return result, nil
}

// writeResourceSyso creates a syso file which contains the required Microsoft Windows Version Information
func writeResourceSyso(opt Options, orbitPath string) error {
	if err := secure.MkdirAll(orbitPath, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	outPath := filepath.Join(orbitPath, "resource.syso")

	// Create manifest.xml
	manifestPath, err := writeManifestXML(opt, orbitPath)
	if err != nil {
		return fmt.Errorf("creating manifest.xml: %w", err)
	}

	// Create vertsioninfo.json
	vi, err := writeVersionInfoJSON(opt, orbitPath, manifestPath)
	if err != nil {
		return fmt.Errorf("creating versioninfo.json: %w", err)
	}

	// Build syso file
	vi.Build()
	vi.Walk()

	// Output syso file
	if err := vi.WriteSyso(outPath, "amd64"); err != nil {
		return fmt.Errorf("creating syso file: %w", err)
	}

	return nil
}
