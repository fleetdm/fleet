package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/scep"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// ScepService runs local SCEP servers (one shared in-repo binary, many
// depot-based profiles) for QA. A separate top-level tab drives it; it's
// independent of the active Fleet server. Thin adapter over internal/scep.
type ScepService struct {
	pm *processes.Manager
}

// NewScepService wires the service to the shared process manager (used to
// start/stop and stream logs for the SCEP servers).
func NewScepService(pm *processes.Manager) *ScepService {
	return &ScepService{pm: pm}
}

// buildTimeout bounds the one-shot `go build` of the scepserver binary.
const buildTimeout = 5 * time.Minute

// BinaryStatus reports the cached scepserver binary's presence + build time
// without building it.
func (s *ScepService) BinaryStatus() (scep.BinaryInfo, error) {
	dataDir, err := paths.DataDir()
	if err != nil {
		return scep.BinaryInfo{}, err
	}
	return scep.StatBinary(dataDir), nil
}

// EnsureBinary returns the cached binary, building it from the primary repo
// (server 1) on first use.
func (s *ScepService) EnsureBinary() (scep.BinaryInfo, error) {
	dataDir, err := paths.DataDir()
	if err != nil {
		return scep.BinaryInfo{}, err
	}
	if info := scep.StatBinary(dataDir); info.Exists {
		return info, nil
	}
	return s.buildBinary(dataDir)
}

// RebuildBinary force-rebuilds the cached binary from the primary repo (e.g.
// after pulling new code).
func (s *ScepService) RebuildBinary() (scep.BinaryInfo, error) {
	dataDir, err := paths.DataDir()
	if err != nil {
		return scep.BinaryInfo{}, err
	}
	return s.buildBinary(dataDir)
}

func (s *ScepService) buildBinary(dataDir string) (scep.BinaryInfo, error) {
	repo, err := s.primaryRepo()
	if err != nil {
		return scep.BinaryInfo{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
	return scep.BuildBinary(ctx, repo, dataDir)
}

// ResolveDepot returns the profile's depot directory (its explicit path, or the
// managed default under the depots dir).
func (s *ScepService) ResolveDepot(p settings.ScepProfile) (string, error) {
	dd, err := s.depotsDir()
	if err != nil {
		return "", err
	}
	return settings.ResolveDepotPath(dd, p), nil
}

// DepotInfo reads a depot's CA identity (thumbprint / issuer DN) from an
// explicit path.
func (s *ScepService) DepotInfo(depotPath string) scep.DepotInfo {
	return scep.ParseDepot(paths.Expand(depotPath))
}

// ProfileDepotInfo reads a profile's resolved depot CA identity in one call.
func (s *ScepService) ProfileDepotInfo(p settings.ScepProfile) (scep.DepotInfo, error) {
	depot, err := s.ResolveDepot(p)
	if err != nil {
		return scep.DepotInfo{}, err
	}
	return scep.ParseDepot(depot), nil
}

// InitCA creates a new CA in depotPath. Fails clearly if a CA already exists.
func (s *ScepService) InitCA(depotPath string, params scep.InitCAParams) (scep.DepotInfo, error) {
	if strings.TrimSpace(params.CommonName) == "" {
		return scep.DepotInfo{}, errors.New("common name is required")
	}
	depot := paths.Expand(depotPath)
	if strings.TrimSpace(depot) == "" {
		return scep.DepotInfo{}, errors.New("depot path is required")
	}
	if scep.ParseDepot(depot).Exists {
		return scep.DepotInfo{}, errors.New("a CA already exists in this depot; choose another depot or delete it first")
	}
	bin, err := s.EnsureBinary()
	if err != nil {
		return scep.DepotInfo{}, err
	}
	if err := os.MkdirAll(depot, 0o755); err != nil {
		return scep.DepotInfo{}, fmt.Errorf("create depot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := scep.InitCA(ctx, bin.Path, depot, params); err != nil {
		return scep.DepotInfo{}, err
	}
	return scep.ParseDepot(depot), nil
}

// StartProfile launches a profile's SCEP server: ensures the binary, resolves
// the depot (which must have a CA), and starts the process with logs streamed
// to the profile's channel. Concurrent — one process per profile.
func (s *ScepService) StartProfile(p settings.ScepProfile) error {
	depot, err := s.ResolveDepot(p)
	if err != nil {
		return err
	}
	di := scep.ParseDepot(depot)
	if !di.Exists {
		if di.Error != "" {
			return fmt.Errorf("depot problem: %s", di.Error)
		}
		return errors.New("this depot has no CA yet — run Init CA first")
	}
	bin, err := s.EnsureBinary()
	if err != nil {
		return err
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		return err
	}
	label := fmt.Sprintf("scepserver %s :%d", strings.TrimSpace(p.Name), p.Port)
	// Clear the previous run's log ring so the tail shows only the current
	// server, not a mix of dead-process attempts. Best-effort.
	_ = s.pm.ClearLogChannel(scep.LogChannel(p.ID))
	return s.pm.Start(scep.ProcID(p.ID), processes.StartArgs{
		Label:      label,
		Cwd:        dataDir,
		Program:    bin.Path,
		Args:       scep.ServeArgs(depot, p),
		LogChannel: scep.LogChannel(p.ID),
	})
}

// StopProfile stops a profile's running SCEP server.
func (s *ScepService) StopProfile(profileID string) error {
	return s.pm.Stop(scep.ProcID(profileID))
}

// LanIP returns the host's primary LAN IPv4 for building SCEP URLs.
func (s *ScepService) LanIP() string { return scep.LanIP() }

// primaryRepo resolves server 1's repo path (where the scepserver binary is
// built from). Shared with the MDM-assets service via primaryRepoPath.
func (s *ScepService) primaryRepo() (string, error) {
	return primaryRepoPath()
}

// depotsDir resolves where managed CA depots live: the settings override, or
// the default <data-dir>/scep-depots.
func (s *ScepService) depotsDir() (string, error) {
	cur, err := s.load()
	if err != nil {
		return "", err
	}
	if cur.ScepDepotsDir != nil && strings.TrimSpace(*cur.ScepDepotsDir) != "" {
		return paths.Expand(*cur.ScepDepotsDir), nil
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "scep-depots"), nil
}

func (s *ScepService) load() (settings.Settings, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return settings.Settings{}, err
	}
	return settings.Load(dir)
}
