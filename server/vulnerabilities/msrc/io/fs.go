package io

import (
	"os"
	"path/filepath"
	"strings"
)

type FSAPI interface {
	Bulletins() ([]SecurityBulletinName, error)
	Delete(SecurityBulletinName) error
}

type FSClient struct {
	dir string
}

func NewFSClient(dir string) FSClient {
	return FSClient{
		dir: dir,
	}
}

// Delete deletes the provided security bulletin name from 'dir'.
func (fs FSClient) Delete(b SecurityBulletinName) error {
	path := filepath.Join(fs.dir, string(b))
	return os.Remove(path)
}

// Bulletins walks 'dir' returning all security bulletin names.
func (fs FSClient) Bulletins() ([]SecurityBulletinName, error) {
	var result []SecurityBulletinName

	err := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		filePath := filepath.Base(path)
		if strings.HasPrefix(filePath, mSRCFilePrefix) {
			result = append(result, NewSecurityBulletinName(filePath))
		}

		return nil
	})
	return result, err
}
