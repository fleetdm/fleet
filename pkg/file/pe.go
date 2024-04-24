package file

import (
	"fmt"
	"strings"

	"github.com/saferwall/pe"
)

// ExtractPEMetadata extracts the name and version metadata from a .exe file in
// the Portable Executable (PE) format.
func ExtractPEMetadata(b []byte) (name, version string, err error) {
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
		return "", "", fmt.Errorf("error creating PE file: %w", err)
	}
	defer pep.Close()

	if err := pep.Parse(); err != nil {
		return "", "", fmt.Errorf("error parsing PE file: %w", err)
	}

	v, err := pep.ParseVersionResources()
	if err != nil {
		return "", "", fmt.Errorf("error parsing PE version resources: %w", err)
	}
	return strings.TrimSpace(v["ProductName"]), strings.TrimSpace(v["ProductVersion"]), nil
}
