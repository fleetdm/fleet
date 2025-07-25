//go:build darwin || windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Config struct {
	tmpDir                      string
	env                         []string
	installationSearchDirectory string
	operatingSystem             string
	logger                      kitlog.Logger
	logLevel                    string
}

func run(cfg *Config) error {
	apps, err := getListOfApps()
	if err != nil {
		level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error getting list of apps: %v", err))
		return err
	}

	cfg.tmpDir, err = os.MkdirTemp("", "fma-validate-")
	if err != nil {
		level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error creating temporary directory: %v", err))
		return err
	}
	defer func() {
		err := os.RemoveAll(cfg.tmpDir)
		if err != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("warning failed to remove temporary directory: %v", err))
		}
	}()

	ctx := context.Background()

	totalApps := 0
	successfulApps := 0
	appWithError := []string{}
	appWithWarning := []string{}
	frozenApps := []string{}
	for _, app := range apps {
		if app.Platform != cfg.operatingSystem {
			continue
		}

		totalApps++

		level.Info(cfg.logger).Log("msg", fmt.Sprintf("Validating app: %s (%s)", app.Name, app.Slug))
		ac := &AppCommander{cfg: cfg}

		appJson, err := getAppJson(app.Slug)
		if err != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error getting app json manifest: %v", err))
			appWithError = append(appWithError, app.Name)
			continue
		}

		maintainedApp := appFromJson(appJson)
		ac.Name = app.Name
		ac.Slug = app.Slug
		ac.UniqueIdentifier = app.UniqueIdentifier
		// default version to maintained app version
		ac.Version = maintainedApp.Version
		ac.InstallScript = maintainedApp.InstallScript
		ac.UninstallScript = maintainedApp.UninstallScript

		isFrozen, err := ac.isFrozen()
		if err != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error checking if app is frozen: %v", err))
			appWithError = append(appWithError, ac.Name)
			continue
		}
		if isFrozen {
			level.Info(cfg.logger).Log("msg", "App is frozen, skipping validation...")
			frozenApps = append(frozenApps, ac.Name)
			continue
		}

		installerTFR, err := DownloadMaintainedApp(cfg, maintainedApp)
		if err != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error downloading maintained app: %v", err))
			appWithError = append(appWithError, ac.Name)
			continue
		}

		err = ac.extractAppVersion(installerTFR)
		installerTFR.Close()
		if err != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error extracting installer version: %v. Using '%s'", err, ac.Version))
			appWithError = append(appWithError, ac.Name)
		}

		// If application is already installed, attempt to uninstall it
		if slices.Contains(preInstalled, ac.Slug) {
			ac.uninstallPreInstalled(ctx)
		}

		appPath, changerError, listError := ac.expectToChangeFileSystem(
			func() error {
				level.Info(cfg.logger).Log("msg", "Executing install script...")
				output, err := executeScript(cfg, ac.InstallScript)
				if err != nil {
					level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error executing install script: %v", err))
					level.Error(cfg.logger).Log("msg", fmt.Sprintf("Output: %s", output))
					return err
				}
				level.Debug(cfg.logger).Log("msg", fmt.Sprintf("Output: %s", output))
				return nil
			},
		)
		if listError != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error listing directory contents: %v", listError))
		}
		if changerError != nil {
			appWithError = append(appWithError, ac.Name)
			continue
		}
		ac.AppPath = appPath
		if ac.AppPath == "" {
			appWithWarning = append(appWithWarning, ac.Name)
		}

		err = postApplicationInstall(cfg, ac.AppPath)
		if err != nil {
			level.Warn(cfg.logger).Log("msg", fmt.Sprintf("Error detected in post-installation steps: %v", err))
			appWithWarning = append(appWithWarning, ac.Name)
		}

		existance, err := appExists(ctx, cfg.logger, ac.Name, ac.UniqueIdentifier, ac.Version, ac.AppPath)
		if err != nil {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("Error checking if app exists: %v", err))
			appWithError = append(appWithError, ac.Name)
			continue
		}
		if !existance {
			level.Error(cfg.logger).Log("msg", fmt.Sprintf("App version '%s' was not found by osquery", ac.Version))
			appWithError = append(appWithError, ac.Name)
			continue
		}

		// Uninstall
		uninstalled := ac.uninstallApp(ctx)
		if !uninstalled {
			appWithError = append(appWithError, ac.Name)
			continue
		}

		level.Info(cfg.logger).Log("msg", fmt.Sprintf("All checks passed for app: %s (%s)", ac.Name, ac.Slug))
		successfulApps++
	}

	if len(frozenApps) > 0 {
		level.Info(cfg.logger).Log("msg", fmt.Sprintf("Some apps were skipped: %v", frozenApps))
	}
	if len(appWithWarning) > 0 {
		level.Warn(cfg.logger).Log("msg", fmt.Sprintf("Some apps were validated with warnings: %v", appWithWarning))
	}

	if successfulApps == totalApps-len(frozenApps) {
		// All apps were successfully validated!
		level.Info(cfg.logger).Log("msg", fmt.Sprintf("All %d apps were successfully validated.", totalApps))
		return nil
	}

	level.Info(cfg.logger).Log("msg", fmt.Sprintf("Validated %d out of %d apps successfully.", successfulApps, totalApps))
	level.Info(cfg.logger).Log("msg", fmt.Sprintf("Apps with errors: %v", appWithError))
	return errors.New("Some maintained apps failed validation")
}

func main() {
	cfg := &Config{}

	// logger
	cfg.logger = kitlog.NewLogfmtLogger(os.Stderr)
	cfg.logLevel = os.Getenv("LOG_LEVEL")
	if cfg.logLevel == "" {
		cfg.logLevel = "info"
	}

	var lvl level.Option
	switch strings.ToLower(cfg.logLevel) {
	case "debug":
		lvl = level.AllowDebug()
	case "error":
		lvl = level.AllowError()
	default:
		lvl = level.AllowInfo()
	}

	cfg.logger = level.NewFilter(cfg.logger, lvl)
	cfg.logger = kitlog.With(cfg.logger,
		"ts", kitlog.DefaultTimestampUTC,
		"caller", kitlog.DefaultCaller,
	)

	// os detection
	cfg.operatingSystem = strings.ToLower(os.Getenv("GOOS"))
	if cfg.operatingSystem == "" {
		cfg.operatingSystem = runtime.GOOS
		level.Info(cfg.logger).Log("msg", fmt.Sprintf("GOOS environment variable is not set. Using system detected: '%s'", cfg.operatingSystem))
	}
	if cfg.operatingSystem != "darwin" && cfg.operatingSystem != "windows" {
		level.Error(cfg.logger).Log("msg", fmt.Sprintf("Unsupported operating system: %s", cfg.operatingSystem))
		os.Exit(1)
	}

	// installation directory detection
	cfg.installationSearchDirectory = os.Getenv("INSTALLATION_SEARCH_DIRECTORY")
	if cfg.installationSearchDirectory == "" {
		switch cfg.operatingSystem {
		case "darwin":
			cfg.installationSearchDirectory = "/Applications"
		case "windows":
			cfg.installationSearchDirectory = "C:\\Program Files"
		}
		level.Info(cfg.logger).Log("msg", fmt.Sprintf("INSTALLATION_SEARCH_DIRECTORY environment variable is not set. Using default: '%s'", cfg.installationSearchDirectory))
	}

	err := run(cfg)
	if err != nil {
		os.Exit(1)
	}
}

func getListOfApps() ([]maintained_apps.FMAListFileApp, error) {
	appListFilePath := path.Join(maintained_apps.OutputPath, "apps.json")
	inputJson, err := os.ReadFile(appListFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading output apps list file: %w", err)
	}
	var outputAppsFile maintained_apps.FMAListFile
	if err := json.Unmarshal(inputJson, &outputAppsFile); err != nil {
		return nil, fmt.Errorf("unmarshaling output apps list file: %w", err)
	}
	return outputAppsFile.Apps, nil
}

func getAppJson(slug string) (*maintained_apps.FMAManifestFile, error) {
	appJsonFilePath := path.Join(maintained_apps.OutputPath, fmt.Sprintf("%s.json", slug))
	inputJson, err := os.ReadFile(appJsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading app '%s' json manifest: %w", slug, err)
	}

	var manifest maintained_apps.FMAManifestFile
	if err := json.Unmarshal(inputJson, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshaling app '%s' json manifest: %w", slug, err)
	}

	return &manifest, nil
}

func appFromJson(manifest *maintained_apps.FMAManifestFile) fleet.MaintainedApp {
	var app fleet.MaintainedApp
	app.Version = manifest.Versions[0].Version
	app.Platform = manifest.Versions[0].Platform()
	app.InstallerURL = manifest.Versions[0].InstallerURL
	app.SHA256 = manifest.Versions[0].SHA256
	app.InstallScript = manifest.Refs[manifest.Versions[0].InstallScriptRef]
	app.UninstallScript = manifest.Refs[manifest.Versions[0].UninstallScriptRef]
	app.AutomaticInstallQuery = manifest.Versions[0].Queries.Exists
	app.Categories = manifest.Versions[0].DefaultCategories

	return app
}

func DownloadMaintainedApp(cfg *Config, app fleet.MaintainedApp) (*fleet.TempFileReader, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	level.Info(cfg.logger).Log("msg", "Downloading...")
	installerTFR, filename, err := mdm_maintained_apps.DownloadInstaller(ctx, app.InstallerURL, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("downloading installer: %w", err)
	}

	// Create a file in tmpDir for the installer
	cleanFilename := filepath.Base(filename)
	if cleanFilename == "." || cleanFilename == ".." {
		cleanFilename = fmt.Sprintf("installer_%d", time.Now().UnixNano())
	}
	filePath := filepath.Join(cfg.tmpDir, cleanFilename)
	out, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	// Copy from TempFileReader to our file
	_, err = io.Copy(out, installerTFR)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Rewind the TempFileReader for future use
	err = installerTFR.Rewind()
	if err != nil {
		return nil, fmt.Errorf("rewinding temp file: %w", err)
	}

	cfg.env = os.Environ()
	installerPathEnv := fmt.Sprintf("INSTALLER_PATH=%s", filePath)
	cfg.env = append(cfg.env, installerPathEnv)

	return installerTFR, nil
}

func listDirectoryContents(dir string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}
	contents := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			contents[entry.Name()] = struct{}{}
		}
	}
	return contents, nil
}

var pathJoin = filepath.Join

func filepathJoin(parts ...string) string {
	return pathJoin(parts...)
}

func detectApplicationChange(installationSearchDirectory string, appListPre, appListPost map[string]struct{}) (string, bool) {
	// Check for added applications
	for app := range appListPost {
		if _, exists := appListPre[app]; !exists {
			return filepathJoin(installationSearchDirectory, app), true // true = added
		}
	}

	// Check for removed applications
	for app := range appListPre {
		if _, exists := appListPost[app]; !exists {
			return filepathJoin(installationSearchDirectory, app), false // false = removed
		}
	}

	return "", false // no change detected
}

func validateSqlInput(input string) error {
	// Allow alphanumeric, spaces, dots, hyphens, underscores, forward/back slashes, colons, parentheses
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s.\-_/\\:()]*$`, input); !matched {
		return fmt.Errorf("invalid characters in input: %s", input)
	}

	return nil
}
