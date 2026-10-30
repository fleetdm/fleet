// Package cabundle holds the CA bundle that fleetctl installs as <orbit root>/certs.pem,
// which orbit passes to osquery as --tls_server_certs when no Fleet certificate is set.
package cabundle

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the name of the bundle in the orbit root directory.
const FileName = "certs.pem"

// Bundle is generated and updated by .github/workflows/update-certs.yml.
//
//go:embed certs.pem
var Bundle []byte

// Refresh overwrites <rootDir>/certs.pem with Bundle if they differ, and reports whether it did.
// Package upgrades and auto-updates never rewrite the installed file, so without this hosts keep
// the bundle of the fleetctl version that built their package. A missing file is left missing.
//
// Callers must not call it when a Fleet certificate (--fleet-certificate) is configured.
func Refresh(rootDir string) (bool, error) {
	path := filepath.Join(rootDir, FileName)
	installed, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	if bytes.Equal(installed, Bundle) {
		return false, nil
	}
	if err := writeAtomic(path, Bundle); err != nil {
		return false, err
	}
	return true, nil
}

// writeAtomic writes to a temp file and renames it over path, so osquery never reads a partial bundle.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), FileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		// Same mode fleetctl package uses; the bundle holds only public CA certificates.
		err = os.Chmod(tmpPath, 0o644) //nolint:gosec // G302
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
