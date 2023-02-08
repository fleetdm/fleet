package io

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	MSRCFilePrefix              = "fleet_msrc_"
	MacOfficeReleaseNotesPrefix = "fleet_macoffice_release_notes_"
	fileExt                     = "json"
	dateLayout                  = "2006_01_02"
)

// MSRC Bulletins and other metadata files are published as assets to GH and copies are downloaded to the local FS. The file name
// of those assets contain some useful information like the 'product name' and the date the asset was modified. This type
// provides an abstration around the asset 'file name' to allow us to easy extract/compare the encoded info.
type MetadataFileName struct {
	prefix   string
	filename string
}

func NewMSRCMetadata(filename string) MetadataFileName {
	return MetadataFileName{prefix: MSRCFilePrefix, filename: filename}
}

func NewMacOfficeReleasesMetadata(filename string) MetadataFileName {
	return MetadataFileName{prefix: MacOfficeReleaseNotesPrefix, filename: filename}
}

func (sbn MetadataFileName) date() (time.Time, error) {
	parts := strings.Split(sbn.filename, "-")

	if len(parts) != 2 {
		return time.Now(), errors.New("invalid file name")
	}
	timeRaw := strings.TrimSuffix(parts[1], "."+fileExt)
	return time.Parse(dateLayout, timeRaw)
}

func ToFileName(prefix string, productName string, date time.Time) string {
	pName := strings.Replace(productName, " ", "_", -1)
	return fmt.Sprintf("%s%s-%d_%02d_%02d.%s", prefix, pName, date.Year(), date.Month(), date.Day(), fileExt)
}

func (sbn MetadataFileName) Before(other MetadataFileName) bool {
	a, err := sbn.date()
	if err != nil {
		return false
	}

	b, err := other.date()
	if err != nil {
		return false
	}

	return a.Before(b)
}

func (sbn MetadataFileName) ProductName() string {
	pName := strings.TrimPrefix(sbn.filename, sbn.prefix)
	parts := strings.Split(pName, "-")

	if len(parts) != 2 {
		return ""
	}

	return strings.Replace(parts[0], "_", " ", -1)
}

func (sbn MetadataFileName) String() string {
	return sbn.filename
}
