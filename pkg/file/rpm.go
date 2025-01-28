package file

import (
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/cavaliergopher/rpm"
)

func ExtractRPMMetadata(r io.Reader) (*InstallerMetadata, error) {
	h := sha256.New()
	r = io.TeeReader(r, h)

	// Read the package headers
	pkg, err := rpm.Read(r)
	if err != nil {
		return nil, fmt.Errorf("read headers: %w", err)
	}
	// r is now positioned at the RPM payload.

	// Ensure the whole file is read to get the correct hash
	if _, err := io.Copy(io.Discard, r); err != nil {
		return nil, fmt.Errorf("read all RPM content: %w", err)
	}

	return &InstallerMetadata{
		Name:       pkg.Name(),
		Version:    pkg.Version(),
		SHASum:     h.Sum(nil),
		PackageIDs: []string{pkg.Name()},
	}, nil
}
