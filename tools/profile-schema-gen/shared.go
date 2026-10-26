package main

// Helpers used by both the Apple and Windows pipelines.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func writeFile(name string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	return f.Close()
}

// mergeTypes returns a with b's values appended, skipping ones a already has.
func mergeTypes[S ~[]E, E comparable](a, b S) S {
	out := slices.Clone(a)
	for _, t := range b {
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// replaceDownload clears dir, runs fetch to fill it, and writes the marker fetch returns. The
// marker is written last, so an interrupted download never looks complete.
func replaceDownload(ctx context.Context, logger *slog.Logger, dir, marker string, fetch func() (string, error)) (string, error) {
	logger.DebugContext(ctx, "clearing previous download", "dir", dir)
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	value, err := fetch()
	if err != nil {
		return "", err
	}
	return value, os.WriteFile(filepath.Join(dir, marker), []byte(value+"\n"), 0o644)
}

// readMarker returns which upstream version dir holds.
func readMarker(dir, marker string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, marker))
	return strings.TrimSpace(string(b)), err
}

// httpGet returns the body of a 200 response, read up to limit bytes.
func httpGet(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s from %s", resp.Status, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return body, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
