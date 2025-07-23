package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

// memoized directory path
var (
	tmpDir                      string
	env                         []string
	installationSearchDirectory string
	operatingSystem             string
)

type AppCommander struct {
	Name             string
	Slug             string
	UniqueIdentifier string
	// computed fields
	Version string
	AppPath string

	MaintainedApp fleet.MaintainedApp
}

func (ac *AppCommander) isFrozen() (bool, error) {
	var inputPath string
	switch operatingSystem {
	case "darwin":
		inputPath = "ee/maintained-apps/inputs/homebrew"
	case "windows":
		inputPath = "ee\\maintained-apps\\inputs\\winget"
	}
	parts := strings.Split(ac.Slug, "/")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid slug format: %s, expected <name>/<platform>", ac.Slug)
	}
	inputPath = path.Join(inputPath, parts[0]+".json")

	fileBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return false, fmt.Errorf("reading app input file: %w", err)
	}

	var input struct {
		Frozen bool `json:"frozen"`
	}
	if err := json.Unmarshal(fileBytes, &input); err != nil {
		return false, fmt.Errorf("unmarshal app input file: %w", err)
	}

	return input.Frozen, nil
}

func (ac *AppCommander) extractAppVersion(installerTFR *fleet.TempFileReader) error {
	// default to maintained app version
	ac.Version = ac.MaintainedApp.Version

	if ac.Version == "latest" {
		meta, err := file.ExtractInstallerMetadata(installerTFR)
		if err != nil {
			return err
		}
		ac.Version = meta.Version
		installerTFR.Rewind()
	}

	return nil
}

func (ac *AppCommander) uninstallPreInstalled(installationSearchDirectory string) {
	fmt.Printf("App '%s' is marked as pre-installed, attempting to run uninstall script.\n", ac.Name)

	_, _, listError := ac.expectToChangeFileSystem(
		func() error {
			uninstalled := ac.uninstallApp()
			if !uninstalled {
				fmt.Printf("Failed to uninstall pre-installed app '%s'", ac.Name)
			}
			return nil
		},
	)

	if listError != nil {
		fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, listError)
	}

	// appListPre, err := listDirectoryContents(installationSearchDirectory)
	// if err != nil {
	// 	fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, err)
	// }

	// uninstalled := ac.uninstallApp()
	// if !uninstalled {
	// 	fmt.Printf("Failed to uninstall pre-installed app '%s'", ac.Name)
	// }

	// appListPost, err := listDirectoryContents(installationSearchDirectory)
	// if err != nil {
	// 	fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, err)
	// }

	// ac.AppPath = detectRemovedApplication(installationSearchDirectory, appListPre, appListPost)
	// if ac.AppPath == "" {
	// 	fmt.Printf("Warning: no changes found in %s directory after running application uninstall script.\n", installationSearchDirectory)
	// } else {
	// 	fmt.Printf("Removed application detected at: %s\n", ac.AppPath)
	// }
}

func (ac *AppCommander) uninstallApp() bool {
	fmt.Print("Executing uninstall script...\n")
	output, err := executeScript(ac.MaintainedApp.UninstallScript)
	if err != nil {
		fmt.Printf("Error uninstalling app: %v\n", err)
		fmt.Printf("Output: %s\n", output)
		return false
	}

	existance, err := doesAppExists(ac.Name, ac.UniqueIdentifier, ac.Version, ac.AppPath)
	if err != nil {
		fmt.Printf("Error checking if app exists after uninstall: %v\n", err)
		return false
	}
	if existance {
		fmt.Printf("App version '%s' was found after uninstall\n", ac.Version)
		return false
	}

	return true
}

func (ac *AppCommander) expectToChangeFileSystem(changer func() error) (string, error, error) {
	var listError error
	appListPre, err := listDirectoryContents(installationSearchDirectory)
	if err != nil {
		listError = fmt.Errorf("Error listing %s directory: %v\n", installationSearchDirectory, err)
	}

	changerError := changer()

	appListPost, err := listDirectoryContents(installationSearchDirectory)
	if err != nil {
		listError = fmt.Errorf("Error listing %s directory: %v\n", installationSearchDirectory, err)
	}

	appPath, changed := detectApplicationChange(installationSearchDirectory, appListPre, appListPost)
	if appPath == "" {
		fmt.Printf("Warning: no changes detected in %s directory after running application script.\n", installationSearchDirectory)
	} else {
		if changed {
			fmt.Printf("New application detected at: %s\n", appPath)
		} else {
			// If changed is false, it means an application was removed
			fmt.Printf("Application removal detected at: %s\n", appPath)
		}
	}

	return appPath, changerError, listError
}

func main() {
	operatingSystem = strings.ToLower(os.Getenv("GOOS"))
	if operatingSystem == "" {
		operatingSystem = runtime.GOOS
		fmt.Printf("GOOS environment variable is not set. Using system detected: '%s'\n", operatingSystem)
	}
	if operatingSystem != "darwin" && operatingSystem != "windows" {
		fmt.Printf("Unsupported operating system: %s\n", operatingSystem)
		os.Exit(1)
	}
	installationSearchDirectory = os.Getenv("INSTALLATION_SEARCH_DIRECTORY")
	if installationSearchDirectory == "" {
		switch operatingSystem {
		case "darwin":
			installationSearchDirectory = "/Applications"
		case "windows":
			installationSearchDirectory = "C:\\Program Files"
		}
		fmt.Printf("INSTALLATION_SEARCH_DIRECTORY environment variable is not set. Using default: '%s'\n", installationSearchDirectory)
	}

	apps, err := getListOfApps()
	if err != nil {
		fmt.Printf("Error getting list of apps: %v\n", err)
		os.Exit(1)
	}

	// Create a temporary directory to store downloaded apps
	tmpDir, err = os.MkdirTemp("", "fma-validate-")
	if err != nil {
		fmt.Printf("Error creating temporary directory: %v\n", err)
		os.Exit(1)
	}
	defer func() error {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			return fmt.Errorf("removing temporary directory: %w", err)
		}
		return nil
	}()

	totalApps := 0
	successfulApps := 0
	appWithError := []string{}
	appWithWarning := []string{}
	frozenApps := []string{}
	for _, app := range apps {
		if app.Platform != operatingSystem {
			continue
		}
		// Name             string `json:"name"`
		// Slug             string `json:"slug"`
		// Platform         string `json:"platform"`
		// UniqueIdentifier string `json:"unique_identifier"`
		// Description      string `json:"description"`

		totalApps++

		fmt.Print("\n\nValidating app: ", app.Name, " (", app.Slug, ")\n")
		ac := &AppCommander{}

		appJson, err := getAppJson(app.Slug)
		if err != nil {
			fmt.Printf("Error getting app json manifest: %v\n", err)
			appWithError = append(appWithError, app.Name)
			continue
		}

		maintainedApp := appFromJson(appJson)
		// app.Version = manifest.Versions[0].Version
		// app.Platform = manifest.Versions[0].Platform()
		// app.InstallerURL = manifest.Versions[0].InstallerURL
		// app.SHA256 = manifest.Versions[0].SHA256
		// app.InstallScript = manifest.Refs[manifest.Versions[0].InstallScriptRef]
		// app.UninstallScript = manifest.Refs[manifest.Versions[0].UninstallScriptRef]
		// app.AutomaticInstallQuery = manifest.Versions[0].Queries.Exists
		// app.Categories = manifest.Versions[0].DefaultCategories
		ac.Name = app.Name
		ac.Slug = app.Slug
		ac.UniqueIdentifier = app.UniqueIdentifier
		ac.MaintainedApp = maintainedApp

		isFrozen, err := ac.isFrozen()
		if err != nil {
			fmt.Printf("Error checking if app is frozen: %v\n", err)
			appWithError = append(appWithError, ac.Name)
			continue
		}
		if isFrozen {
			fmt.Printf("App is frozen, skipping validation...\n")
			frozenApps = append(frozenApps, ac.Name)
			continue
		}

		installerTFR, err := DownloadMaintainedApp(maintainedApp)
		if err != nil {
			fmt.Printf("Error downloading maintained app: %v\n", err)
			appWithError = append(appWithError, ac.Name)
			continue
		}
		defer installerTFR.Close()

		err = ac.extractAppVersion(installerTFR)
		if err != nil {
			fmt.Printf("Error extracting installer version: %v. Using '%s'\n", err, ac.Version)
			appWithError = append(appWithError, ac.Name)
		}

		// If application is already installed, attempt to uninstall it
		if slices.Contains(preInstalled, ac.Slug) {
			ac.uninstallPreInstalled(installationSearchDirectory)
		}

		appPath, changerError, listError := ac.expectToChangeFileSystem(
			func() error {
				fmt.Print("Executing install script...\n")
				output, err := executeScript(ac.MaintainedApp.InstallScript)
				if err != nil {
					fmt.Printf("Error executing install script: %v\n", err)
					fmt.Printf("Output: %s\n", output)
					return err
				}
				return nil
			},
		)
		if listError != nil {
			fmt.Printf("error listing directory contents: %v", listError)
		}
		if changerError != nil {
			appWithError = append(appWithError, ac.Name)
			continue
		}
		ac.AppPath = appPath
		if ac.AppPath == "" {
			appWithWarning = append(appWithWarning, ac.Name)
		}

		// Install
		// appListPre, err := listDirectoryContents(installationSearchDirectory)
		// if err != nil {
		// 	fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, err)
		// }

		// fmt.Print("Executing install script...\n")
		// output, err := executeScript(ac.MaintainedApp.InstallScript)
		// if err != nil {
		// 	fmt.Printf("Error executing install script: %v\n", err)
		// 	fmt.Printf("Output: %s\n", output)
		// 	appWithError = append(appWithError, ac.Name)
		// 	continue
		// }

		// appListPost, err := listDirectoryContents(installationSearchDirectory)
		// if err != nil {
		// 	fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, err)
		// }

		// ac.AppPath = detectNewApplication(installationSearchDirectory, appListPre, appListPost)
		// if ac.AppPath == "" {
		// 	fmt.Printf("Warning: no new application detected in %s directory after installation.\n", installationSearchDirectory)
		// 	appWithWarning = append(appWithWarning, ac.Name)
		// } else {
		// 	fmt.Printf("New application detected at: %s\n", ac.AppPath)
		// }

		err = postApplicationInstall(ac.AppPath)
		if err != nil {
			fmt.Printf("Warning: Error detected in post-installation steps: %v\n", err)
			appWithWarning = append(appWithWarning, ac.Name)
		}

		existance, err := doesAppExists(ac.Name, ac.UniqueIdentifier, ac.Version, ac.AppPath)
		if err != nil {
			fmt.Printf("Error checking if app exists: %v\n", err)
			appWithError = append(appWithError, ac.Name)
			continue
		}
		if !existance {
			fmt.Printf("App version '%s' was not found by osquery\n", ac.Version)
			appWithError = append(appWithError, ac.Name)
			continue
		}

		// Uninstall
		uninstalled := ac.uninstallApp()
		if !uninstalled {
			appWithError = append(appWithError, ac.Name)
			continue
		}

		fmt.Print("All checks passed for app: ", ac.Name)
		successfulApps++
	}

	if successfulApps == totalApps-len(frozenApps) {
		fmt.Printf("\nAll %d apps were successfully validated.\n", totalApps)
		if len(appWithWarning) > 0 {
			fmt.Printf("Some apps were validated with warnings: %v\n", appWithWarning)
		}
		if len(frozenApps) > 0 {
			fmt.Printf("Some apps were frozen and skipped validation: %v\n", frozenApps)
		}
		os.Exit(0)
	} else {
		fmt.Printf("\nValidated %d out of %d apps successfully.\n", successfulApps, totalApps)
		if len(appWithWarning) > 0 {
			fmt.Printf("Some apps were validated with warnings: %v\n", appWithWarning)
		}
		if len(frozenApps) > 0 {
			fmt.Printf("Some apps were frozen and skipped validation: %v\n", frozenApps)
		}
		fmt.Printf("Apps with errors: %v\n", appWithError)
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
		return nil, fmt.Errorf("unmarshaling app '%s' json manifest: %w", err)
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

func DownloadMaintainedApp(app fleet.MaintainedApp) (*fleet.TempFileReader, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Print("Downloading...\n")
	installerTFR, filename, err := mdm_maintained_apps.DownloadInstaller(ctx, app.InstallerURL, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("downloading installer: %w", err)
	}

	// Create a file in tmpDir for the installer
	filePath := filepath.Join(tmpDir, filename)
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

	env = os.Environ()
	installerPathEnv := fmt.Sprintf("INSTALLER_PATH=%s", filePath)
	env = append(env, installerPathEnv)

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

func detectApplicationChange(installationSearchDirectory string, appListPre, appListPost map[string]struct{}) (string, bool) {
	// Check for added applications
	for app := range appListPost {
		if _, exists := appListPre[app]; !exists {
			return filepath.Join(installationSearchDirectory, app), true // true = added
		}
	}

	// Check for removed applications
	for app := range appListPre {
		if _, exists := appListPost[app]; !exists {
			return filepath.Join(installationSearchDirectory, app), false // false = removed
		}
	}

	return "", false // no change detected
}
