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
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

// memoized directory path
var (
	tmpDir string
	env    []string
)

func main() {
	operatingSystem := strings.ToLower(os.Getenv("GOOS"))
	if operatingSystem == "" {
		operatingSystem = runtime.GOOS
		fmt.Printf("GOOS environment variable is not set. Using system detected: '%s'\n", operatingSystem)
	}
	if operatingSystem != "darwin" && operatingSystem != "windows" {
		fmt.Printf("Unsupported operating system: %s\n", operatingSystem)
		os.Exit(1)
	}
	installationSearchDirectory := os.Getenv("INSTALLATION_SEARCH_DIRECTORY")
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
	for _, app := range apps {
		if app.Platform != operatingSystem {
			continue
		}
		totalApps++
		fmt.Print("\n\nValidating app: ", app.Name, " (", app.Slug, ")\n")
		appJson, err := getAppJson(app.Slug)
		if err != nil {
			fmt.Printf("Error getting app json manifest: %v\n", err)
			appWithError = append(appWithError, app.Name)
			continue
		}

		maintainedApp := appFromJson(appJson)

		installerTFR, err := DownloadMaintainedApp(maintainedApp)
		if err != nil {
			fmt.Printf("Error downloading maintained app: %v\n", err)
			appWithError = append(appWithError, app.Name)
			continue
		}
		defer installerTFR.Close()

		appListPre, err := listDirectoryContents(installationSearchDirectory)
		if err != nil {
			fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, err)
		}

		fmt.Print("Executing install script...\n")
		output, err := executeScript(maintainedApp.InstallScript)
		if err != nil {
			fmt.Printf("Error executing install script: %v\n", err)
			fmt.Printf("Output: %s\n", output)
			appWithError = append(appWithError, app.Name)
			continue
		}

		appListPost, err := listDirectoryContents(installationSearchDirectory)
		if err != nil {
			fmt.Printf("Error listing %s directory: %v\n", installationSearchDirectory, err)
		}

		appPath := detectNewApplication(installationSearchDirectory, appListPre, appListPost)
		if appPath == "" {
			fmt.Printf("Warning: no new application detected in %s directory after installation.\n", installationSearchDirectory)
			appWithWarning = append(appWithWarning, app.Name)
		} else {
			fmt.Printf("New application detected at: %s\n", appPath)
		}

		err = postApplicationInstall(appPath)
		if err != nil {
			fmt.Printf("Warning: Error detected in post-installation steps: %v\n", err)
			appWithWarning = append(appWithWarning, app.Name)
		}

		appVersion, err := extractAppVersion(maintainedApp, installerTFR)
		if err != nil {
			fmt.Printf("Error extracting installer version: %v. Using '%s'\n", err, appVersion)
			appWithError = append(appWithError, app.Name)
		}

		existance, err := doesAppExists(app.Name, app.UniqueIdentifier, appVersion, appPath)
		if err != nil {
			fmt.Printf("Error checking if app exists: %v\n", err)
			appWithError = append(appWithError, app.Name)
			continue
		}
		if !existance {
			fmt.Printf("App version '%s' was not found by osquery\n", maintainedApp.Version)
			appWithError = append(appWithError, app.Name)
			continue
		}

		fmt.Print("Executing uninstall script...\n")
		output, err = executeScript(maintainedApp.UninstallScript)
		if err != nil {
			fmt.Printf("Error uninstalling app: %v\n", err)
			fmt.Printf("Output: %s\n", output)
			appWithError = append(appWithError, app.Name)
			continue
		}

		existance, err = doesAppExists(app.Name, app.UniqueIdentifier, appVersion, appPath)
		if err != nil {
			fmt.Printf("Error checking if app exists after uninstall: %v\n", err)
			appWithError = append(appWithError, app.Name)
			continue
		}
		if existance {
			fmt.Printf("App version '%s' was found after uninstall\n", maintainedApp.Version)
			appWithError = append(appWithError, app.Name)
			continue
		}

		fmt.Print("All checks passed for app: ", app.Name)
		successfulApps++
	}

	if successfulApps == totalApps {
		fmt.Printf("\nAll %d apps were successfully validated.\n", totalApps)
		if len(appWithWarning) > 0 {
			fmt.Printf("Some apps were validated with warnings: %v\n", appWithWarning)
		}
		os.Exit(0)
	} else {
		fmt.Printf("\nValidated %d out of %d apps successfully.\n", successfulApps, totalApps)
		if len(appWithWarning) > 0 {
			fmt.Printf("Some apps were validated with warnings: %v\n", appWithWarning)
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

func extractAppVersion(maintainedApp fleet.MaintainedApp, installerTFR *fleet.TempFileReader) (string, error) {
	appVersion := maintainedApp.Version

	if appVersion == "latest" {
		meta, err := file.ExtractInstallerMetadata(installerTFR)
		if err != nil {
			return appVersion, err
		}
		appVersion = meta.Version
	}

	return appVersion, nil
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

func executeScript(scriptContents string) (string, error) {
	// Similar code in:
	// orbit/pkg/installer/installer.go:runInstallerScript
	scriptExtension := ".sh"
	if runtime.GOOS == "windows" {
		scriptExtension = ".ps1"
	}

	scriptPath := filepath.Join(tmpDir, "script"+scriptExtension)
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing script: %w", err)
	}

	timeout := 5 * time.Minute
	if runtime.GOOS == "windows" {
		timeout = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, exitCode, err := scripts.ExecCmd(ctx, scriptPath, env)
	result := fmt.Sprintf(`
--------------------
%s
--------------------
`, string(output))

	if err != nil {
		return result, err
	}
	if exitCode != 0 {
		return result, fmt.Errorf("script execution failed with exit code %d: %s", exitCode, string(output))
	}
	return result, nil
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

func detectNewApplication(installationSearchDirectory string, appListPre, appListPost map[string]struct{}) string {
	for app := range appListPost {
		if _, exists := appListPre[app]; !exists {
			return filepath.Join(installationSearchDirectory, app)
		}
	}
	return ""
}
