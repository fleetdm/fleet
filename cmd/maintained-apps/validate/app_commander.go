package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type AppCommander struct {
	Name             string
	Slug             string
	UniqueIdentifier string
	Version          string
	AppPath          string
	MaintainedApp    fleet.MaintainedApp
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
	inputPath = filepath.Join(inputPath, parts[0]+".json")

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
		err = installerTFR.Rewind()
		if err != nil {
			return err
		}
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
	var preListError, postListError, listError error
	appListPre, err := listDirectoryContents(installationSearchDirectory)
	if err != nil {
		preListError = fmt.Errorf("Error listing %s directory: %v\n", installationSearchDirectory, err)
	}

	changerError := changer()

	appListPost, err := listDirectoryContents(installationSearchDirectory)
	if err != nil {
		postListError = fmt.Errorf("Error listing %s directory: %v\n", installationSearchDirectory, err)
	}

	appPath, changed := detectApplicationChange(installationSearchDirectory, appListPre, appListPost)
	if appPath == "" {
		fmt.Printf("Warning: no changes detected in %s directory after running application script.\n", installationSearchDirectory)
	} else {
		if changed {
			fmt.Printf("New application detected at: %s\n", appPath)
		} else {
			fmt.Printf("Application removal detected at: %s\n", appPath)
		}
	}

	switch {
	case preListError != nil && postListError != nil:
		listError = fmt.Errorf("app pre list: %v, app post list: %v", preListError, postListError)
	case preListError != nil:
		listError = fmt.Errorf("app pre list: %v", preListError)
	case postListError != nil:
		listError = fmt.Errorf("app post list: %v", postListError)
	}

	return appPath, changerError, listError
}
