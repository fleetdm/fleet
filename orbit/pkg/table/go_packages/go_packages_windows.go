//go:build windows

package go_packages

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// GenerateGoPackages is called to return the results for the go_packages table at query time.
func GenerateGoPackages(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	homeDirs, err := windowsHomeDirs()
	if err != nil {
		log.Debug().Err(err).Msg("go_packages: failed to list home directories")
		return nil, nil
	}
	return generateForDirs(homeDirs), nil
}

// windowsHomeDirs returns home directories for real users on Windows.
func windowsHomeDirs() ([]string, error) {
	systemDrive := os.Getenv("SystemDrive")
	if systemDrive == "" {
		systemDrive = "C:"
	}
	usersDir := systemDrive + `\Users`
	entries, err := os.ReadDir(usersDir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if name == "public" || name == "default" || name == "default user" || name == "all users" {
			continue
		}
		dirs = append(dirs, filepath.Join(usersDir, entry.Name()))
	}
	return dirs, nil
}
