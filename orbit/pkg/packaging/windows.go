package packaging

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/msi"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/josephspurrier/goversioninfo"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"
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
	if opt.Architecture == ArchAmd64 {
		updateOpt.Targets = update.WindowsTargets
	} else {
		updateOpt.Targets = update.WindowsArm64Targets
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
			updateOpt.Targets[constant.DesktopTUFTargetName] = update.DesktopWindowsArm64Target
		} else {
			updateOpt.Targets[constant.DesktopTUFTargetName] = update.DesktopWindowsTarget
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

	orbitVersion := updatesData.OrbitVersion
	if !strings.HasPrefix(orbitVersion, "v") {
		orbitVersion = "v" + orbitVersion
	}
	// v1.28.0 introduced configurable END_USER_EMAIL property for MSI package: https://github.com/fleetdm/fleet/issues/19219
	if semver.Compare(orbitVersion, "v1.28.0") >= 0 {
		opt.EnableEndUserEmailProperty = true
	}
	// v1.55.0 introduced EUA_TOKEN property for MSI package: https://github.com/fleetdm/fleet/issues/41379
	if semver.Compare(orbitVersion, "v1.55.0") >= 0 {
		opt.EnableEUATokenProperty = true
	}

	// Write files

	// Don't write the dummy enroll secret to secret.txt. The installer will supply it on install and writing it
	// here can cause "dummy" to attempt to be used as the secret, which will of course fail
	if opt.EnrollSecret != constant.UnusedFlagKeyword {
		if err := writeSecret(opt, orbitRoot); err != nil {
			return "", fmt.Errorf("write enroll secret: %w", err)
		}
	}

	if err := writeOsqueryFlagfile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write flagfile: %w", err)
	}

	if err := writeOsqueryCertPEM(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write certs.pem: %w", err)
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

	if err := writeEventLogFile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write eventlog file: %w", err)
	}

	if err := writePowershellInstallerUtilsFile(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write powershell installer utils file: %w", err)
	}

	if err := writeResourceSyso(opt, orbitRoot); err != nil {
		return "", fmt.Errorf("write VERSIONINFO: %w", err)
	}

	// The MSI is assembled in pure Go: no Docker, Wine, or WiX toolset
	// required. The legacy flags selecting alternative WiX toolchains are
	// accepted but no longer change how the installer is built.
	if opt.LocalWixDir != "" {
		fmt.Println("NOTE: --local-wix-dir is deprecated: the MSI is now built in pure Go and WiX is no longer used.")
	}

	msiOpt := msi.Options{
		Architecture:                       opt.Architecture,
		Version:                            opt.Version,
		FleetURL:                           opt.FleetURL,
		EnrollSecret:                       opt.EnrollSecret != "",
		FleetCertificate:                   opt.FleetCertificate != "",
		Insecure:                           opt.Insecure,
		Debug:                              opt.Debug,
		UpdateURL:                          opt.UpdateURL,
		UpdateTLSServerCertificate:         opt.UpdateTLSServerCertificate != "",
		DisableUpdates:                     opt.DisableUpdates,
		Desktop:                            opt.Desktop,
		DesktopChannel:                     opt.DesktopChannel,
		OrbitChannel:                       opt.OrbitChannel,
		OsquerydChannel:                    opt.OsquerydChannel,
		FleetDesktopAlternativeBrowserHost: opt.FleetDesktopAlternativeBrowserHost,
		HostIdentifier:                     opt.HostIdentifier,
		EnableScripts:                      opt.EnableScripts,
		EnableEndUserEmailProperty:         opt.EnableEndUserEmailProperty,
		EndUserEmail:                       opt.EndUserEmail,
		EnableEUATokenProperty:             opt.EnableEUATokenProperty,
		OsqueryDB:                          opt.OsqueryDB,
		DisableSetupExperience:             opt.DisableSetupExperience,
		BypassEndUserAuth:                  opt.BypassEndUserAuth,
		OrbitUpdateInterval:                opt.OrbitUpdateInterval.String(),
		NativePlatform:                     opt.NativePlatform,
	}
	if err := msi.Build(msiOpt, filesystemRoot, filepath.Join(tmpDir, "orbit.msi")); err != nil {
		return "", fmt.Errorf("build msi: %w", err)
	}

	filename := "fleet-osquery.msi"
	if opt.CustomOutfile != "" {
		filename = opt.CustomOutfile
	}
	if opt.Architecture == ArchArm64 {
		filename = "fleet-osquery-arm64.msi"
	}
	if opt.NativeTooling {
		filename = filepath.Join("build", filename)
	}
	if err := file.Copy(filepath.Join(tmpDir, "orbit.msi"), filename, constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("rename msi: %w", err)
	}
	log.Info().Str("path", filename).Msg("wrote msi package")

	return filename, nil
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

	if err := os.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
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

	if err := os.WriteFile(path, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("powershell installer utils file write: %w", err)
	}

	return nil
}

// writeManifestXML creates the manifest.xml file used when generating the 'resource_windows.syso' metadata
// (see writeResourceSyso). Returns the path of the newly created file.
func writeManifestXML(vParts []string, orbitPath string, arch string) (string, error) {
	filePath := filepath.Join(orbitPath, "manifest.xml")

	tmplOpts := struct {
		Version string
		Arch    string
	}{
		Version: strings.Join(vParts, "."),
		Arch:    arch,
	}

	var contents bytes.Buffer
	if err := ManifestXMLTemplate.Execute(&contents, tmplOpts); err != nil {
		return "", fmt.Errorf("parsing manifest.xml template: %w", err)
	}

	if err := os.WriteFile(filePath, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing manifest.xml file: %w", err)
	}

	return filePath, nil
}

// createVersionInfo returns a VersionInfo struct pointer to be used to generate the 'resource_windows.syso'
// metadata file (see writeResourceSyso).
func createVersionInfo(vParts []string, manifestPath string) (*goversioninfo.VersionInfo, error) {
	vIntParts := make([]int, 0, len(vParts))
	for _, p := range vParts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing version part %s: %w", p, err)
		}
		vIntParts = append(vIntParts, v)
	}
	version := strings.Join(vParts, ".")
	copyright := fmt.Sprintf("%d Fleet Device Management Inc.", time.Now().Year())

	// Taken from https://github.com/josephspurrier/goversioninfo/blob/master/testdata/resource/versioninfo.json
	langID, err := strconv.ParseUint("0409", 16, 16)
	if err != nil {
		return nil, errors.New("invalid LangID")
	}
	// Taken from https://github.com/josephspurrier/goversioninfo/blob/master/testdata/resource/versioninfo.json
	charsetID, err := strconv.ParseUint("04B0", 16, 16)
	if err != nil {
		return nil, errors.New("invalid charsetID")
	}

	result := goversioninfo.VersionInfo{
		FixedFileInfo: goversioninfo.FixedFileInfo{
			FileVersion: goversioninfo.FileVersion{
				Major: vIntParts[0],
				Minor: vIntParts[1],
				Patch: vIntParts[2],
				Build: vIntParts[3],
			},
			ProductVersion: goversioninfo.FileVersion{
				Major: vIntParts[0],
				Minor: vIntParts[1],
				Patch: vIntParts[2],
				Build: vIntParts[3],
			},
			FileFlagsMask: "3f",
			FileFlags:     "00",
			FileOS:        "040004",
			FileType:      "01",
			FileSubType:   "00",
		},
		StringFileInfo: goversioninfo.StringFileInfo{
			Comments:         "Fleet osquery",
			CompanyName:      "Fleet Device Management (fleetdm.com)",
			FileDescription:  "Fleet osquery installer",
			FileVersion:      version,
			InternalName:     "",
			LegalCopyright:   copyright,
			LegalTrademarks:  "",
			OriginalFilename: "",
			PrivateBuild:     "",
			ProductName:      "Fleet osquery",
			ProductVersion:   version,
			SpecialBuild:     "",
		},
		VarFileInfo: goversioninfo.VarFileInfo{
			Translation: goversioninfo.Translation{
				LangID:    goversioninfo.LangID(langID),
				CharsetID: goversioninfo.CharsetID(charsetID),
			},
		},
		IconPath:     "",
		ManifestPath: manifestPath,
	}

	return &result, nil
}

// SanitizeVersion returns the version parts (Major, Minor, Patch and Build), filling the Build part
// with '0' if missing. Will error out if the version string is missing the Major, Minor or
// Patch part(s).
// It supports the version with a pre-release part (e.g. 1.2.3-1) and returns it as the Build number.
func SanitizeVersion(version string) ([]string, error) {
	vParts := strings.Split(version, ".")
	if len(vParts) < 3 {
		return nil, errors.New("invalid version string")
	}
	if len(vParts) == 3 && strings.Contains(vParts[2], "-") {
		parts := strings.SplitN(vParts[2], "-", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid patch and pre-release version: %s", vParts[2])
		}
		patch, preRelease := parts[0], parts[1]
		vParts = []string{vParts[0], vParts[1], patch, preRelease}
	}

	if len(vParts) < 4 {
		vParts = append(vParts, "0")
	}

	return vParts[:4], nil
}

// writeResourceSyso creates the 'resource_windows.syso' metadata file which contains the required Microsoft
// Windows Version Information
func writeResourceSyso(opt Options, orbitPath string) error {
	if err := secure.MkdirAll(orbitPath, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	vParts, err := SanitizeVersion(opt.Version)
	if err != nil {
		return fmt.Errorf("invalid version %s: %w", opt.Version, err)
	}

	manifestPath, err := writeManifestXML(vParts, orbitPath, opt.Architecture)
	if err != nil {
		return fmt.Errorf("creating manifest.xml: %w", err)
	}
	defer os.RemoveAll(manifestPath)

	vi, err := createVersionInfo(vParts, manifestPath)
	if err != nil {
		return fmt.Errorf("parsing versioninfo: %w", err)
	}

	vi.Build()
	vi.Walk()

	outPath := filepath.Join(orbitPath, "resource_windows.syso")
	if err := vi.WriteSyso(outPath, opt.Architecture); err != nil {
		return fmt.Errorf("creating syso file: %w", err)
	}

	return nil
}
