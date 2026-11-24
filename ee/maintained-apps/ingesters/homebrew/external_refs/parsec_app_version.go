package externalrefs

import (
	"cmp"
	"context"
	"encoding/xml"
	"fmt"
)

const parsecAppBundleID = "tv.parsec.www"

type pkgInfoBundles struct {
	Bundles []struct {
		ID                 string `xml:"id,attr"`
		ShortVersionString string `xml:"CFBundleShortVersionString,attr"`
		BundleVersion      string `xml:"CFBundleVersion,attr"`
	} `xml:"bundle"`
}

// ParsecPackageAppVersion returns the version of Parsec.app inside the
// installer at installerURL, as the apps table reports it. The cask version
// tracks the parsecd dylib that Parsec.app downloads and updates by itself at
// runtime, which no osquery table reports; the app bundle only changes when
// Parsec republishes the pkg.
func ParsecPackageAppVersion(ctx context.Context, installerURL string) (string, error) {
	return parsecPackageAppVersion(func(start, end int64) ([]byte, error) {
		return rangeGet(ctx, installerURL, start, end)
	})
}

func parsecPackageAppVersion(fetch func(start, end int64) ([]byte, error)) (string, error) {
	pkgInfoXML, err := readXarTopLevelFile(fetch, "PackageInfo")
	if err != nil {
		return "", err
	}

	var info pkgInfoBundles
	if err := xml.Unmarshal(pkgInfoXML, &info); err != nil {
		return "", fmt.Errorf("parsing PackageInfo file: %w", err)
	}
	for _, b := range info.Bundles {
		if b.ID == parsecAppBundleID {
			if v := cmp.Or(b.ShortVersionString, b.BundleVersion); v != "" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("no versioned bundle %q in PackageInfo file", parsecAppBundleID)
}
