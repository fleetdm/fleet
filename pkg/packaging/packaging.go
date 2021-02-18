// package packaging provides tools for buildin Orbit installation packages.
package packaging

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Options are the configurable options provided for the package.
type Options struct {
	// FleetURL is the URL to the Fleet server.
	FleetURL string
	// EnrollSecret is the enroll secret used to authenticate to the Fleet
	// server.
	EnrollSecret string
	// Version is the version number for this package.
	Version string
	// Identifier is the identifier (eg. com.fleetdm.orbit) for the package product.
	Identifier string
	// StartService is a boolean indicating whether to start a system-specific
	// background service.
	StartService bool
	// Insecure enables insecure TLS connections for the generated package.
	Insecure bool
	// SignIdentity is the codesigning identity to use (only macOS at this time)
	SignIdentity string
	// Notarize sets whether macOS packages should be Notarized.
	Notarize bool
}

func copyFile(srcPath, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrap(err, "open src for copy")
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return errors.Wrap(err, "create dst dir for copy")
	}

	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return errors.Wrap(err, "open dst for copy")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copy src to dst")
	}

	return nil
}
