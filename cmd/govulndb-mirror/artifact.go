package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
)

// dateLayout is the snapshot date in the artifact's name. Fleet takes the lexically last asset
// matching govulndb.FilePrefix and govulndb.FileExt on the latest release, so it has to stay
// sortable.
const dateLayout = "2006-01-02"

// writeArtifact writes the snapshot into dir and returns its path.
//
// It is written to a temporary file and renamed, because the release step uploads everything in
// that directory: a crash partway through a write would otherwise attach a truncated artifact,
// which is the one outcome this command exists to prevent. The temporary name is dot-prefixed
// so that a hard kill, which skips the deferred cleanup, cannot leave behind a file the release
// step's govulndb-* glob picks up.
func writeArtifact(artifact *govulndb.Artifact, dir string, day time.Time) (string, error) {
	name := govulndb.FilePrefix + day.UTC().Format(dateLayout) + govulndb.FileExt

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+name+".tmp")
	if err != nil {
		return "", fmt.Errorf("creating a temporary file in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name())

	// os.CreateTemp makes the file 0600; the release step reads it back as an ordinary asset.
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return "", fmt.Errorf("setting the mode on %s: %w", tmp.Name(), err)
	}

	if err := encode(tmp, artifact); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("closing %s: %w", tmp.Name(), err)
	}

	path := filepath.Join(dir, name)
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", fmt.Errorf("renaming %s to %s: %w", tmp.Name(), path, err)
	}

	return path, nil
}

func encode(f *os.File, artifact *govulndb.Artifact) error {
	gz := gzip.NewWriter(f)

	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(artifact); err != nil {
		return fmt.Errorf("encoding the artifact: %w", err)
	}

	if err := gz.Close(); err != nil {
		return fmt.Errorf("finishing the gzip stream: %w", err)
	}

	return nil
}

// loadPrevious reads the previously published artifact the drop check compares against.
//
// A path that is unreadable is suspect rather than fatal: publishing without the drop check would
// be publishing blind, and the alert names the file so a human can go look.
func loadPrevious(path string) (*govulndb.Artifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, suspectf("opening the previously published artifact %s: %v", path, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, suspectf("gunzipping the previously published artifact %s: %v", path, err)
	}
	defer gz.Close()

	var artifact govulndb.Artifact
	if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
		return nil, suspectf("decoding the previously published artifact %s: %v", path, err)
	}

	return &artifact, nil
}
