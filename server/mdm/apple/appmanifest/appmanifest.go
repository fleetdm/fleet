// package appmanifest provides utilities for managing app manifest files
// used by MDM InstallApplication commands.
//
// It's heavily based on the micromdm/mdm/appmanifest package but it uses
// SHA256 as the hashing algorithm instead of MD5.
package appmanifest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/micromdm/plist"
)

type Manifest appmanifest.Manifest

func (m *Manifest) Plist() ([]byte, error) {
	var buf bytes.Buffer
	enc := plist.NewEncoder(&buf)
	enc.Indent("  ")
	if err := enc.Encode(m); err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	return buf.Bytes(), nil
}

// Create builds an AppManifest using SHA256 checksums and the provided URL
func New(file io.Reader, url string) (*Manifest, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return NewFromSha(hash.Sum(nil), url), nil
}

func NewFromSha(sha []byte, url string) *Manifest {
	ast := appmanifest.Asset{
		Kind:       "software-package",
		SHA256Size: sha256.Size,
		SHA256s:    []string{fmt.Sprintf("%x", sha)},
		URL:        url,
	}

	return &Manifest{
		ManifestItems: []appmanifest.Item{
			{
				Assets: []appmanifest.Asset{ast},
			},
		},
	}
}

func NewPlist(file io.Reader, url string) ([]byte, error) {
	m, err := New(file, url)
	if err != nil {
		return nil, err
	}

	return m.Plist()
}
