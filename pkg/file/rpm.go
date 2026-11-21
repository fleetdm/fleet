package file

import (
	"fmt"
	"io"

	"github.com/cavaliergopher/rpm"
)

func ExtractRPMMetadata(r io.Reader) (*InstallerMetadata, error) {
	r, h := newSequentialHasher(r)

	// Read the package headers
	pkg, err := rpm.Read(r)
	if err != nil {
		return nil, fmt.Errorf("read headers: %w", err)
	}
	// r is now positioned at the RPM payload.

	sum, err := h.finish()
	if err != nil {
		return nil, fmt.Errorf("read all RPM content: %w", err)
	}

	return &InstallerMetadata{
		Name:       pkg.Name(),
		Version:    pkg.Version(),
		SHASum:     sum,
		PackageIDs: []string{pkg.Name()},
	}, nil
}
