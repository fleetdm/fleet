package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/version"
)

func (svc *Service) GetFleetdInstallerPkg(ctx context.Context, teamID uint) (*fleet.DownloadFleetdInstallerPayload, error) {
	// teamID == 0 means "No team" (unassigned hosts).
	var teamIDPtr *uint
	if teamID == 0 {
		// Global/no-team: authorize against enroll secrets (global scope).
		if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
			return nil, err
		}
	} else {
		if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
		teamIDPtr = &teamID
	}

	// Get the enroll secret. For teamID == 0, pass nil to get global (no-team) secrets.
	secrets, err := svc.ds.GetEnrollSecrets(ctx, teamIDPtr)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get enroll secrets")
	}
	if len(secrets) == 0 || secrets[0].Secret == "" {
		return nil, &fleet.BadRequestError{Message: "no enroll secret configured"}
	}
	enrollSecret := secrets[0].Secret

	// Get the server URL from AppConfig.
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}
	serverURL := appConfig.ServerSettings.ServerURL

	// Compute cache key: team ID + truncated secret hash + fleet version.
	secretHash := sha256.Sum256([]byte(enrollSecret))
	fleetVersion := version.Version().Version
	cacheKey := fmt.Sprintf("team-%d-%s-%s.pkg", teamID, hex.EncodeToString(secretHash[:8]), fleetVersion)

	// Acquire a per-team mutex so concurrent requests for the same team
	// don't build in parallel (BuildPkg is not thread-safe).
	mu := svc.getFleetdBuildMutex(teamID)
	mu.Lock()
	defer mu.Unlock()

	// Check cache.
	exists, err := svc.fleetdInstallerStore.Exists(ctx, cacheKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "check cached fleetd installer")
	}
	if exists {
		installer, size, err := svc.fleetdInstallerStore.Get(ctx, cacheKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get cached fleetd installer")
		}
		return &fleet.DownloadFleetdInstallerPayload{
			Filename:  "fleet-osquery.pkg",
			Installer: installer,
			Size:      size,
		}, nil
	}

	// Build the .pkg installer.
	tmpDir, err := os.MkdirTemp("", "fleetd-pkg-*")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create temp dir for fleetd build")
	}
	defer os.RemoveAll(tmpDir)

	outFile := filepath.Join(tmpDir, "fleet-osquery.pkg")
	opt := packaging.Options{
		FleetURL:            serverURL,
		EnrollSecret:        enrollSecret,
		Desktop:             true,
		EnableScripts:       true,
		StartService:        true,
		Identifier:          "com.fleetdm.orbit",
		CustomOutfile:       outFile,
		UpdateURL:           "https://updates.fleetdm.com",
		OrbitChannel:        "stable",
		DesktopChannel:      "stable",
		OsquerydChannel:     "stable",
		OrbitUpdateInterval: 15 * time.Minute,
	}

	if _, err := packaging.BuildPkg(opt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build fleetd pkg")
	}

	// Upload the built package to the cache store.
	f, err := os.Open(outFile)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "open built pkg")
	}
	defer f.Close()

	if err := svc.fleetdInstallerStore.Put(ctx, cacheKey, f); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "cache fleetd installer")
	}

	// Stream the cached file back to the caller.
	installer, size, err := svc.fleetdInstallerStore.Get(ctx, cacheKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get newly cached fleetd installer")
	}

	return &fleet.DownloadFleetdInstallerPayload{
		Filename:  "fleet-osquery.pkg",
		Installer: installer,
		Size:      size,
	}, nil
}

func (svc *Service) getFleetdBuildMutex(teamID uint) *sync.Mutex {
	mu, _ := svc.fleetdBuildMutexes.LoadOrStore(teamID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}
