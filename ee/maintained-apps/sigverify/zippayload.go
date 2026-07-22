package sigverify

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// maxZipEntrySize bounds a single extracted zip entry (decompression-bomb
// guard); comfortably above any real installer payload.
const maxZipEntrySize = 10 << 30 // 10 GiB

// ExtractZipPayload extracts the single payload file from zipPath whose
// extension matches one of exts — preferring earlier extensions in exts, then
// top-level entries over ones a directory deep (entries nested deeper are not
// considered) — into destDir, and returns its path. Only that one entry is
// extracted, so an archive stuffed with other large entries can't exhaust
// disk on a runner. It returns "" with no error when the archive contains no
// matching payload. Used for zip-wrapped Windows installers, whose
// Authenticode signature lives on the .msi/.exe inside the archive, not on
// the zip container.
func ExtractZipPayload(zipPath, destDir string, exts []string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	payload := selectPayloadEntry(r.File, destDir, exts)
	if payload == nil {
		return "", nil
	}

	target := filepath.Join(destDir, filepath.FromSlash(payload.Name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("creating parent directory for %s: %w", payload.Name, err)
	}
	if err := extractZipEntry(payload, target, maxZipEntrySize); err != nil {
		return "", err
	}
	return target, nil
}

// selectPayloadEntry ranks candidate payload entries by extension preference
// (earlier in exts wins), then depth (top-level before one directory deep),
// then name for determinism, and returns the best one. Directory entries,
// entries nested more than one directory deep, and entries that would escape
// destDir (zip-slip) are never candidates.
func selectPayloadEntry(files []*zip.File, destDir string, exts []string) *zip.File {
	var best *zip.File
	bestExt, bestDepth := 0, 0
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		// Zip-slip guard: entries must stay inside destDir.
		target := filepath.Join(destDir, filepath.FromSlash(f.Name))
		rel, err := filepath.Rel(destDir, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		name := path.Clean(f.Name) // zip entry names always use forward slashes
		depth := strings.Count(name, "/")
		if depth > 1 {
			continue
		}
		extIdx := slices.IndexFunc(exts, func(ext string) bool {
			return strings.EqualFold(path.Ext(name), ext)
		})
		if extIdx < 0 {
			continue
		}

		better := best == nil ||
			extIdx < bestExt ||
			(extIdx == bestExt && depth < bestDepth) ||
			(extIdx == bestExt && depth == bestDepth && f.Name < best.Name)
		if better {
			best, bestExt, bestDepth = f, extIdx, depth
		}
	}
	return best
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
