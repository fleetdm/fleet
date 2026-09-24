package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/mdmassets"
	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// MdmAssetsService runs the in-repo tools/mdm/assets exporter and persists saved
// export configs. A separate top-level tab drives it; it's independent of the
// active Fleet server (the export targets whatever MySQL the config points at).
// Thin adapter over internal/mdmassets.
type MdmAssetsService struct{}

// exportTimeout bounds the one-shot `go run ./tools/mdm/assets export` (allows
// for a first-run compile).
const exportTimeout = 5 * time.Minute

// readFileMax caps how much of an exported file ReadFile will return (asset
// files are tiny; this guards against accidentally slurping a huge path).
const readFileMax = 10 << 20 // 10 MiB

// MdmAssetsConfigsList returns all saved export configs.
func (s *MdmAssetsService) MdmAssetsConfigsList() ([]mdmassets.Config, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return nil, err
	}
	return mdmassets.List(dir)
}

// MdmAssetsConfigSave upserts a config (backend-stamped timestamps).
func (s *MdmAssetsService) MdmAssetsConfigSave(cfg mdmassets.Config) (mdmassets.Config, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return mdmassets.Config{}, err
	}
	return mdmassets.Save(dir, cfg, uint64(time.Now().UnixMilli()))
}

// MdmAssetsConfigDelete removes a config by id.
func (s *MdmAssetsService) MdmAssetsConfigDelete(id string) error {
	dir, err := paths.ConfigDir()
	if err != nil {
		return err
	}
	return mdmassets.Delete(dir, id)
}

// MdmAssetsDefaultDir is the default export destination (the primary repo root),
// or "" if server 1 has no repo configured yet.
func (s *MdmAssetsService) MdmAssetsDefaultDir() string {
	repo, err := primaryRepoPath()
	if err != nil {
		return ""
	}
	return repo
}

// MdmAssetsExport runs the exporter for a config and returns the captured
// output plus the files written.
func (s *MdmAssetsService) MdmAssetsExport(cfg mdmassets.Config) (mdmassets.ExportResult, error) {
	repo, err := primaryRepoPath()
	if err != nil {
		return mdmassets.ExportResult{}, err
	}
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		dir = repo
	}
	absDir, err := filepath.Abs(paths.Expand(dir))
	if err != nil {
		return mdmassets.ExportResult{}, fmt.Errorf("resolve export dir: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), exportTimeout)
	defer cancel()
	return mdmassets.Export(ctx, repo, cfg, absDir)
}

// MdmAssetsReadFile returns a file's contents (for the per-file copy button).
func (s *MdmAssetsService) MdmAssetsReadFile(path string) (string, error) {
	p := paths.Expand(path)
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", errors.New("path is a directory")
	}
	if st.Size() > readFileMax {
		return "", fmt.Errorf("file too large (%d bytes)", st.Size())
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// primaryRepoPath resolves server 1's repo path (where `go run ./tools/mdm/assets`
// executes).
func primaryRepoPath() (string, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return "", err
	}
	cur, err := settings.Load(dir)
	if err != nil {
		return "", err
	}
	if len(cur.Servers) == 0 || cur.Servers[0].WorktreePath == nil || strings.TrimSpace(*cur.Servers[0].WorktreePath) == "" {
		return "", errors.New("server 1 has no repo configured — set it in Settings first")
	}
	return *cur.Servers[0].WorktreePath, nil
}
