package go_packages

import (
	"debug/buildinfo"
	"os"
	"path/filepath"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// GoPackagesColumns is the schema of the go_packages table.
func GoPackagesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("name"),
		table.TextColumn("version"),
		table.TextColumn("module_path"),
		table.TextColumn("import_path"),
		table.TextColumn("go_version"),
		table.TextColumn("installed_path"),
	}
}

// generateForDirs scans each directory's go/bin subdirectory for Go binaries
// and returns rows with embedded build metadata.
func generateForDirs(homeDirs []string) []map[string]string {
	var results []map[string]string
	for _, homeDir := range homeDirs {
		goBinDir := filepath.Join(homeDir, "go", "bin")
		entries, err := os.ReadDir(goBinDir)
		if err != nil {
			continue // directory doesn't exist or can't be read
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(goBinDir, entry.Name())
			row := readGoBinary(fullPath)
			if row != nil {
				results = append(results, row)
			}
		}
	}
	return results
}

// readGoBinary extracts embedded build information from a Go binary.
// Returns nil if the file is not a Go binary or cannot be read.
func readGoBinary(path string) map[string]string {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("go_packages: failed to read build info")
		return nil
	}
	return map[string]string{
		"name":           filepath.Base(path),
		"version":        info.Main.Version,
		"module_path":    info.Main.Path,
		"import_path":    info.Path,
		"go_version":     info.GoVersion,
		"installed_path": path,
	}
}
