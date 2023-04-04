// package appmanifest provides utilities for managing app manifest files
// used by MDM InstallApplication commands.
//
// It's heavily based on the micromdm/mdm/appmanifest package but it uses
// SHA256 as the hashing algorithm instead of MD5.
package appmanifest

import (
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/micromdm/micromdm/mdm/appmanifest"
)

// Create an AppManifest and write it to an io.Writer.
func Create(file io.Reader, url string) (*appmanifest.Manifest, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	sum := fmt.Sprintf("%x", hash.Sum(nil))

	ast := appmanifest.Asset{
		Kind:       "software-package",
		SHA256Size: int64(hash.Size()),
		SHA256s:    []string{sum},
		URL:        url,
	}

	return &appmanifest.Manifest{
		ManifestItems: []appmanifest.Item{
			{
				Assets: []appmanifest.Asset{ast},
			},
		},
	}, nil
}
