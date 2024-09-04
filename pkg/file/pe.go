package file

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/saferwall/pe"
)

// ExtractPEMetadata extracts the name and version metadata from a .exe file in
// the Portable Executable (PE) format.
func ExtractPEMetadata(r io.Reader) (*InstallerMetadata, error) {
	h := sha256.New()
	r = io.TeeReader(r, h)
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read all content: %w", err)
	}

	// cannot use the "Fast" option, we need the data directories for the
	// resources to be available.
	pep, err := pe.NewBytes(b, &pe.Options{
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

	v, err := pep.ParseVersionResources()
	if err != nil {
		return nil, fmt.Errorf("error parsing PE version resources: %w", err)
	}
	return applySpecialCases(&InstallerMetadata{
		Name:    strings.TrimSpace(v["ProductName"]),
		Version: strings.TrimSpace(v["ProductVersion"]),
		SHASum:  h.Sum(nil),
	}, v), nil
}

var exeSpecialCases = map[string]func(*InstallerMetadata, map[string]string) *InstallerMetadata{
	"Notion": func(meta *InstallerMetadata, resources map[string]string) *InstallerMetadata {
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
func applySpecialCases(meta *InstallerMetadata, resources map[string]string) *InstallerMetadata {
	if fn := exeSpecialCases[meta.Name]; fn != nil {
		return fn(meta, resources)
	}
	return meta
}
