// Package download has utilities to download resources from URLs.
package download

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"
)

// Decompressed downloads and decompresses a file from a URL to a local path.
//
// It supports gz, bz2 and xz compressed files.
func Decompressed(client *http.Client, u url.URL, path string) error {

	// atomically write to file
	dir, file := filepath.Split(path)
	if dir == "" {
		// If the file is in the current working directory, then dir will be "".
		// However, this means that ioutil.TempFile will use the default directory
		// for temporary files, which is wrong.
		dir = "."
	}

	// ensure dir exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(dir, file)
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

	var decompressor io.Reader
	switch {
	case strings.HasSuffix(u.Path, "gz"):
		decompressor, err = gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
	case strings.HasSuffix(u.Path, "bz2"):
		decompressor = bzip2.NewReader(resp.Body)
	case strings.HasSuffix(u.Path, "xz"):
		decompressor, err = xz.NewReader(resp.Body)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown extension: %s", u.Path)
	}

	if _, err := io.Copy(tmpFile, decompressor); err != nil {
		return err
	}

	// Writes are not synchronous. Handle errors from writes returned by Close.
	if err := tmpFile.Close(); err != nil {
		return errors.Wrapf(err, "write and close temporary file")
	}

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return err
	}

	moved = true

	return nil
}
