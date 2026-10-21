// Package cabundle holds the Mozilla-derived CA bundle that fleetctl installs as
// <orbit root>/certs.pem and that orbit hands to osquery as --tls_server_certs.
package cabundle

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileName is the bundle's name inside the orbit root directory.
const FileName = "certs.pem"

// Bundle is the CA bundle, generated and updated by .github/workflows/update-certs.yml.
//
//go:embed certs.pem
var Bundle []byte

// knownBundlesFile lists the SHA-256 of every bundle Fleet has shipped, one per
// line, so Refresh can tell a Fleet-installed bundle from one an admin replaced.
// update-certs.yml appends the new hash whenever the bundle changes.
//
//go:embed known_bundles.txt
var knownBundlesFile string

var knownBundles = func() map[string]struct{} {
	m := make(map[string]struct{})
	for _, line := range strings.Split(knownBundlesFile, "\n") {
		if h := strings.TrimSpace(line); h != "" {
			m[h] = struct{}{}
		}
	}
	return m
}()

// Outcome reports what Refresh did with the installed bundle.
type Outcome int

const (
	// Unchanged means the installed bundle already matches the embedded one.
	Unchanged Outcome = iota
	// Updated means the installed bundle was replaced with the embedded one.
	Updated
	// Missing means there is no installed bundle to refresh.
	Missing
	// Custom means the installed bundle was not shipped by Fleet, so it was left alone.
	Custom
)

func (o Outcome) String() string {
	switch o {
	case Unchanged:
		return "unchanged"
	case Updated:
		return "updated"
	case Missing:
		return "missing"
	case Custom:
		return "custom"
	default:
		return fmt.Sprintf("Outcome(%d)", int(o))
	}
}

// Hash returns the hex SHA-256 of b with CRLF normalized to LF, so a bundle
// checked out with autocrlf still matches the one generated on Linux.
func Hash(b []byte) string {
	sum := sha256.Sum256(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")))
	return hex.EncodeToString(sum[:])
}

// Refresh replaces <rootDir>/certs.pem with the embedded bundle when the
// installed file is an older bundle shipped by Fleet. A missing file, or one
// Fleet did not ship (an admin's own CA bundle), is left untouched.
func Refresh(rootDir string) (Outcome, error) {
	path := filepath.Join(rootDir, FileName)
	installed, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return Missing, nil
	case err != nil:
		return Unchanged, fmt.Errorf("read %s: %w", path, err)
	}

	installedHash := Hash(installed)
	if installedHash == Hash(Bundle) {
		return Unchanged, nil
	}
	if _, ok := knownBundles[installedHash]; !ok {
		return Custom, nil
	}

	if err := writeAtomic(path, Bundle); err != nil {
		return Unchanged, err
	}
	return Updated, nil
}

// writeAtomic writes data to a temp file in the same directory and renames it
// over path, so osquery never observes a partially written bundle.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, FileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	// Same mode fleetctl package uses; osquery runs as root but the file is not secret.
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
