package msrc_io

import (
	"os"
	"path/filepath"
	"strings"
)

// localBulletins walks 'dstDir' returning all security bulletin names.
func localBulletins(dstDir string) func() ([]SecurityBulletinName, error) {
	return func() ([]SecurityBulletinName, error) {
		var result []SecurityBulletinName

		err := filepath.WalkDir(dstDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			filePath := filepath.Base(path)
			if strings.HasPrefix(filePath, MSRCFilePrefix) {
				result = append(result, NewSecurityBulletinName(filePath))
			}

			return nil
		})
		return result, err
	}
}
