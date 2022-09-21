package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// VulnsDelta compares what vulnerabilities already exists with what new vulnerabilities were found
// and returns what to insert and what to delete.
func VulnsDelta(
	found []fleet.Vulnerability,
	existing []fleet.Vulnerability,
) (toInsert []fleet.Vulnerability, toDelete []fleet.Vulnerability) {
	toDelete = make([]fleet.Vulnerability, 0)
	toInsert = make([]fleet.Vulnerability, 0)

	existingSet := make(map[string]bool)
	for _, e := range existing {
		existingSet[e.Key()] = true
	}

	foundSet := make(map[string]bool)
	for _, f := range found {
		foundSet[f.Key()] = true
	}

	for _, e := range existing {
		if _, ok := foundSet[e.Key()]; !ok {
			toDelete = append(toDelete, e)
		}
	}

	for _, f := range found {
		if _, ok := existingSet[f.Key()]; !ok {
			toInsert = append(toInsert, f)
		}
	}

	return toInsert, toDelete
}

// ProductIDsIntersect given two sets of product IDs returns whether they have any elements in common
func ProductIDsIntersect(a map[string]bool, b map[string]bool) bool {
	smallest := a
	biggest := b

	if len(a) > len(b) {
		smallest = b
		biggest = a
	}

	for pID := range smallest {
		if biggest[pID] {
			return true
		}
	}
	return false
}

// LatestFile returns the path of 'fileName' in 'dir' if the file exists, otherwise it will
// return the most recent file (based on the timestamp contained in 'fileName').
func LatestFile(fileName string, dir string) (string, error) {
	target := filepath.Join(dir, fileName)
	ext := filepath.Ext(target)

	switch _, err := os.Stat(target); {
	case err == nil:
		return target, nil
	case errors.Is(err, fs.ErrNotExist):
		files, err := os.ReadDir(dir)
		if err != nil {
			return "", err
		}

		prefix := strings.Split(fileName, "-")[0]
		var latest os.FileInfo
		for _, f := range files {
			if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ext) {
				info, err := f.Info()
				if err != nil {
					continue
				}
				if latest == nil || info.ModTime().After(latest.ModTime()) {
					latest = info
				}
			}
		}
		if latest == nil {
			return "", fmt.Errorf("file not found '%s' in '%s'", fileName, dir)
		}
		return filepath.Join(dir, latest.Name()), nil
	default:
		return "", fmt.Errorf("failed to stat %q: %w", target, err)
	}
}
