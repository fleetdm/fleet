//go:build darwin || windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/winget/external_refs"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
	appLogger        *slog.Logger
}

func (ac *AppCommander) isFrozen() (bool, error) {
	inputPath := ac.cfg.inputsPath
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
		var version string
		var err error

		// Adobe Acrobat requires special handling for ZIP-packaged installers
		if ac.Slug == "adobe-acrobat-pro/windows" {
			version, err = externalrefs.ExtractVersionFromAdobeZIP(installerTFR)
			if err != nil {
				return fmt.Errorf("extract adobe zip version: %w", err)
			}
		} else {
			// Standard metadata extraction for other apps
			meta, err := file.ExtractInstallerMetadata(installerTFR)
			if err != nil {
				return err
			}
			version = meta.Version
			err = installerTFR.Rewind()
			if err != nil {
				return err
			}
		}

		ac.Version = version
	}

	return nil
}

func (ac *AppCommander) uninstallPreInstalled(ctx context.Context) {
	ac.appLogger.InfoContext(ctx, "App is marked as pre-installed, attempting to run uninstall script...")

	_, _, listError := ac.expectToChangeFileSystem(ctx,
		func() error {
			uninstalled := ac.uninstallApp(ctx)
			if !uninstalled {
				ac.appLogger.ErrorContext(ctx, "Failed to uninstall pre-installed app")
			}
			return nil
		},
	)

	if listError != nil {
		ac.appLogger.ErrorContext(ctx, fmt.Sprintf("Error listing %s directory: %v", ac.cfg.installationSearchDirectory, listError))
	}
}

func (ac *AppCommander) uninstallApp(ctx context.Context) bool {
	ac.appLogger.InfoContext(ctx, "Executing uninstall script for app...")
	output, err := executeScript(ac.cfg, ac.UninstallScript)
	if err != nil {
		ac.appLogger.ErrorContext(ctx, fmt.Sprintf("Error uninstalling app: %v", err))
		ac.appLogger.ErrorContext(ctx, fmt.Sprintf("Output: %s", output))
		return false
	}
	ac.appLogger.DebugContext(ctx, fmt.Sprintf("Output: %s", output))

	existance, err := appExists(ctx, ac.appLogger, ac.Name, ac.UniqueIdentifier, ac.Version, ac.AppPath)
	if err != nil {
		ac.appLogger.ErrorContext(ctx, fmt.Sprintf("Error checking if app exists after uninstall: %v", err))
		return false
	}
	if existance {
		ac.appLogger.ErrorContext(ctx, fmt.Sprintf("App version '%s' was found after uninstall", ac.Version))
		return false
	}

	return true
}

func (ac *AppCommander) expectToChangeFileSystem(ctx context.Context, changer func() error) (string, error, error) {
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
		ac.appLogger.WarnContext(ctx, fmt.Sprintf("no changes detected in %s directory after running application script.", ac.cfg.installationSearchDirectory))
	} else {
		if changed {
			ac.appLogger.InfoContext(ctx, fmt.Sprintf("New application detected at: %s", appPath))
		} else {
			ac.appLogger.InfoContext(ctx, fmt.Sprintf("Application removal detected at: %s", appPath))
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
