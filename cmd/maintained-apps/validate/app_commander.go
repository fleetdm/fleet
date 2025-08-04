//go:build darwin || windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type AppCommander struct {
	Name             string
	Slug             string
	UniqueIdentifier string
	Version          string
	AppPath          string
	UninstallScript  string
	InstallScript    string
	cfg              *Config
	appLogger        kitlog.Logger
}

func (ac *AppCommander) isFrozen() (bool, error) {
	var inputPath string
	switch ac.cfg.operatingSystem {
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

func (ac *AppCommander) uninstallPreInstalled(ctx context.Context) {
	level.Info(ac.appLogger).Log("msg", "App is marked as pre-installed, attempting to run uninstall script...")

	_, _, listError := ac.expectToChangeFileSystem(
		func() error {
			uninstalled := ac.uninstallApp(ctx)
			if !uninstalled {
				level.Error(ac.appLogger).Log("msg", "Failed to uninstall pre-installed app")
			}
			return nil
		},
	)

	if listError != nil {
		level.Error(ac.appLogger).Log("msg", fmt.Sprintf("Error listing %s directory: %v", ac.cfg.installationSearchDirectory, listError))
	}
}

func (ac *AppCommander) uninstallApp(ctx context.Context) bool {
	level.Info(ac.appLogger).Log("msg", "Executing uninstall script for app...")
	output, err := executeScript(ac.cfg, ac.UninstallScript)
	if err != nil {
		level.Error(ac.appLogger).Log("msg", fmt.Sprintf("Error uninstalling app: %v", err))
		level.Error(ac.appLogger).Log("msg", fmt.Sprintf("Output: %s", output))
		return false
	}
	level.Debug(ac.appLogger).Log("msg", fmt.Sprintf("Output: %s", output))

	existance, err := appExists(ctx, ac.appLogger, ac.Name, ac.UniqueIdentifier, ac.Version, ac.AppPath)
	if err != nil {
		level.Error(ac.appLogger).Log("msg", fmt.Sprintf("Error checking if app exists after uninstall: %v", err))
		return false
	}
	if existance {
		level.Error(ac.appLogger).Log("msg", fmt.Sprintf("App version '%s' was found after uninstall", ac.Version))
		return false
	}

	return true
}

func (ac *AppCommander) expectToChangeFileSystem(changer func() error) (string, error, error) {
	var preListError, postListError, listError error
	appListPre, err := listDirectoryContents(ac.cfg.installationSearchDirectory)
	if err != nil {
		preListError = fmt.Errorf("Error listing %s directory: %v", ac.cfg.installationSearchDirectory, err)
	}

	changerError := changer()

	appListPost, err := listDirectoryContents(ac.cfg.installationSearchDirectory)
	if err != nil {
		postListError = fmt.Errorf("Error listing %s directory: %v", ac.cfg.installationSearchDirectory, err)
	}

	appPath, changed := detectApplicationChange(ac.cfg.installationSearchDirectory, appListPre, appListPost)
	if appPath == "" {
		level.Warn(ac.appLogger).Log("msg", fmt.Sprintf("no changes detected in %s directory after running application script.", ac.cfg.installationSearchDirectory))
	} else {
		if changed {
			level.Info(ac.appLogger).Log("msg", fmt.Sprintf("New application detected at: %s", appPath))
		} else {
			level.Info(ac.appLogger).Log("msg", fmt.Sprintf("Application removal detected at: %s", appPath))
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
