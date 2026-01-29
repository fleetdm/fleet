// ABOUTME: Adobe Acrobat-specific version handling for ZIP-packaged installers
// ABOUTME: Transforms winget versions to "latest" and provides ZIP extraction utilities

package externalrefs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// AdobeVersionToLatest transforms Adobe Acrobat's winget version to "latest"
// because Adobe's download URL is not version-pinned and the actual installer
// version differs from what winget reports.
func AdobeVersionToLatest(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Transform any winget version to "latest" so validation will extract
	// the actual version from the MSI inside the ZIP
	app.Version = "latest"
	return app, nil
}

// ExtractVersionFromAdobeZIP extracts the MSI version from Adobe's ZIP-packaged installer.
// Adobe distributes Acrobat as a ZIP containing "Adobe Acrobat/AcroPro.msi".
// This function:
// 1. Opens the ZIP archive
// 2. Finds the AcroPro.msi file
// 3. Extracts and reads the MSI metadata
// 4. Returns the actual installer version
//
// This is called during validation when version="latest" to get the real version.
func ExtractVersionFromAdobeZIP(tfr *fleet.TempFileReader) (string, error) {
	// Create temp file for ZIP processing
	tempZIP, err := os.CreateTemp("", "adobe-installer-*.zip")
	if err != nil {
		return "", fmt.Errorf("create temp zip: %w", err)
	}
	defer os.Remove(tempZIP.Name())
	defer tempZIP.Close()

	// Copy ZIP content to temp file
	if _, err := io.Copy(tempZIP, tfr); err != nil {
		return "", fmt.Errorf("copy zip: %w", err)
	}

	// Rewind original reader for potential reuse
	if err := tfr.Rewind(); err != nil {
		return "", fmt.Errorf("rewind tfr: %w", err)
	}

	// Open ZIP archive
	zipReader, err := zip.OpenReader(tempZIP.Name())
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}
	defer zipReader.Close()

	// Find MSI file inside ZIP
	// Adobe's structure is: Adobe Acrobat/AcroPro.msi
	var msiFile *zip.File
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(f.Name), ".msi") {
			msiFile = f
			break
		}
	}

	if msiFile == nil {
		return "", fmt.Errorf("no MSI file found in Adobe ZIP archive")
	}

	// Extract MSI to temp file
	rc, err := msiFile.Open()
	if err != nil {
		return "", fmt.Errorf("open msi in zip: %w", err)
	}
	defer rc.Close()

	tempMSI, err := os.CreateTemp("", "adobe-installer-*.msi")
	if err != nil {
		return "", fmt.Errorf("create temp msi: %w", err)
	}
	defer os.Remove(tempMSI.Name())
	defer tempMSI.Close()

	if _, err := io.Copy(tempMSI, rc); err != nil {
		return "", fmt.Errorf("extract msi: %w", err)
	}

	// Rewind MSI file for metadata extraction
	if _, err := tempMSI.Seek(0, 0); err != nil {
		return "", fmt.Errorf("rewind msi: %w", err)
	}

	// Create TempFileReader for MSI and extract metadata
	msiTFR := &fleet.TempFileReader{File: tempMSI}
	meta, err := file.ExtractInstallerMetadata(msiTFR)
	if err != nil {
		return "", fmt.Errorf("extract msi metadata from %s: %w", filepath.Base(msiFile.Name), err)
	}

	return meta.Version, nil
}
