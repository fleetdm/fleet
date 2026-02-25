package client

import (
	"fmt"
	"image"
	_ "image/png"
	"io"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ValidateIcon validates that the given file is an acceptable PNG icon.
func ValidateIcon(file io.ReadSeeker) error {
	// Check file size first
	fileSize, err := file.Seek(0, io.SeekEnd) // Seek to end to get size
	if err != nil {
		return &fleet.BadRequestError{Message: "failed to read file size"}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil { // Reset to beginning
		return &fleet.BadRequestError{Message: "failed to rewind file"}
	}

	maxSize := int64(100 * 1024) // 100KB
	if fileSize > maxSize {
		return &fleet.BadRequestError{Message: "icon must be less than 100KB"}
	}

	config, format, err := image.DecodeConfig(file)
	if err != nil || format != "png" {
		return &fleet.BadRequestError{Message: "icon must be a PNG image"}
	}

	maxWidth, maxHeight := 1024, 1024
	minWidth, minHeight := 120, 120

	if config.Width > maxWidth || config.Height > maxHeight {
		return &fleet.BadRequestError{Message: fmt.Sprintf("icon must be no larger than %dx%d pixels", maxWidth, maxHeight)}
	}
	if config.Width < minWidth || config.Height < minHeight {
		return &fleet.BadRequestError{Message: fmt.Sprintf("icon must be at least %dx%d pixels", minWidth, minHeight)}
	}
	if config.Width != config.Height {
		return &fleet.BadRequestError{Message: fmt.Sprintf("icon must be a square image (detected %dx%d pixels)", config.Width, config.Height)}
	}

	if _, err := file.Seek(0, 0); err != nil {
		return &fleet.BadRequestError{Message: "failed to rewind file"}
	}

	return nil
}
