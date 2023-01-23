package table

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"hash/crc64"
	"image/png"
	"os"

	"strings"

	"github.com/mat/besticon/ico"
	"github.com/nfnt/resize"
	"github.com/osquery/osquery-go/plugin/table"
	"golang.org/x/sys/windows/registry"
)

var crcTable = crc64.MakeTable(crc64.ECMA)

type icon struct {
	base64 string
	hash   uint64
}

func ProgramIcons() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("name"),
		table.TextColumn("version"),
		table.TextColumn("icon"),
		table.TextColumn("hash"),
	}
	return table.NewPlugin("kolide_program_icons", columns, generateProgramIcons)
}

func generateProgramIcons(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	results = append(results, generateUninstallerProgramIcons()...)
	results = append(results, generateInstallersProgramIcons()...)

	return results, nil
}

func generateUninstallerProgramIcons() []map[string]string {
	var uninstallerIcons []map[string]string

	uninstallRegPaths := map[registry.Key][]string{
		registry.LOCAL_MACHINE: append(expandRegistryKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*`),
			expandRegistryKey(registry.LOCAL_MACHINE, `\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*`)...),
		registry.USERS: expandRegistryKey(registry.USERS, `*\Software\Microsoft\Windows\CurrentVersion\Uninstall\*`),
	}

	for key, paths := range uninstallRegPaths {
		for _, path := range paths {
			key, err := registry.OpenKey(key, path, registry.READ)
			defer key.Close()
			if err != nil {
				continue
			}

			iconPath, _, err := key.GetStringValue("DisplayIcon")
			icon, err := parseIcoFile(iconPath)
			if err != nil {
				continue
			}
			name, _, err := key.GetStringValue("DisplayName")
			if err != nil {
				continue
			}
			version, _, _ := key.GetStringValue("DisplayVersion")

			uninstallerIcons = append(uninstallerIcons, map[string]string{
				"icon":    icon.base64,
				"hash":    fmt.Sprintf("%x", icon.hash),
				"name":    name,
				"version": version,
			})
		}
	}
	return uninstallerIcons
}

func generateInstallersProgramIcons() []map[string]string {
	var installerIcons []map[string]string

	productRegPaths := map[registry.Key][]string{
		registry.CLASSES_ROOT: expandRegistryKey(registry.CLASSES_ROOT, `Installer\Products\*`),
		registry.USERS:        expandRegistryKey(registry.USERS, `*\Software\Microsoft\Installer\Products\*`),
	}

	for key, paths := range productRegPaths {
		for _, path := range paths {
			key, err := registry.OpenKey(key, path, registry.READ)
			defer key.Close()
			if err != nil {
				continue
			}

			iconPath, _, err := key.GetStringValue("ProductIcon")
			icon, err := parseIcoFile(iconPath)
			if err != nil {
				continue
			}
			name, _, _ := key.GetStringValue("ProductName")
			if err != nil {
				continue
			}

			installerIcons = append(installerIcons, map[string]string{
				"icon": icon.base64,
				"hash": fmt.Sprintf("%x", icon.hash),
				"name": name,
			})
		}
	}

	return installerIcons
}

// parseIcoFile returns a base64 encoded version and a hash of the ico.
//
// This doesn't support extracting an icon from a exe. Windows stores some icon in
// the exe like 'OneDriveSetup.exe,-101'
func parseIcoFile(fullPath string) (icon, error) {
	var programIcon icon
	expandedPath, err := registry.ExpandString(fullPath)
	icoReader, err := os.Open(expandedPath)
	if err != nil {
		return programIcon, err
	}
	img, err := ico.Decode(icoReader)
	if err != nil {
		return programIcon, err
	}
	buf := new(bytes.Buffer)
	img = resize.Resize(128, 128, img, resize.Bilinear)
	if err := png.Encode(buf, img); err != nil {
		return programIcon, err
	}

	checksum := crc64.Checksum(buf.Bytes(), crcTable)
	return icon{base64: base64.StdEncoding.EncodeToString(buf.Bytes()), hash: checksum}, nil
}

// expandRegistryKey takes a hive and path, and does a non-recursive glob expansion
//
// For example expandRegistryKey(registry.USERS, `*\Software\Microsoft\Installer\Products\*`)
// expands to
// USER1\Software\Microsoft\Installer\Products\2CCC92FB8B3D5F6499511F546A784ACD
// USER1\Software\Microsoft\Installer\Products\1AAA2FB8B3D5F6499511F546A784ACD
// USER2\Software\Microsoft\Installer\Products\3FFF92FB8B3D5F6499511F546A784ACD
// USER2\Software\Microsoft\Installer\Products\5DDD92FB8B3D5F6499511F546A784ACD
func expandRegistryKey(hive registry.Key, pattern string) []string {
	var paths []string
	magicChar := `*`

	patternsQueue := []string{pattern}
	for len(patternsQueue) > 0 {
		expandablePattern := patternsQueue[0]
		patternsQueue = patternsQueue[1:]

		// add path to results if it doesn't contain the magic char
		if !strings.Contains(expandablePattern, magicChar) {
			paths = append(paths, expandablePattern)
			continue
		}

		patternParts := strings.SplitN(expandablePattern, magicChar, 2)
		key, err := registry.OpenKey(hive, patternParts[0], registry.READ)
		if err != nil {
			continue
		}
		stats, err := key.Stat()
		if err != nil {
			continue
		}
		subKeyNames, err := key.ReadSubKeyNames(int(stats.SubKeyCount))
		if err != nil {
			continue
		}

		for _, subKeyName := range subKeyNames {
			patternsQueue = append(patternsQueue, patternParts[0]+subKeyName+patternParts[1])
		}
	}

	return paths
}
