//go:build darwin

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
	homeDirs, err := darwinHomeDirs()
	if err != nil {
		log.Debug().Err(err).Msg("go_packages: failed to list home directories")
		return nil, nil
	}
	return generateForDirs(homeDirs), nil
}

// darwinHomeDirs returns home directories for real users on macOS.
func darwinHomeDirs() ([]string, error) {
	entries, err := os.ReadDir("/Users")
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "Shared" {
			continue
		}
		dirs = append(dirs, filepath.Join("/Users", name))
	}
	return dirs, nil
}
