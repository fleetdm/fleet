package packaging

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/wix"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/josephspurrier/goversioninfo"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"
)

const wixDownload = "https://github.com/wixtoolset/wix3/releases/download/wix3112rtm/wix311-binaries.zip"

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
	updateOpt.ServerCertificatePath = opt.UpdateTLSServerCertificate

	if opt.UpdateTLSClientCertificate != "" {
		updateClientCrt, err := tls.LoadX509KeyPair(opt.UpdateTLSClientCertificate, opt.UpdateTLSClientKey)
		if err != nil {
			return "", fmt.Errorf("error loading update client certificate and key: %w", err)
		}
		updateOpt.ClientCertificate = &updateClientCrt
	}

	if opt.Desktop {
		updateOpt.Targets[constant.DesktopTUFTargetName] = update.DesktopWindowsTarget
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

	absWixDir := opt.LocalWixDir
	wineChecked := false

	// Download wix for macOS running on arm64, unless a local-wix-dir is provided.
	// We are using native MSI build on macOS arm64, instead of Docker, because the current fleetdm/wix Docker image is unreliable on macOS arm64.
	// We are looking into creating a new Docker image for macOS arm64.
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && absWixDir == "" {
		fmt.Println("Detected macOS arm64. fleetctl must use locally installed wine and wix to build the MSI package.")

		// Ensure wine is installed before downloading wix
		if err = checkWine(false); err != nil {
			return "", err
		}
		wineChecked = true

		fmt.Printf("Downloading wix from %s\n", wixDownload)
		client := fleethttp.NewClient()
		absWixDir = filepath.Join(tmpDir, "wix")
		err = downloadAndExtractZip(client, wixDownload, absWixDir)
		if err != nil {
			return "", err
		}
	}

	if absWixDir != "" {
		absWixDir, err = filepath.Abs(absWixDir)
		if err != nil {
			return "", fmt.Errorf("could not get filepath from local-wix-dir %s: %w", opt.LocalWixDir, err)
		}
		if err = checkWine(wineChecked); err != nil {
			return "", err
		}
	}
	if err := wix.Heat(tmpDir, opt.NativeTooling, absWixDir); err != nil {
		return "", fmt.Errorf("package root files: %w", err)
	}

	if err := wix.TransformHeat(filepath.Join(tmpDir, "heat.wxs")); err != nil {
		return "", fmt.Errorf("transform heat: %w", err)
	}

	if err := wix.Candle(tmpDir, opt.NativeTooling, absWixDir); err != nil {
		return "", fmt.Errorf("build package: %w", err)
	}

	if err := wix.Light(tmpDir, opt.NativeTooling, absWixDir); err != nil {
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

func checkWine(wineChecked bool) error {
	if !wineChecked && runtime.GOOS == "darwin" {
		// Ensure wine is installed
		cmd := exec.Command(wix.WineCmd, "--version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf(
				"%s failed. Is Wine installed? Creating a fleetd agent for Windows (.msi) requires Wine. To install Wine see the script here: https://fleetdm.com/install-wine %w",
				wix.WineCmd, err,
			)
		}
	}
	return nil
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

	if err := os.WriteFile(path, contents.Bytes(), 0o666); err != nil {
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
func writeManifestXML(vParts []string, orbitPath string) (string, error) {
	filePath := filepath.Join(orbitPath, "manifest.xml")

	tmplOpts := struct {
		Version string
	}{
		Version: strings.Join(vParts, "."),
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

	manifestPath, err := writeManifestXML(vParts, orbitPath)
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
	if err := vi.WriteSyso(outPath, "amd64"); err != nil {
		return fmt.Errorf("creating syso file: %w", err)
	}

	return nil
}

func downloadAndExtractZip(client *http.Client, urlPath string, destPath string) error {
	zipFile, err := os.CreateTemp("", "file.zip")
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer zipFile.Close()
	defer os.Remove(zipFile.Name())

	req, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", urlPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download %s: received http status code %s", urlPath, resp.Status)
	}
	_, err = io.Copy(zipFile, resp.Body)
	if err != nil {
		return fmt.Errorf("could not write %s: %w", zipFile.Name(), err)
	}

	// Open the downloaded file for reading. With zip, we cannot unzip directly from resp.Body
	zipReader, err := zip.OpenReader(zipFile.Name())
	if err != nil {
		return fmt.Errorf("could not open %s: %w", zipFile.Name(), err)
	}
	defer zipReader.Close()

	err = os.MkdirAll(filepath.Dir(destPath), 0o755)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", filepath.Dir(destPath), err)
	}

	// Extract each file in the archive
	for _, archiveReader := range zipReader.File {
		err = extractZipFile(archiveReader, destPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractZipFile(archiveReader *zip.File, destPath string) error {
	if archiveReader.FileInfo().Mode()&os.ModeSymlink != 0 {
		// Skip symlinks for security reasons
		return nil
	}

	// Open the file in the archive
	archiveFile, err := archiveReader.Open()
	if err != nil {
		return fmt.Errorf("could not open archive %s: %w", archiveReader.Name, err)
	}
	defer archiveFile.Close()

	// Clean the archive path to prevent extracting files outside the destination.
	archivePath := filepath.Clean(archiveReader.Name)
	if strings.HasPrefix(archivePath, ".."+string(filepath.Separator)) {
		// Skip relative paths for security reasons
		return nil
	}
	// Prepare to write the file
	finalPath := filepath.Join(destPath, archivePath)

	// Check if the file to extract is just a directory
	if archiveReader.FileInfo().IsDir() {
		err = os.MkdirAll(finalPath, 0o755)
		if err != nil {
			return fmt.Errorf("could not create directory %s: %w", finalPath, err)
		}
	} else {
		// Create all needed directories
		if os.MkdirAll(filepath.Dir(finalPath), 0o755) != nil {
			return fmt.Errorf("could not create directory %s: %w", filepath.Dir(finalPath), err)
		}

		// Prepare to write the destination file
		destinationFile, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, archiveReader.Mode())
		if err != nil {
			return fmt.Errorf("could not open file %s: %w", finalPath, err)
		}
		defer destinationFile.Close()

		// Write the destination file
		if _, err = io.Copy(destinationFile, archiveFile); err != nil {
			return fmt.Errorf("could not write file %s: %w", finalPath, err)
		}
	}
	return nil
}
