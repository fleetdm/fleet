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

	"github.com/ulikunitz/xz"
)

// Decompressed downloads and decompresses a file from a URL to a local path.
//
// It supports gz, bz2 and gz compressed files.
func Decompressed(client *http.Client, u url.URL, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpFile, err := ioutil.TempFile("", fmt.Sprintf("%s*", filepath.Base(path)))
	if err != nil {
		return err
	}
	defer tmpFile.Close()

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
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return err
	}
	return nil
}
