package msrc_io

import (
	"os"
	"path/filepath"
	"strings"
)

type MSRCFSAPI interface {
	Bulletins() ([]SecurityBulletinName, error)
	Delete(SecurityBulletinName) error
}

type MSRCFSClient struct {
	dir string
}

func NewMSRCFSClient(dir string) MSRCFSClient {
	return MSRCFSClient{
		dir: dir,
	}
}

// Delete the provided security bulletin from the dir specified at instantiation time.
func (fs MSRCFSClient) Delete(b SecurityBulletinName) error {
	path := filepath.Join(fs.dir, string(b))
	return os.Remove(path)
}

// Bulletins walks 'dstDir' returning all security bulletin names.
func (fs MSRCFSClient) Bulletins() ([]SecurityBulletinName, error) {
	var result []SecurityBulletinName

	err := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
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
