package io

import (
	"errors"
	"strings"
	"time"
)

const (
	MSRCFilePrefix = "fleet_msrc_"
	fileExt        = "json"
	dateLayout     = "2006_01_02"
)

// Bulletins are published as assets to GH and copies are downloaded to the local FS. The file name
// of those assets contain some useful information like the 'product name' and the date the asset was modified. This type
// provides an abstration around the asset 'file name' to allow us to easy extract/compare the encoded info.
type SecurityBulletinName string

func NewSecurityBulletinName(str string) SecurityBulletinName {
	return SecurityBulletinName(str)
}

func (sbn SecurityBulletinName) date() (time.Time, error) {
	parts := strings.Split(string(sbn), "-")

	if len(parts) != 2 {
		return time.Now(), errors.New("invalid security bulletin name")
	}
	timeRaw := strings.TrimSuffix(parts[1], "."+fileExt)
	return time.Parse(dateLayout, timeRaw)
}

func (sbn SecurityBulletinName) Before(other SecurityBulletinName) bool {
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

func (sbn SecurityBulletinName) ProductName() string {
	pName := strings.TrimPrefix(string(sbn), MSRCFilePrefix)
	parts := strings.Split(pName, "-")

	if len(parts) != 2 {
		return ""
	}

	return strings.Replace(parts[0], "_", " ", -1)
}
