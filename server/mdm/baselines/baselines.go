// Package baselines provides embedded security baseline definitions.
//
// A baseline is a curated bundle of SyncML profiles (enforcement),
// osquery policies (verification), and PowerShell scripts (remediation)
// that can be applied to a Fleet team using existing Fleet primitives.
package baselines

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"gopkg.in/yaml.v2"
)

//go:embed windows
var baselineFS embed.FS

// ListBaselines returns all available baseline manifests.
func ListBaselines() ([]fleet.BaselineManifest, error) {
	var baselines []fleet.BaselineManifest

	platforms, err := fs.ReadDir(baselineFS, ".")
	if err != nil {
		return nil, fmt.Errorf("reading baseline platforms: %w", err)
	}

	for _, platform := range platforms {
		if !platform.IsDir() {
			continue
		}
		entries, err := fs.ReadDir(baselineFS, platform.Name())
		if err != nil {
			return nil, fmt.Errorf("reading baselines for %s: %w", platform.Name(), err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			manifest, err := readManifest(path.Join(platform.Name(), entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading manifest for %s/%s: %w", platform.Name(), entry.Name(), err)
			}
			baselines = append(baselines, *manifest)
		}
	}

	return baselines, nil
}

// GetBaseline returns a single baseline manifest by ID.
func GetBaseline(id string) (*fleet.BaselineManifest, error) {
	all, err := ListBaselines()
	if err != nil {
		return nil, err
	}
	for _, b := range all {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("baseline %q not found", id)
}

// GetProfileContent returns the raw SyncML XML content for a profile within a baseline.
func GetProfileContent(baselineID, profilePath string) ([]byte, error) {
	dir, err := baselineDir(baselineID)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(baselineFS, path.Join(dir, profilePath))
}

// GetPolicyContent returns the raw YAML content for a policy definition within a baseline.
func GetPolicyContent(baselineID, policyPath string) ([]byte, error) {
	dir, err := baselineDir(baselineID)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(baselineFS, path.Join(dir, policyPath))
}

// GetScriptContent returns the raw script content within a baseline.
func GetScriptContent(baselineID, scriptPath string) ([]byte, error) {
	dir, err := baselineDir(baselineID)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(baselineFS, path.Join(dir, scriptPath))
}

// baselineDir finds the embedded directory for a baseline by its manifest ID.
func baselineDir(id string) (string, error) {
	platforms, err := fs.ReadDir(baselineFS, ".")
	if err != nil {
		return "", fmt.Errorf("reading baseline platforms: %w", err)
	}

	for _, platform := range platforms {
		if !platform.IsDir() {
			continue
		}
		entries, err := fs.ReadDir(baselineFS, platform.Name())
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := path.Join(platform.Name(), entry.Name())
			manifest, err := readManifest(dir)
			if err != nil {
				continue
			}
			if manifest.ID == id {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("baseline %q not found", id)
}

// readManifest reads and parses a manifest.yaml from the given directory.
func readManifest(dir string) (*fleet.BaselineManifest, error) {
	data, err := fs.ReadFile(baselineFS, path.Join(dir, "manifest.yaml"))
	if err != nil {
		return nil, fmt.Errorf("reading manifest.yaml: %w", err)
	}

	var manifest fleet.BaselineManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest.yaml: %w", err)
	}

	return &manifest, nil
}
