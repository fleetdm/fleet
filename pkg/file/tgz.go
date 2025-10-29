package file

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
)

// ValidateTarball confirms that a .tar.gz file is valid, then returns empty installer metadata for fallback
func ValidateTarball(r io.Reader) (*InstallerMetadata, error) {
	h := sha256.New()
	r = io.TeeReader(r, h)
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()
	r = gz

	// validate tar archive
	tr := tar.NewReader(r)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	// ensure the whole file is read to get the correct hash
	if _, err := io.Copy(io.Discard, r); err != nil {
		return nil, fmt.Errorf("failed to read all content: %w", err)
	}

	// return empty installer metadata; fallback for name/version is handled in the caller
	return &InstallerMetadata{SHASum: h.Sum(nil)}, nil
}
