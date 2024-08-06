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
	return &InstallerMetadata{
		Name:    strings.TrimSpace(v["ProductName"]),
		Version: strings.TrimSpace(v["ProductVersion"]),
		SHASum:  h.Sum(nil),
	}, nil
}
