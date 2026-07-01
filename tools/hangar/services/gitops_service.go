package services

import "github.com/fleetdm/fleet/tools/hangar/internal/gitops"

// GitopsService exposes GitOps repo discovery + target checks. Mirrors gitops.rs.
type GitopsService struct{}

func (s *GitopsService) GitopsListRepos(dir string) (gitops.DirScan, error) {
	return gitops.ListRepos(dir)
}
func (s *GitopsService) GitopsCheckTarget(dir, name string) (gitops.TargetCheck, error) {
	return gitops.CheckTarget(dir, name)
}
