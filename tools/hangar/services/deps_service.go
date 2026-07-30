package services

import "github.com/fleetdm/fleet/tools/hangar/internal/deps"

// DepsService exposes the first-run dependency checklist. Mirrors deps.rs.
type DepsService struct{}

func (s *DepsService) CheckDependencies(repoPath string, refreshPath bool) deps.DepReport {
	return deps.CheckDependencies(repoPath, refreshPath)
}
