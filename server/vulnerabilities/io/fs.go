package io

import (
	"os"
	"path/filepath"
	"strings"
)

type FSAPI interface {
	MSRCBulletins() ([]MetadataFileName, error)
	MacOfficeReleaseNotes() ([]MetadataFileName, error)
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

// Delete deletes the provided metadata file from 'dir'.
func (fs FSClient) Delete(b MetadataFileName) error {
	path := filepath.Join(fs.dir, b.filename)
	return os.Remove(path)
}

// MSRCBulletins walks 'dir' returning all security bulletin files.
func (fs FSClient) MSRCBulletins() ([]MetadataFileName, error) {
	return fs.list(mSRCFilePrefix, NewMSRCMetadata)
}

// MacOfficeReleaseNotes walks 'dir' returning all mac office release notes
func (fs FSClient) MacOfficeReleaseNotes() ([]MetadataFileName, error) {
	return fs.list(macOfficeReleaseNotesPrefix, NewMacOfficeRelNotesMetadata)
}

func (fs FSClient) list(
	prefix string,
	ctor func(filePath string) MetadataFileName,
) ([]MetadataFileName, error) {
	var result []MetadataFileName
	err := filepath.WalkDir(fs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		filePath := filepath.Base(path)
		if strings.HasPrefix(filePath, prefix) {
			result = append(result, ctor(filePath))
		}
		return nil
	})
	return result, err
}
