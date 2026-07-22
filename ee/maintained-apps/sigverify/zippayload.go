package sigverify

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxZipEntrySize bounds a single extracted zip entry (decompression-bomb
// guard); comfortably above any real installer payload.
const maxZipEntrySize = 10 << 30 // 10 GiB

// ExtractZipPayload extracts zipPath into destDir and returns the path of the
// first payload file whose extension matches one of exts (searched top-level
// first, then one directory deep — the same lookup VerifyZipPayload uses for
// .app bundles). It returns "" with no error when the archive contains no
// matching payload. Used for zip-wrapped Windows installers, whose
// Authenticode signature lives on the .msi/.exe inside the archive, not on
// the zip container.
func ExtractZipPayload(zipPath, destDir string, exts []string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, filepath.FromSlash(f.Name))
		// Zip-slip guard: entries must stay inside destDir.
		rel, err := filepath.Rel(destDir, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", fmt.Errorf("creating directory %s: %w", f.Name, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("creating parent directory for %s: %w", f.Name, err)
		}

		if err := extractZipEntry(f, target, maxZipEntrySize); err != nil {
			return "", err
		}
	}

	for _, ext := range exts {
		if payload := findPayload(destDir, ext); payload != "" {
			return payload, nil
		}
	}
	return "", nil
}

func extractZipEntry(f *zip.File, target string, limit int64) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("opening zip entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating %s: %w", target, err)
	}

	// Copy through a limit of max+1 so an entry exactly at the limit passes;
	// only reading past it means the entry is oversized.
	n, copyErr := io.Copy(out, io.LimitReader(rc, limit+1))
	// An explicit Close catches flush errors a deferred close would swallow;
	// the extracted payload is about to be signature-verified, so a silently
	// truncated file must not pass as the real payload.
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("extracting %s: %w", f.Name, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing %s: %w", target, closeErr)
	}
	if n > limit {
		return fmt.Errorf("zip entry %s exceeds the %d byte extraction limit", f.Name, limit)
	}
	return nil
}
