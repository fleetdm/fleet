package file

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/saferwall/pe"
)

// ExtractPEMetadata extracts the name and version metadata from a .exe file in
// the Portable Executable (PE) format.
func ExtractPEMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	// compute its hash
	h := sha256.New()
	_, _ = io.Copy(h, tfr) // writes to a hash cannot fail

	if err := tfr.Rewind(); err != nil {
		return nil, err
	}

	// cannot use the "Fast" option, we need the data directories for the
	// resources to be available.
	pep, err := pe.New(tfr.Name(), &pe.Options{
		OmitExportDirectory:       true,
		OmitImportDirectory:       true,
		OmitExceptionDirectory:    true,
		OmitSecurityDirectory:     true,
		OmitRelocDirectory:        true,
		OmitDebugDirectory:        true,
		OmitArchitectureDirectory: true,
		OmitGlobalPtrDirectory:    true,
		OmitTLSDirectory:          true,
		OmitLoadConfigDirectory:   true,
		OmitBoundImportDirectory:  true,
		OmitIATDirectory:          true,
		OmitDelayImportDirectory:  true,
		OmitCLRHeaderDirectory:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating PE file: %w", err)
	}
	defer pep.Close()

	if err := pep.Parse(); err != nil {
		return nil, fmt.Errorf("error parsing PE file: %w", err)
	}

	resources, err := pep.ParseVersionResourcesForEntries()
	if err != nil {
		return nil, fmt.Errorf("error parsing PE version resources: %w", err)
	}
	var name, version, sfxName, sfxVersion string

	for _, e := range resources {
		productName, ok := e["ProductName"]
		if !ok {
			productName = e["productname"] // used by Opera SFX (self-extracting archive)
		}
		productVersion := strings.TrimSpace(e["ProductVersion"])
		if productName != "" {
			productName = strings.TrimSpace(productName)
			if productName == "7-Zip" {
				// This may be a 7-Zip self-extracting archive.
				sfxName = productName
				sfxVersion = productVersion
				continue
			}
			name = productName
		}
		if productVersion != "" {
			version = productVersion
		}
	}
	if name == "" && sfxName != "" {
		// If we didn't find a ProductName, we may be
		// dealing with an archive executable (e.g., if we're dealing with the 7-Zip executable itself rather than Opera)
		name = sfxName
		if sfxVersion != "" {
			version = sfxVersion
		}
	}

	return applySpecialCases(&InstallerMetadata{
		Name:       name,
		Version:    version,
		PackageIDs: []string{name},
		SHASum:     h.Sum(nil),
	}, resources), nil
}

var exeSpecialCases = map[string]func(*InstallerMetadata, []map[string]string) *InstallerMetadata{
	"Notion": func(meta *InstallerMetadata, _ []map[string]string) *InstallerMetadata {
		if meta.Version != "" {
			meta.Name = meta.Name + " " + meta.Version
		}
		return meta
	},
}

// Unlike .exe files that are the software itself (and just need to be copied
// over to the host), and unlike standard installer formats like .msi where the
// metadata defines the name under which the software will be installed, .exe
// installers may do pretty much anything they want when installing the
// software, regardless of what the .exe metadata contains.
//
// For example, the Notion .exe installer installs the app under a name like
// "Notion 3.11.1", and not just "Notion". There's no way to detect that by
// parsing the installer's metadata, so we need to apply some special cases at
// least for the most popular apps that use unusual naming.
//
// See https://github.com/fleetdm/fleet/issues/20440#issuecomment-2260500661
func applySpecialCases(meta *InstallerMetadata, resources []map[string]string) *InstallerMetadata {
	if fn := exeSpecialCases[meta.Name]; fn != nil {
		return fn(meta, resources)
	}
	return meta
}
