// Package download has utilities to download resources from URLs.
package download

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// Download downloads a file from a URL and writes it to path.
func Download(client *http.Client, u *url.URL, path string) error {
	return download(client, u, path, false)
}

// DownloadAndExtract downloads and extracts a file from a URL and writes it to path.
// The compression method is determined using extension from the url path. Only .gz, .bz2, or .xz extensions are supported.
func DownloadAndExtract(client *http.Client, u *url.URL, path string) error {
	return download(client, u, path, true)
}

func download(client *http.Client, u *url.URL, path string, extract bool) error {
	// atomically write to file
	dir, file := filepath.Split(path)
	if dir == "" {
		// If the file is in the current working directory, then dir will be "".
		// However, this means that os.CreateTemp will use the default directory
		// for temporary files, which is wrong.
		dir = "."
	}

	// ensure dir exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, file)
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer tmpFile.Close() // ignore err from closing twice

	// Clean up tmp file if not moved
	moved := false
	defer func() {
		if !moved {
			os.Remove(tmpFile.Name())
		}
	}()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := io.Reader(resp.Body)

	// extract (optional)
	if extract {
		switch {
		case strings.HasSuffix(u.Path, "gz"):
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			r = gr
		case strings.HasSuffix(u.Path, "bz2"):
			r = bzip2.NewReader(resp.Body)
		case strings.HasSuffix(u.Path, "xz"):
			xzr, err := xz.NewReader(resp.Body)
			if err != nil {
				return err
			}
			r = xzr
		default:
			return fmt.Errorf("unknown extension: %s", u.Path)
		}
	}

	if _, err := io.Copy(tmpFile, r); err != nil {
		return fmt.Errorf("write to temporary file: %w", err)
	}

	// Writes are not synchronous. Handle errors from writes returned by Close.
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("write and close temporary file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return err
	}

	moved = true

	return nil
}
