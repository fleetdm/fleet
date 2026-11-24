package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const tarGzFormat = "tar.gz"

var gzipMagic = []byte{0x1f, 0x8b}

// ValidateTarball confirms that a .tar.gz file is valid, then returns empty installer metadata for fallback
func ValidateTarball(r io.Reader) (*InstallerMetadata, error) {
	if p, ok := r.(fleet.PrevalidatedInstaller); ok && p.PrevalidatedFormat() == tarGzFormat {
		// Already fully validated while it was streamed to the store.
		sum, err := installerSHA256(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read all content: %w", err)
		}
		return &InstallerMetadata{SHASum: sum}, nil
	}

	src, h := newSequentialHasher(r)
	if err := validateTarGz(src); err != nil {
		return nil, err
	}
	sum, err := h.finish()
	if err != nil {
		return nil, fmt.Errorf("failed to read all content: %w", err)
	}

	// return empty installer metadata; fallback for name/version is handled in the caller
	return &InstallerMetadata{SHASum: sum}, nil
}

// validateTarGz reads the whole gzip stream from r and checks that it is a
// valid tar archive.
func validateTarGz(r io.Reader) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	// ensure the whole archive is read so it is fully validated
	if _, err := io.Copy(io.Discard, gz); err != nil { //nolint:gosec // G110: output is discarded, not buffered
		return fmt.Errorf("failed to read all content: %w", err)
	}
	return nil
}

// TarGzStreamValidator is a fleet.InstallerStreamValidator for .tar.gz
// installers. Content starting with the gzip magic bytes is validated the way
// ValidateTarball does it, in a goroutine fed as the content is written; other
// content is ignored. It is not safe for concurrent use.
type TarGzStreamValidator struct {
	head []byte // content written before its type is known
	pw   *io.PipeWriter
	done chan error
	// skip is set once the content isn't a .tar.gz or validation stopped.
	skip bool
}

var _ fleet.InstallerStreamValidator = (*TarGzStreamValidator)(nil)

// NewTarGzStreamValidator returns a validator for a single installer.
func NewTarGzStreamValidator() *TarGzStreamValidator {
	return &TarGzStreamValidator{}
}

// Write feeds content to the validator. It never fails.
func (v *TarGzStreamValidator) Write(p []byte) (int, error) {
	if v.skip || len(p) == 0 {
		return len(p), nil
	}
	b := p
	if v.pw == nil {
		v.head = append(v.head, p...)
		if len(v.head) < len(gzipMagic) {
			return len(p), nil
		}
		if !bytes.HasPrefix(v.head, gzipMagic) {
			v.skip, v.head = true, nil
			return len(p), nil
		}
		pr, pw := io.Pipe()
		v.pw, v.done = pw, make(chan error, 1)
		go func() {
			err := validateTarGz(pr)
			// Unblocks (and fails) the writes if validation stopped early.
			_ = pr.CloseWithError(err)
			v.done <- err
		}()
		b, v.head = v.head, nil
	}
	if _, err := v.pw.Write(b); err != nil {
		v.skip = true
	}
	return len(p), nil
}

// Finish ends the content and returns "tar.gz" if it is a valid .tar.gz
// installer, "" otherwise.
func (v *TarGzStreamValidator) Finish() string {
	if v.pw == nil {
		v.skip = true
		return ""
	}
	_ = v.pw.Close()
	err := <-v.done
	v.pw, v.skip = nil, true
	if err != nil {
		return ""
	}
	return tarGzFormat
}
