//go:build !darwin

package file

import (
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ExtractDMGMetadata on non-macOS platforms cannot mount DMG files (requires hdiutil).
// It returns the SHA256 hash with no bundle identifier or icon; callers must handle an
// empty InstallerMetadata.BundleIdentifier.
func ExtractDMGMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	h := sha256.New()
	_, _ = io.Copy(h, tfr)
	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	return &InstallerMetadata{
		SHASum: h.Sum(nil),
	}, nil
}
