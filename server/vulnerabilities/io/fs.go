package io

import (
	"os"
	"path/filepath"
	"strings"
)

type FSAPI interface {
	MSRCBulletins() ([]MetadataFileName, error)
	Delete(MetadataFileName) error
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
func (fs FSClient) Delete(b MetadataFileName) error {
	path := filepath.Join(fs.dir, b.filename)
	return os.Remove(path)
}

// MSRC walks 'dir' returning all security bulletin names.
func (fs FSClient) MSRCBulletins() ([]MetadataFileName, error) {
	var result []MetadataFileName

	err := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		filePath := filepath.Base(path)
		if strings.HasPrefix(filePath, MSRCFilePrefix) {
			result = append(result, NewMSRCMetadataFileName(filePath))
		}

		return nil
	})
	return result, err
}
