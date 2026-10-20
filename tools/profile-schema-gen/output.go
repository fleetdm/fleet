package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

// readOutputVersion returns the version block of an existing output file, or nil if the file is
// missing or empty. Only the header is decoded, so an output from an older generator still pins.
func readOutputVersion(path string) (*profilevalidation.Version, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(data) == 0) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var header struct {
		Version profilevalidation.Version `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("read version from %s: %w", path, err)
	}
	return &header.Version, nil
}
