package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/groob/plist"
)

type InfoPlist struct {
	BundleName         string `plist:"CFBundleName"`
	BundleDisplayName  string `plist:"CFBundleDisplayName"`
	BundleVersion      string `plist:"CFBundleVersion"`
	BundleShortVersion string `plist:"CFBundleShortVersionString"`
}

func GetAppInfo(path string) (string, string, error) {
	plistPath := filepath.Join(path, "Contents", "Info.plist")

	rawPlist, err := os.ReadFile(plistPath)
	if err != nil {
		return "", "", fmt.Errorf("reading Info.plist: %w", err)
	}

	var info InfoPlist
	err = plist.Unmarshal(rawPlist, &info)
	if err != nil {
		return "", "", fmt.Errorf("unmarshal Info.plist: %w", err)
	}

	name := info.BundleDisplayName
	if name == "" {
		name = info.BundleName
	}

	version := info.BundleShortVersion
	if version == "" {
		version = info.BundleVersion
	}

	return name, version, nil
}
