package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/homebrew"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

// prepareHomebrewUpload fetches cask metadata from homebrew, downloads the installer,
// extracts the bundle identifier and icon from the .app bundle, and fills in the fields
// on payload that UploadSoftwareInstaller would otherwise resolve from the uploaded file.
//
// It is called at the top of UploadSoftwareInstaller when payload.FromHomebrew is set.
// The returned iconPNG (nil if extraction failed) is uploaded by the caller after the
// installer and software title have been persisted.
//
// The homebrew API is contacted exactly once via FetchCask. Install/uninstall scripts
// are built locally from the parsed cask — once up front for URL/SHA, then again after
// the bundle identifier is extracted from the downloaded .app bundle — so extracting
// the bundle ID does not trigger a second HTTP round-trip.
func (svc *Service) prepareHomebrewUpload(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) ([]byte, error) {
	ingester := homebrew.BrewIngester{
		BaseURL: homebrew.BaseBrewAPIURL,
		Logger:  slog.New(slog.DiscardHandler),
		Client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
	}
	input := homebrew.InputApp{Token: payload.FromHomebrew}

	cask, err := ingester.FetchCask(ctx, input)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching homebrew cask")
	}

	fma, err := ingester.BuildManifest(ctx, input, cask)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building homebrew manifest")
	}

	payload.URL = fma.InstallerURL
	payload.Version = fma.Version
	payload.Title = payload.FromHomebrew
	payload.Extension = strings.TrimPrefix(filepath.Ext(payload.URL), ".")
	payload.Platform = "darwin"
	payload.Source = "apps"

	client := fleethttp.NewClient(fleethttp.WithTimeout(maintained_apps.InstallerTimeout))
	installerTFR, filename, err := maintained_apps.DownloadInstaller(ctx, fma.InstallerURL, client)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "downloading homebrew installer")
	}

	payload.Filename = filename
	payload.InstallerFile = installerTFR
	payload.StorageID = fma.SHA256

	// Extract bundle identifier and icon from the .app bundle. Install/uninstall
	// scripts need the bundle ID so macOS can quit/relaunch the app cleanly.
	meta, err := file.ExtractInstallerMetadataWithHint(installerTFR, filename)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "extracting installer metadata")
	}
	if meta.BundleIdentifier != "" {
		payload.BundleIdentifier = meta.BundleIdentifier
	}
	if err := installerTFR.Rewind(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "rewind installer after metadata extraction")
	}

	// Rebuild the manifest (and its scripts) locally with the extracted bundle
	// identifier — no second HTTP call to the homebrew API.
	if payload.BundleIdentifier != "" {
		input.UniqueIdentifier = payload.BundleIdentifier
		fma, err = ingester.BuildManifest(ctx, input, cask)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "rebuilding homebrew manifest with extracted bundle identifier")
		}
	}

	payload.InstallScript = fma.InstallScript
	payload.UninstallScript = fma.UninstallScript

	return meta.IconPNG, nil
}

// uploadHomebrewIcon stores the icon extracted from the .app bundle against the
// newly-created software title. Icon failures are logged and swallowed — they
// shouldn't fail the whole upload.
func (svc *Service) uploadHomebrewIcon(ctx context.Context, titleID uint, teamID *uint, iconPNG []byte) {
	if len(iconPNG) == 0 {
		return
	}
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}
	iconHash := fmt.Sprintf("%x", sha256.Sum256(iconPNG))
	iconTFR, err := fleet.NewTempFileReader(bytes.NewReader(iconPNG), nil)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to build temp file for homebrew app icon", "err", err)
		return
	}
	defer iconTFR.Close()

	if err := svc.softwareTitleIconStore.Put(ctx, iconHash, iconTFR); err != nil {
		svc.logger.ErrorContext(ctx, "failed to store homebrew app icon", "err", err)
		return
	}
	iconPayload := &fleet.UploadSoftwareTitleIconPayload{
		TitleID:   titleID,
		TeamID:    tmID,
		Filename:  "icon.png",
		StorageID: iconHash,
		IconFile:  iconTFR,
	}
	if _, err := svc.ds.CreateOrUpdateSoftwareTitleIcon(ctx, iconPayload); err != nil {
		svc.logger.ErrorContext(ctx, "failed to create homebrew app icon record", "err", err)
	}
}
