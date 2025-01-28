// Package download has utilities to download resources from URLs.
package download

import (
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ulikunitz/xz"
)

const backoffMaxElapsedTime = 5 * time.Minute

// Download downloads a file from a URL and writes it to path.
//
// It will retry requests until it succeeds. If the server returns a 404
// then it will not retry and return a NotFound error.
func Download(client *http.Client, u *url.URL, path string) error {
	return download(client, u, path, false)
}

// DownloadAndExtract downloads and extracts a file from a URL and writes it to path.
// The compression method is determined using extension from the url path. Only .gz, .bz2, or .xz extensions are supported.
//
// It will retry requests until it succeeds. If the server returns a 404
// then it will not retry and return a NotFound error.
func DownloadAndExtract(client *http.Client, u *url.URL, path string) error {
	return download(client, u, path, true)
}

var NotFound = errors.New("resource not found")

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
		return fmt.Errorf("create directory: %w", err)
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

	operation := func() error {
		if err := tmpFile.Truncate(0); err != nil {
			return fmt.Errorf("truncate temporary file: %w", err)
		}

		if _, err := tmpFile.Seek(0, 0); err != nil {
			return fmt.Errorf("seek temporary file: %w", err)
		}

		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			// OK
		case resp.StatusCode == http.StatusNotFound:
			return &backoff.PermanentError{Err: NotFound}
		default:
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}

		r := io.Reader(resp.Body)

		// extract (optional)
		if extract {
			switch {
			case strings.HasSuffix(u.Path, "gz"):
				gr, err := gzip.NewReader(resp.Body)
				if err != nil {
					return fmt.Errorf("gzip reader: %w", err)
				}
				r = gr
			case strings.HasSuffix(u.Path, "bz2"):
				r = bzip2.NewReader(resp.Body)
			case strings.HasSuffix(u.Path, "xz"):
				xzr, err := xz.NewReader(resp.Body)
				if err != nil {
					return fmt.Errorf("xz reader: %w", err)
				}
				r = xzr
			default:
				return fmt.Errorf("unknown extension: %s", u.Path)
			}
		}

		if _, err := io.Copy(tmpFile, r); err != nil {
			return fmt.Errorf("copy to temporary file: %w", err)
		}

		return nil
	}

	expBackOff := backoff.NewExponentialBackOff()
	expBackOff.MaxElapsedTime = backoffMaxElapsedTime
	if err := backoff.RetryNotify(operation, expBackOff, func(err error, d time.Duration) {
		fmt.Printf("Download failed on %s: %v. Retrying in %v\n", u.String(), err, d)
	}); err != nil {
		return fmt.Errorf("download and write file: %w", err)
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
