package io

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	mSRCFilePrefix              = "fleet_msrc_"
	macOfficeReleaseNotesPrefix = "fleet_macoffice_release_notes_"
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

func NewMSRCMetadata(filename string) (MetadataFileName, error) {
	mfn := MetadataFileName{prefix: mSRCFilePrefix, filename: filename}

	// Check that the filename contains a valid timestamp
	_, err := mfn.date()

	return mfn, err
}

func NewMacOfficeRelNotesMetadata(filename string) (MetadataFileName, error) {
	mfn := MetadataFileName{prefix: macOfficeReleaseNotesPrefix, filename: filename}

	// Check that the filename contains a valid timestamp
	_, err := mfn.date()

	return mfn, err
}

func (mfn MetadataFileName) date() (time.Time, error) {
	parts := strings.Split(mfn.filename, "-")

	if len(parts) != 2 {
		return time.Now(), errors.New("invalid file name")
	}
	timeRaw := strings.TrimSuffix(parts[1], "."+fileExt)
	return time.Parse(dateLayout, timeRaw)
}

func (mfn MetadataFileName) Before(other MetadataFileName) bool {
	// If mfn is empty ...
	if mfn.filename == "" {
		return true
	}

	// If other is empty ...
	if other.filename == "" {
		return false
	}

	// We check that the MetadataFileName contains a valid timestamp at construction time so both of
	// these calls shouldn't fail, only reason for having an error at this point is if we are
	// dealing with an 'empty' (MetadataFileName{}) struct.
	a, _ := mfn.date()
	b, _ := other.date()

	return a.Before(b)
}

func (mfn MetadataFileName) ProductName() string {
	pName := strings.TrimPrefix(mfn.filename, mfn.prefix)
	parts := strings.Split(pName, "-")

	if len(parts) != 2 {
		return ""
	}

	return strings.ReplaceAll(parts[0], "_", " ")
}

func (mfn MetadataFileName) String() string {
	return mfn.filename
}

func MSRCFileName(productName string, date time.Time) string {
	pName := strings.ReplaceAll(productName, " ", "_")
	return fmt.Sprintf("%s%s-%d_%02d_%02d.%s", mSRCFilePrefix, pName, date.Year(), date.Month(), date.Day(), fileExt)
}

func MacOfficeRelNotesFileName(date time.Time) string {
	return fmt.Sprintf("%s%s-%d_%02d_%02d.%s", macOfficeReleaseNotesPrefix, "macoffice", date.Year(), date.Month(), date.Day(), fileExt)
}
