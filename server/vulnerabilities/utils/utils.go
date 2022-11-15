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

// RecentVulns filters vulnerabilities based on whether the vulnerability cve is contained in 'meta'.
// Returns the filtered vulnerabilities and their meta data.
func RecentVulns[T fleet.Vulnerability](
	vulns []T,
	meta []fleet.CVEMeta,
) ([]T, map[string]fleet.CVEMeta) {
	if len(vulns) == 0 {
		return nil, nil
	}

	recent := make(map[string]fleet.CVEMeta)
	for _, r := range meta {
		recent[r.CVE] = r
	}

	seen := make(map[string]bool)
	var r []T

	for _, v := range vulns {
		if _, ok := recent[v.GetCVE()]; ok && !seen[v.Key()] {
			seen[v.Key()] = true
			r = append(r, v)
		}
	}

	return r, recent
}

func BatchProcess[T fleet.Vulnerability](
	values map[string]T,
	dsFunc func(v []T) error,
	batchSize int,
) error {
	if len(values) == 0 {
		return nil
	}

	bSize := batchSize
	if bSize > len(values) {
		bSize = len(values)
	}

	buffer := make([]T, bSize)
	var offset, i int
	for _, v := range values {
		buffer[offset] = v
		offset++
		i++

		// Consume buffer if full or if we are at the last iteration
		if offset == bSize || i >= len(values) {
			err := dsFunc(buffer[:offset])
			if err != nil {
				return err
			}
			offset = 0
		}
	}
	return nil
}

// VulnsDelta compares what vulnerabilities already exists with what new vulnerabilities were found
// and returns what to insert and what to delete.
func VulnsDelta[T fleet.Vulnerability](
	found []T,
	existing []T,
) (toInsert []T, toDelete []T) {
	toDelete = make([]T, 0)
	toInsert = make([]T, 0)

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
			// existing not in found, delete
			toDelete = append(toDelete, e)
		}
	}

	for _, f := range found {
		if _, ok := existingSet[f.Key()]; !ok {
			// found not in existing, insert
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
