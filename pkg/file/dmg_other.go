//go:build !darwin

package file

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ExtractDMGMetadata on non-macOS platforms cannot mount DMG files (requires hdiutil).
// It returns only the SHA256 hash and logs a warning.
func ExtractDMGMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	log.Println("WARNING: DMG metadata extraction is only supported on macOS. Bundle identifier cannot be extracted on this platform.")

	h := sha256.New()
	_, _ = io.Copy(h, tfr)
	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	return &InstallerMetadata{
		SHASum: h.Sum(nil),
	}, nil
}
