//go:build darwin

// Package brew_list implements the table for getting Homebrew package information
// on macOS systems.
//
// Based on https://github.com/allenhouchins/fleet-extensions/tree/main/brew_list
package brew_list

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("package_name"),
		table.TextColumn("version"),
		table.TextColumn("install_path"),
		table.TextColumn("type"),
	}
}

func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// Find Homebrew installation path
	brewPath, err := findHomebrewPath()
	if err != nil {
		log.Debug().Err(err).Msg("failed to find Homebrew installation")
		return []map[string]string{}, nil
	}

	// Read from Homebrew database directly
	results, err := readHomebrewDatabase(ctx, brewPath)
	if err != nil {
		log.Debug().Err(err).Msg("failed to read Homebrew database")
		return []map[string]string{}, nil
	}

	return results, nil
}

func findHomebrewPath() (string, error) {
	// First, try to find brew using 'which' command
	whichCmd := exec.Command("which", "brew")
	output, err := whichCmd.Output()
	if err == nil {
		brewPath := strings.TrimSpace(string(output))
		if brewPath != "" {
			// Extract the Homebrew root directory from the brew binary path
			// e.g., /opt/homebrew/bin/brew -> /opt/homebrew
			homebrewRoot := filepath.Dir(filepath.Dir(brewPath))
			if _, err := os.Stat(homebrewRoot); err == nil {
				return homebrewRoot, nil
			}
		}
	}

	// Fallback: check common Homebrew installation paths
	homebrewPaths := []string{
		"/opt/homebrew", // Apple Silicon Mac
		"/usr/local",    // Intel Mac
	}

	for _, path := range homebrewPaths {
		// Check if the path exists and contains Homebrew
		if _, err := os.Stat(path); err == nil {
			// Check for Homebrew-specific files
			if _, err := os.Stat(filepath.Join(path, "bin", "brew")); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("Homebrew installation not found")
}

func readHomebrewDatabase(ctx context.Context, brewPath string) ([]map[string]string, error) {
	// Try multiple possible database locations
	possibleDBPaths := []string{
		filepath.Join(brewPath, "var", "db", "formula_versions.db"),
		filepath.Join(brewPath, "var", "db", "formula_versions.sqlite"),
		filepath.Join(brewPath, "var", "db", "formula_versions"),
		filepath.Join(brewPath, "var", "db", "homebrew.db"),
		filepath.Join(brewPath, "var", "db", "homebrew.sqlite"),
		filepath.Join(brewPath, "var", "db", "homebrew"),
	}

	var dbPath string
	for _, path := range possibleDBPaths {
		if _, err := os.Stat(path); err == nil {
			dbPath = path
			break
		}
	}

	if dbPath == "" {
		return readBrewCommands(ctx, brewPath)
	}

	// Copy database to temporary location to avoid locking issues
	tempDB, err := copyDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy database: %v", err)
	}
	defer os.Remove(tempDB)

	// Open the temporary database
	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Query the database for installed packages
	rows, err := db.QueryContext(ctx, `
		SELECT name, version, path 
		FROM formula_versions 
		WHERE installed = 1
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %v", err)
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var name, version, path sql.NullString
		err := rows.Scan(&name, &version, &path)
		if err != nil {
			log.Debug().Err(err).Msg("failed to scan row")
			continue
		}

		packageName := ""
		if name.Valid {
			packageName = name.String
		}

		packageVersion := ""
		if version.Valid {
			packageVersion = version.String
		}

		installPath := ""
		if path.Valid {
			installPath = path.String
		} else {
			// Fallback: construct path from Homebrew prefix
			installPath = filepath.Join(brewPath, "opt", packageName)
		}

		if packageName != "" {
			// Determine if it's a cask or formula
			packageType := determinePackageType(brewPath, packageName)

			results = append(results, map[string]string{
				"package_name": packageName,
				"version":      packageVersion,
				"install_path": installPath,
				"type":         packageType,
			})
		}
	}

	return results, nil
}

func copyDatabase(srcPath string) (string, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "homebrew_db_*.db")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Copy the database file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		os.Remove(tempPath)
		return "", err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(tempPath)
	if err != nil {
		os.Remove(tempPath)
		return "", err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		os.Remove(tempPath)
		return "", err
	}

	return tempPath, nil
}

func readBrewCommands(ctx context.Context, brewPath string) ([]map[string]string, error) {
	// Use brew commands with proper environment setup to avoid root issues
	brewBinary := filepath.Join(brewPath, "bin", "brew")

	// Get list of installed packages
	cmd := exec.CommandContext(ctx, brewBinary, "list")
	cmd.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ANALYTICS=1")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("brew list command failed: %v", err)
	}

	// Try to get versions using brew list --versions first
	versionMap := make(map[string]string)
	versionCmd := exec.CommandContext(ctx, brewBinary, "list", "--versions")
	versionCmd.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ANALYTICS=1")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		// Fallback: try to get versions from package directories
		versionMap = getVersionsFromDirectories(brewPath, strings.Fields(string(output)))
	} else {
		// Parse versions from command output
		scanner := bufio.NewScanner(strings.NewReader(string(versionOutput)))
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				versionMap[parts[0]] = parts[1]
			}
		}
	}

	// Parse package list and build results
	results := []map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		packageName := strings.TrimSpace(scanner.Text())
		if packageName == "" {
			continue
		}

		// Get version from the map and construct install path
		version := versionMap[packageName]
		installPath := filepath.Join(brewPath, "opt", packageName)

		// Determine if it's a cask or formula
		packageType := determinePackageType(brewPath, packageName)

		results = append(results, map[string]string{
			"package_name": packageName,
			"version":      version,
			"install_path": installPath,
			"type":         packageType,
		})
	}

	return results, nil
}

func getVersionsFromDirectories(brewPath string, packageNames []string) map[string]string {
	versionMap := make(map[string]string)

	for _, packageName := range packageNames {
		// Try to read version from package directory
		packagePath := filepath.Join(brewPath, "opt", packageName)

		// Check if there's a version file
		versionFile := filepath.Join(packagePath, "VERSION")
		if content, err := os.ReadFile(versionFile); err == nil {
			version := strings.TrimSpace(string(content))
			if version != "" {
				versionMap[packageName] = version
				continue
			}
		}

		// Check if there's a .version file
		versionFile = filepath.Join(packagePath, ".version")
		if content, err := os.ReadFile(versionFile); err == nil {
			version := strings.TrimSpace(string(content))
			if version != "" {
				versionMap[packageName] = version
				continue
			}
		}

		// Try to extract version from symlink target
		if linkTarget, err := os.Readlink(packagePath); err == nil {
			// Extract version from path like /opt/homebrew/Cellar/package/1.2.3
			parts := strings.Split(linkTarget, "/")
			for i, part := range parts {
				if part == "Cellar" && i+2 < len(parts) {
					// The version should be the part after the package name
					versionMap[packageName] = parts[i+2]
					break
				}
			}
		}
	}

	return versionMap
}

func determinePackageType(brewPath, packageName string) string {
	// Check if it's a cask by looking in the Caskroom directory
	caskPath := filepath.Join(brewPath, "Caskroom", packageName)
	if _, err := os.Stat(caskPath); err == nil {
		return "cask"
	}

	// Check if it's a formula by looking in the Cellar directory
	cellarPath := filepath.Join(brewPath, "Cellar", packageName)
	if _, err := os.Stat(cellarPath); err == nil {
		return "formula"
	}

	// Default to formula if we can't determine
	return "formula"
}
