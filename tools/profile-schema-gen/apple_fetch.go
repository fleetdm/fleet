package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

const (
	appleRepo   = "apple/device-management"
	appleBranch = "release"
)

// appleDirs are the only parts of Apple's repo the generator reads.
var appleDirs = []string{"mdm/profiles/", "declarative/declarations/"}

// downloadAppleTarball fetches Apple's repo at sha and extracts the YAML files under appleDirs into
// dest, with the archive's top-level directory stripped.
func downloadAppleTarball(ctx context.Context, logger *slog.Logger, client *http.Client, codeloadURL, sha, dest string) error {
	url := fmt.Sprintf("%s/%s/tar.gz/%s", codeloadURL, appleRepo, sha)
	logger.InfoContext(ctx, "downloading Apple tarball", "url", url, "dest", dest)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s@%s: %w", appleRepo, sha, err)
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return fmt.Errorf("download %s@%s: %w", appleRepo, sha, err)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("download %s@%s: %w", appleRepo, sha, err)
	}
	defer gz.Close()

	var extracted, skipped int
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read %s@%s tarball: %w", appleRepo, sha, err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Entries are "<repo>-<sha>/<path>".
		_, rel, ok := strings.Cut(hdr.Name, "/")
		if !ok || !wantedAppleFile(rel) {
			skipped++
			continue
		}
		// Guard against "../" entries escaping dest.
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("tarball entry %q escapes destination", hdr.Name)
		}
		if err := writeFile(filepath.Join(dest, filepath.FromSlash(rel)), tr); err != nil {
			return err
		}
		logger.DebugContext(ctx, "extracted", "file", rel)
		extracted++
	}
	logger.InfoContext(ctx, "extracted Apple tarball", "commit", sha, "extracted", extracted, "skipped", skipped)
	if extracted == 0 {
		return fmt.Errorf("tarball %s@%s has no YAML files under %v", appleRepo, sha, appleDirs)
	}
	return nil
}

func wantedAppleFile(rel string) bool {
	if path.Ext(rel) != ".yaml" {
		return false
	}
	for _, dir := range appleDirs {
		if strings.HasPrefix(rel, dir) {
			return true
		}
	}
	return false
}
