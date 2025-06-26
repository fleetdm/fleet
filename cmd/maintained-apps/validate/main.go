package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

	for _, app := range apps {
		if app.Platform != operatingSystem {
			continue
		}
		fmt.Print("Validating app: ", app.Name, " (", app.Slug, ")\n")
		appJson, err := getAppJson(app.Slug)
		if err != nil {
			fmt.Printf("Error getting app json manifest: %v\n", err)
			continue
		}

		maintainedApp := appFromJson(appJson)

		err = DownloadMaintainedApp(maintainedApp)
		if err != nil {
			fmt.Printf("Error downloading maintained app: %v\n", err)
			continue
		}

		appListPre, err := listDirectoryContents("/Applications")
		if err != nil {
			fmt.Printf("Error listing /Applications directory: %v\n", err)
		}

		fmt.Print("Executing install script...\n")
		err = executeScript(maintainedApp.InstallScript)
		if err != nil {
			fmt.Printf("Error executing install script: %v\n", err)
			continue
		}

		appListPost, err := listDirectoryContents("/Applications")
		if err != nil {
			fmt.Printf("Error listing /Applications directory: %v\n", err)
		}

		appPath := detectNewApplication(appListPre, appListPost)
		err = forceLaunchServicesRefresh(appPath)
		if err != nil {
			fmt.Printf("Error forcing LaunchServices refresh: %v. Attempting to continue.\n", err)
		}
		err = removeAppQuarentine(appPath)
		if err != nil {
			fmt.Printf("Error removing app quarantine: %v. Attempting to continue.\n", err)
		}

		existance, err := doesAppExists(app.Name, app.UniqueIdentifier, maintainedApp.Version)
		if err != nil {
			fmt.Printf("Error checking if app exists: %v\n", err)
			continue
		}
		if !existance {
			fmt.Printf("App version '%s' was not found by osquery\n", maintainedApp.Version)
			continue
		}

		fmt.Print("Executing uninstall script...\n")
		err = executeScript(maintainedApp.UninstallScript)
		if err != nil {
			fmt.Printf("Error uninstalling app: %v\n", err)
			continue
		}

		existance, err = doesAppExists(app.Name, app.UniqueIdentifier, maintainedApp.Version)
		if err != nil {
			fmt.Printf("Error checking if app exists after uninstall: %v\n", err)
			continue
		}
		if existance {
			fmt.Printf("App version '%s' was found after uninstall\n", maintainedApp.Version)
			continue
		}

		fmt.Print("All checks passed for app: ", app.Name, "\n")
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

func getFilename(resp *http.Response, url string) string {
	cd := resp.Header.Get("Content-Disposition")
	if cd != "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil && params["filename"] != "" {
			return params["filename"]
		}
	}
	// fallback: get the last part of the URL path
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func DownloadMaintainedApp(app fleet.MaintainedApp) error {
	// Similar to code in:
	// server/service/orbit_client.go:DownloadSoftwareInstallerFromURL
	// server/service/orbit_client.go:requestWithExternal
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, "GET", app.InstallerURL, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("GET %s: %w", app.InstallerURL, err)
	}
	defer response.Body.Close()
	// server/service/base_client.go:parseResponse
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned status %d", app.InstallerURL, response.StatusCode)
	}

	filePath := filepath.Join(tmpDir, getFilename(response, app.InstallerURL))
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	fmt.Print("Downloading...\n")
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	env = os.Environ()
	installerPathEnv := fmt.Sprintf("INSTALLER_PATH=%s", filePath)
	env = append(env, installerPathEnv)

	return nil
}

func executeScript(scriptContents string) error {
	// Similar code in:
	// orbit/pkg/installer/installer.go:runInstallerScript
	scriptExtension := ".sh"
	if runtime.GOOS == "windows" {
		scriptExtension = ".ps1"
	}

	scriptPath := filepath.Join(tmpDir, "script"+scriptExtension)
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("writing script: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, exitCode, err := scripts.ExecCmd(ctx, scriptPath, env)
	fmt.Print("--------------------\n")
	fmt.Printf("Script output:\n%s\n", string(output))
	fmt.Print("--------------------\n")
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("script execution failed with exit code %d: %s", exitCode, string(output))
	}
	return nil
}

func doesAppExists(appName, uniqueAppIdentifier, appVersion string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Looking for app: %s, version: %s\n", appName, appVersion)
	cmd := exec.CommandContext(ctx, "osqueryi", "--json", `
    SELECT name, path, bundle_short_version, bundle_version 
    FROM apps
    WHERE 
    bundle_identifier LIKE '%`+uniqueAppIdentifier+`%' OR
    name LIKE '%`+appName+`%'
  `)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("executing osquery command: %w", err)
	}

	type AppResult struct {
		Name           string `json:"name"`
		Path           string `json:"path"`
		Version        string `json:"bundle_short_version"`
		BundledVersion string `json:"bundle_version"`
	}
	var results []AppResult
	if err := json.Unmarshal(output, &results); err != nil {
		return false, fmt.Errorf("parsing osquery JSON output: %w", err)
	}

	if len(results) > 0 {
		for _, result := range results {
			fmt.Printf("Found app: '%s' at %s, Version: %s, Bundled Version: %s\n", result.Name, result.Path, result.Version, result.BundledVersion)
			if result.Version == appVersion || result.BundledVersion == appVersion {
				return true, nil
			}
		}
	}

	return false, nil
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

func detectNewApplication(appListPre, appListPost map[string]struct{}) string {
	for app := range appListPost {
		if _, exists := appListPre[app]; !exists {
			return path.Join("/Applications", app)
		}
	}
	return ""
}

func removeAppQuarentine(appPath string) error {
	if appPath == "" {
		return nil
	}
	fmt.Printf("Attempting to remove quarantine for: '%s'\n", appPath)
	cmd := exec.Command("xattr", "-p", "com.apple.quarantine", appPath)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("checking quarantine status: %v\n", err)
	}
	fmt.Printf("Quarantine status: %s\n", strings.TrimSpace(string(output)))
	cmd = exec.Command("spctl", "-a", "-v", appPath)
	output, err = cmd.Output()
	if err != nil {
		fmt.Printf("checking spctl status: %v\n", err)
	}
	fmt.Printf("spctl status: %s\n", strings.TrimSpace(string(output)))

	cmd = exec.Command("sudo", "spctl", "--add", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("adding app to quarantine exceptions: %w", err)
	}

	cmd = exec.Command("sudo", "xattr", "-r", "-d", "com.apple.quarantine", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("removing quarantine attribute: %w", err)
	}

	return nil
}

func forceLaunchServicesRefresh(appPath string) error {
	if appPath == "" {
		return nil
	}
	fmt.Printf("Forcing LaunchServices refresh for: '%s'\n", appPath)
	cmd := exec.Command("/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", "-f", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forcing LaunchServices refresh: %w", err)
	}
	time.Sleep(2 * time.Second)
	return nil
}
